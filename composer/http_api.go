package composer

import (
	"clash-composer/config"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultServeAddr = "127.0.0.1:8080"
	defaultGroupName = "default"
)

type ServeOptions struct {
	Addr      string
	ConfigDir string
	Token     string
}

type httpAPI struct {
	configDir string
	token     string
}

type apiError struct {
	Status  int    `json:"-"`
	Message string `json:"error"`
}

func (e *apiError) Error() string {
	return e.Message
}

func ServeHTTPAPI(options ServeOptions) error {
	handler, addr, err := newHTTPAPI(options)
	if err != nil {
		return err
	}

	log.Printf("http api listen start: addr=%s config_dir=%s", addr, handler.configDir)
	return http.ListenAndServe(addr, handler)
}

func newHTTPAPI(options ServeOptions) (*httpAPI, string, error) {
	addr := options.Addr
	if addr == "" {
		addr = defaultServeAddr
	}

	token := options.Token
	if token == "" {
		token = os.Getenv("CLASH_COMPOSER_TOKEN")
	}
	if token == "" {
		return nil, "", fmt.Errorf("serve token is required: pass -token or set CLASH_COMPOSER_TOKEN")
	}

	if options.ConfigDir == "" {
		return nil, "", fmt.Errorf("config dir is required")
	}
	configDir, err := filepath.Abs(options.ConfigDir)
	if err != nil {
		return nil, "", fmt.Errorf("resolve config dir: %w", err)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, "", fmt.Errorf("create config dir: %w", err)
	}
	realConfigDir, err := filepath.EvalSymlinks(configDir)
	if err != nil {
		return nil, "", fmt.Errorf("resolve config dir symlinks: %w", err)
	}

	return &httpAPI{
		configDir: realConfigDir,
		token:     token,
	}, addr, nil
}

func (api *httpAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts, err := splitRequestPath(r.URL.Path)
	if err != nil {
		writeAPIError(w, badRequest(err.Error()))
		return
	}

	if len(parts) == 0 {
		writeAPIError(w, notFound("not found"))
		return
	}

	if parts[0] == "subscriptions" {
		api.handleSubscription(w, r, parts)
		return
	}

	if !api.authorizeBearer(r) {
		writeAPIError(w, &apiError{Status: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	if parts[0] == "configs" {
		api.handleConfigs(w, r, parts)
		return
	}

	writeAPIError(w, notFound("not found"))
}

func (api *httpAPI) handleConfigs(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeAPIError(w, methodNotAllowed("method not allowed"))
			return
		}
		api.handleListConfigs(w, r)
		return
	}

	if len(parts) < 2 {
		writeAPIError(w, notFound("not found"))
		return
	}

	id := parts[1]
	if err := validateConfigID(id); err != nil {
		writeAPIError(w, badRequest(err.Error()))
		return
	}

	if len(parts) == 2 {
		api.handleConfig(w, r, id)
		return
	}

	if len(parts) >= 4 && parts[2] == "template" {
		switch parts[3] {
		case "rule-providers":
			api.handleRuleProviders(w, r, id, parts[4:])
			return
		case "rule-groups":
			api.handleRuleGroups(w, r, id, parts[4:])
			return
		}
	}

	writeAPIError(w, notFound("not found"))
}

func (api *httpAPI) handleListConfigs(w http.ResponseWriter, _ *http.Request) {
	entries, err := os.ReadDir(api.configDir)
	if err != nil {
		writeAPIError(w, internalError("list configs", err))
		return
	}

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if validateConfigID(id) == nil {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	writeJSON(w, http.StatusOK, map[string]any{"configs": ids})
}

func (api *httpAPI) handleConfig(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		rule, err := api.readMergeRule(id)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rule)
	case http.MethodPost:
		rule, err := decodeMergeRuleBody(r)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		if err := api.writeMergeRule(id, rule, true); err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, rule)
	case http.MethodPut:
		rule, err := decodeMergeRuleBody(r)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		if err := api.writeMergeRule(id, rule, false); err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rule)
	case http.MethodDelete:
		if err := api.deleteMergeRule(id); err != nil {
			writeAPIError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeAPIError(w, methodNotAllowed("method not allowed"))
	}
}

func (api *httpAPI) handleSubscription(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Method != http.MethodGet {
		writeAPIError(w, methodNotAllowed("method not allowed"))
		return
	}
	if !api.authorizeQueryToken(r) {
		writeAPIError(w, &apiError{Status: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}
	if len(parts) != 2 || !strings.HasSuffix(parts[1], ".yaml") {
		writeAPIError(w, notFound("not found"))
		return
	}

	id := strings.TrimSuffix(parts[1], ".yaml")
	if err := validateConfigID(id); err != nil {
		writeAPIError(w, badRequest(err.Error()))
		return
	}

	rule, apiErr := api.readMergeRule(id)
	if apiErr != nil {
		writeAPIError(w, apiErr)
		return
	}
	mergedRule, err := api.resolveMergeRule(rule)
	if err != nil {
		writeAPIError(w, badRequest(err.Error()))
		return
	}

	merged, mergeErr := Merge(mergedRule)
	if mergeErr != nil {
		writeAPIError(w, internalError("merge config", mergeErr))
		return
	}

	data, err := config.MarshalRawConfig(merged)
	if err != nil {
		writeAPIError(w, internalError("marshal merged config", err))
		return
	}

	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Printf("write subscription response failed: %v", err)
	}
}

func (api *httpAPI) readMergeRule(id string) (MergeRule, *apiError) {
	path := api.configPath(id)
	if err := ensureRegularFile(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MergeRule{}, notFound("config not found")
		}
		return MergeRule{}, internalError("stat config", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MergeRule{}, notFound("config not found")
		}
		return MergeRule{}, internalError("read config", err)
	}

	var rule MergeRule
	if err := json.Unmarshal(data, &rule); err != nil {
		return MergeRule{}, badRequest(fmt.Sprintf("parse config: %v", err))
	}
	if err := api.validateMergeRule(rule); err != nil {
		return MergeRule{}, badRequest(err.Error())
	}
	return rule, nil
}

func (api *httpAPI) writeMergeRule(id string, rule MergeRule, create bool) *apiError {
	if err := api.validateMergeRule(rule); err != nil {
		return badRequest(err.Error())
	}

	path := api.configPath(id)
	_, err := os.Lstat(path)
	if create && err == nil {
		return conflict("config already exists")
	}
	if create && err != nil && !errors.Is(err, os.ErrNotExist) {
		return internalError("stat config", err)
	}
	if !create && errors.Is(err, os.ErrNotExist) {
		return notFound("config not found")
	}
	if !create && err != nil {
		return internalError("stat config", err)
	}
	if !create {
		if err := ensureRegularFile(path); err != nil {
			return internalError("stat config", err)
		}
	}

	data, err := json.MarshalIndent(rule, "", "  ")
	if err != nil {
		return internalError("marshal config", err)
	}
	data = append(data, '\n')
	if err := writeFileAtomic(path, data, 0600); err != nil {
		return internalError("write config", err)
	}
	return nil
}

func (api *httpAPI) deleteMergeRule(id string) *apiError {
	path := api.configPath(id)
	if err := ensureRegularFile(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFound("config not found")
		}
		return internalError("stat config", err)
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFound("config not found")
		}
		return internalError("delete config", err)
	}
	return nil
}

func (api *httpAPI) validateMergeRule(rule MergeRule) error {
	if strings.TrimSpace(rule.Template) == "" {
		return fmt.Errorf("template is required")
	}
	if _, err := api.resolveManagedPath(rule.Template); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	for group, sources := range rule.Configurations {
		if strings.TrimSpace(group) == "" {
			return fmt.Errorf("configuration group name is required")
		}
		for _, source := range sources {
			if err := validateConfigSource(source); err != nil {
				return err
			}
			if source.Path != "" {
				if _, err := api.resolveManagedPath(source.Path); err != nil {
					return fmt.Errorf("source path: %w", err)
				}
			}
		}
	}
	return nil
}

func (api *httpAPI) resolveMergeRule(rule MergeRule) (MergeRule, error) {
	next := rule

	template, err := api.resolveExistingManagedPath(rule.Template)
	if err != nil {
		return MergeRule{}, fmt.Errorf("template: %w", err)
	}
	next.Template = template

	next.Configurations = make(map[string][]ConfigSource, len(rule.Configurations))
	for name, sources := range rule.Configurations {
		copied := make([]ConfigSource, 0, len(sources))
		for _, source := range sources {
			if source.Path != "" {
				path, err := api.resolveExistingManagedPath(source.Path)
				if err != nil {
					return MergeRule{}, fmt.Errorf("source path: %w", err)
				}
				source.Path = path
			}
			copied = append(copied, source)
		}
		next.Configurations[name] = copied
	}

	return next, nil
}

func (api *httpAPI) templatePath(id string) (string, *apiError) {
	rule, err := api.readMergeRule(id)
	if err != nil {
		return "", err
	}
	path, resolveErr := api.resolveExistingManagedPath(rule.Template)
	if resolveErr != nil {
		return "", badRequest(fmt.Sprintf("template: %v", resolveErr))
	}
	return path, nil
}

func (api *httpAPI) configPath(id string) string {
	return filepath.Join(api.configDir, id+".json")
}

func (api *httpAPI) resolveManagedPath(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("path is required")
	}

	var candidate string
	if filepath.IsAbs(raw) {
		candidate = filepath.Clean(raw)
	} else {
		candidate = filepath.Join(api.configDir, raw)
	}
	if err := ensurePathInside(api.configDir, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func (api *httpAPI) resolveExistingManagedPath(raw string) (string, error) {
	path, err := api.resolveManagedPath(raw)
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if err := ensurePathInside(api.configDir, realPath); err != nil {
		return "", err
	}
	return realPath, nil
}

func (api *httpAPI) authorizeBearer(r *http.Request) bool {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	return constantTimeEqual(strings.TrimPrefix(header, prefix), api.token)
}

func (api *httpAPI) authorizeQueryToken(r *http.Request) bool {
	return constantTimeEqual(r.URL.Query().Get("token"), api.token)
}

func decodeMergeRuleBody(r *http.Request) (MergeRule, *apiError) {
	var rule MergeRule
	if err := decodeJSON(r, &rule); err != nil {
		return MergeRule{}, badRequest(err.Error())
	}
	return rule, nil
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid JSON: multiple values")
		}
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if decoder.More() {
		return fmt.Errorf("invalid JSON: multiple values")
	}
	return nil
}

func decodeJSONAny(r *http.Request) (any, error) {
	var value any
	if err := decodeJSON(r, &value); err != nil {
		return nil, err
	}
	return normalizeJSONNumbers(value), nil
}

func normalizeJSONNumbers(value any) any {
	switch v := value.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
		f, _ := v.Float64()
		return f
	case []any:
		for i := range v {
			v[i] = normalizeJSONNumbers(v[i])
		}
		return v
	case map[string]any:
		for k := range v {
			v[k] = normalizeJSONNumbers(v[k])
		}
		return v
	default:
		return value
	}
}

func splitRequestPath(path string) ([]string, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil, nil
	}

	rawParts := strings.Split(path, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part == "" {
			return nil, fmt.Errorf("invalid path")
		}
		unescaped, err := url.PathUnescape(part)
		if err != nil {
			return nil, fmt.Errorf("invalid path escape")
		}
		parts = append(parts, unescaped)
	}
	return parts, nil
}

func validateConfigID(id string) error {
	if id == "" {
		return fmt.Errorf("config id is required")
	}
	if id == "." || id == ".." || strings.Contains(id, "..") {
		return fmt.Errorf("invalid config id")
	}
	for _, r := range id {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			continue
		}
		return fmt.Errorf("invalid config id")
	}
	return nil
}

func validateProviderName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("provider name is required")
	}
	if strings.ContainsAny(name, "\r\n/") {
		return fmt.Errorf("invalid provider name")
	}
	return nil
}

func validateGroupName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("group name is required")
	}
	if name != strings.TrimSpace(name) || strings.ContainsAny(name, "\r\n/") {
		return fmt.Errorf("invalid group name")
	}
	return nil
}

func validateNonDefaultGroupName(name string) error {
	if err := validateGroupName(name); err != nil {
		return err
	}
	if name == defaultGroupName {
		return fmt.Errorf("%q is reserved", defaultGroupName)
	}
	return nil
}

func validateRuleString(rule string) error {
	if strings.TrimSpace(rule) == "" {
		return fmt.Errorf("rule is required")
	}
	if strings.ContainsAny(rule, "\r\n") {
		return fmt.Errorf("invalid rule")
	}
	return nil
}

func parseIndex(raw string) (int, *apiError) {
	index, err := strconv.Atoi(raw)
	if err != nil || index < 0 {
		return 0, badRequest("invalid index")
	}
	return index, nil
}

func constantTimeEqual(got, want string) bool {
	if got == "" || want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func ensurePathInside(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes config dir")
	}
	return nil
}

func ensureRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", path)
	}
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer os.Remove(tmp)

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(perm); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write json response failed: %v", err)
	}
}

func writeAPIError(w http.ResponseWriter, err *apiError) {
	if err == nil {
		return
	}
	writeJSON(w, err.Status, err)
}

func badRequest(message string) *apiError {
	return &apiError{Status: http.StatusBadRequest, Message: message}
}

func notFound(message string) *apiError {
	return &apiError{Status: http.StatusNotFound, Message: message}
}

func conflict(message string) *apiError {
	return &apiError{Status: http.StatusConflict, Message: message}
}

func methodNotAllowed(message string) *apiError {
	return &apiError{Status: http.StatusMethodNotAllowed, Message: message}
}

func internalError(action string, err error) *apiError {
	log.Printf("%s failed: %v", action, err)
	return &apiError{Status: http.StatusInternalServerError, Message: action + " failed"}
}
