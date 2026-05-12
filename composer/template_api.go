package composer

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type RuleGroup struct {
	Name  string   `json:"name"`
	Rules []string `json:"rules"`
}

type createRuleGroupRequest struct {
	Name  string   `json:"name"`
	Index *int     `json:"index"`
	Rules []string `json:"rules"`
}

type updateRuleGroupRequest struct {
	Name  *string   `json:"name"`
	Rules *[]string `json:"rules"`
}

type updateRuleRequest struct {
	Rule  string `json:"rule"`
	Index *int   `json:"index,omitempty"`
}

func (api *httpAPI) handleRuleProviders(w http.ResponseWriter, r *http.Request, id string, rest []string) {
	path, err := api.templatePath(id)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	switch len(rest) {
	case 0:
		if r.Method != http.MethodGet {
			writeAPIError(w, methodNotAllowed("method not allowed"))
			return
		}
		providers, err := listRuleProviders(path)
		if err != nil {
			writeAPIError(w, internalError("list rule providers", err))
			return
		}
		writeJSON(w, http.StatusOK, providers)
	case 1:
		name := rest[0]
		if err := validateProviderName(name); err != nil {
			writeAPIError(w, badRequest(err.Error()))
			return
		}
		api.handleRuleProvider(w, r, path, name)
	default:
		writeAPIError(w, notFound("not found"))
	}
}

func (api *httpAPI) handleRuleProvider(w http.ResponseWriter, r *http.Request, path, name string) {
	switch r.Method {
	case http.MethodGet:
		provider, apiErr := getRuleProvider(path, name)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusOK, provider)
	case http.MethodPost:
		provider, err := decodeProviderBody(r)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		if apiErr := setRuleProvider(path, name, provider, true); apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusCreated, provider)
	case http.MethodPut:
		provider, err := decodeProviderBody(r)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		if apiErr := setRuleProvider(path, name, provider, false); apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusOK, provider)
	case http.MethodDelete:
		if apiErr := deleteRuleProvider(path, name); apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeAPIError(w, methodNotAllowed("method not allowed"))
	}
}

func (api *httpAPI) handleRuleGroups(w http.ResponseWriter, r *http.Request, id string, rest []string) {
	path, err := api.templatePath(id)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			groups, apiErr := listRuleGroups(path)
			if apiErr != nil {
				writeAPIError(w, apiErr)
				return
			}
			writeJSON(w, http.StatusOK, groups)
		case http.MethodPost:
			api.handleCreateRuleGroup(w, r, path)
		default:
			writeAPIError(w, methodNotAllowed("method not allowed"))
		}
		return
	}

	name := rest[0]
	if err := validateGroupName(name); err != nil {
		writeAPIError(w, badRequest(err.Error()))
		return
	}

	if len(rest) == 1 {
		api.handleRuleGroup(w, r, path, name)
		return
	}

	if len(rest) >= 2 && rest[1] == "rules" {
		api.handleRuleGroupRules(w, r, path, name, rest[2:])
		return
	}

	writeAPIError(w, notFound("not found"))
}

func (api *httpAPI) handleCreateRuleGroup(w http.ResponseWriter, r *http.Request, path string) {
	var req createRuleGroupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, badRequest(err.Error()))
		return
	}
	if err := validateNonDefaultGroupName(req.Name); err != nil {
		writeAPIError(w, badRequest(err.Error()))
		return
	}
	if req.Index == nil {
		writeAPIError(w, badRequest("index is required"))
		return
	}
	if len(req.Rules) == 0 {
		writeAPIError(w, badRequest("at least one rule is required"))
		return
	}
	if err := validateRules(req.Rules); err != nil {
		writeAPIError(w, badRequest(err.Error()))
		return
	}

	group := RuleGroup{Name: req.Name, Rules: req.Rules}
	if apiErr := createRuleGroup(path, *req.Index, group); apiErr != nil {
		writeAPIError(w, apiErr)
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (api *httpAPI) handleRuleGroup(w http.ResponseWriter, r *http.Request, path, name string) {
	switch r.Method {
	case http.MethodGet:
		group, apiErr := getRuleGroup(path, name)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusOK, group)
	case http.MethodPut:
		var req updateRuleGroupRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, badRequest(err.Error()))
			return
		}
		group, apiErr := updateRuleGroup(path, name, req)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusOK, group)
	case http.MethodDelete:
		if apiErr := deleteRuleGroup(path, name); apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeAPIError(w, methodNotAllowed("method not allowed"))
	}
}

func (api *httpAPI) handleRuleGroupRules(w http.ResponseWriter, r *http.Request, path, name string, rest []string) {
	switch len(rest) {
	case 0:
		if r.Method != http.MethodPost {
			writeAPIError(w, methodNotAllowed("method not allowed"))
			return
		}
		var req updateRuleRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, badRequest(err.Error()))
			return
		}
		if err := validateRuleString(req.Rule); err != nil {
			writeAPIError(w, badRequest(err.Error()))
			return
		}
		group, apiErr := addRuleToGroup(path, name, req)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusCreated, group)
	case 1:
		index, err := parseIndex(rest[0])
		if err != nil {
			writeAPIError(w, err)
			return
		}
		api.handleRuleGroupRule(w, r, path, name, index)
	default:
		writeAPIError(w, notFound("not found"))
	}
}

func (api *httpAPI) handleRuleGroupRule(w http.ResponseWriter, r *http.Request, path, name string, index int) {
	switch r.Method {
	case http.MethodGet:
		rule, apiErr := getRuleFromGroup(path, name, index)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"rule": rule})
	case http.MethodPut:
		var req updateRuleRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, badRequest(err.Error()))
			return
		}
		if err := validateRuleString(req.Rule); err != nil {
			writeAPIError(w, badRequest(err.Error()))
			return
		}
		group, apiErr := updateRuleInGroup(path, name, index, req.Rule)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusOK, group)
	case http.MethodDelete:
		group, apiErr := deleteRuleFromGroup(path, name, index)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		writeJSON(w, http.StatusOK, group)
	default:
		writeAPIError(w, methodNotAllowed("method not allowed"))
	}
}

func decodeProviderBody(r *http.Request) (map[string]any, *apiError) {
	value, err := decodeJSONAny(r)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	provider, ok := value.(map[string]any)
	if !ok {
		return nil, badRequest("provider must be a JSON object")
	}
	return provider, nil
}

func listRuleProviders(path string) (any, error) {
	doc, err := loadTemplateDocument(path)
	if err != nil {
		return nil, err
	}
	root, err := documentMapping(doc)
	if err != nil {
		return nil, err
	}
	providers := mappingValue(root, "rule-providers")
	if providers == nil {
		return map[string]any{}, nil
	}
	if providers.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("rule-providers must be a mapping")
	}

	var value any
	if err := providers.Decode(&value); err != nil {
		return nil, err
	}
	if value == nil {
		return map[string]any{}, nil
	}
	return value, nil
}

func getRuleProvider(path, name string) (any, *apiError) {
	doc, err := loadTemplateDocument(path)
	if err != nil {
		return nil, internalError("read template", err)
	}
	providers, err := ruleProvidersMapping(doc, false)
	if err != nil {
		return nil, internalError("read rule providers", err)
	}
	if providers == nil {
		return nil, notFound("rule provider not found")
	}
	_, valueIndex := findMappingKey(providers, name)
	if valueIndex < 0 {
		return nil, notFound("rule provider not found")
	}

	var value any
	if err := providers.Content[valueIndex].Decode(&value); err != nil {
		return nil, internalError("decode rule provider", err)
	}
	return value, nil
}

func setRuleProvider(path, name string, provider map[string]any, create bool) *apiError {
	doc, err := loadTemplateDocument(path)
	if err != nil {
		return internalError("read template", err)
	}
	providers, err := ruleProvidersMapping(doc, true)
	if err != nil {
		return internalError("read rule providers", err)
	}

	_, valueIndex := findMappingKey(providers, name)
	if create && valueIndex >= 0 {
		return conflict("rule provider already exists")
	}
	if !create && valueIndex < 0 {
		return notFound("rule provider not found")
	}

	valueNode := &yaml.Node{}
	if err := valueNode.Encode(provider); err != nil {
		return internalError("encode rule provider", err)
	}

	if valueIndex >= 0 {
		providers.Content[valueIndex] = valueNode
	} else {
		providers.Content = append(providers.Content, stringNode(name), valueNode)
	}

	if err := saveTemplateDocument(path, doc); err != nil {
		return internalError("write template", err)
	}
	return nil
}

func deleteRuleProvider(path, name string) *apiError {
	doc, err := loadTemplateDocument(path)
	if err != nil {
		return internalError("read template", err)
	}
	providers, err := ruleProvidersMapping(doc, false)
	if err != nil {
		return internalError("read rule providers", err)
	}
	if providers == nil {
		return notFound("rule provider not found")
	}

	keyIndex, valueIndex := findMappingKey(providers, name)
	if valueIndex < 0 {
		return notFound("rule provider not found")
	}
	providers.Content = append(providers.Content[:keyIndex], providers.Content[valueIndex+1:]...)

	if err := saveTemplateDocument(path, doc); err != nil {
		return internalError("write template", err)
	}
	return nil
}

func listRuleGroups(path string) ([]RuleGroup, *apiError) {
	doc, err := loadTemplateDocument(path)
	if err != nil {
		return nil, internalError("read template", err)
	}
	seq, err := rulesSequence(doc, false)
	if err != nil {
		return nil, internalError("read rules", err)
	}
	return parseRuleGroups(seq), nil
}

func getRuleGroup(path, name string) (RuleGroup, *apiError) {
	groups, apiErr := listRuleGroups(path)
	if apiErr != nil {
		return RuleGroup{}, apiErr
	}
	index, apiErr := uniqueGroupIndex(groups, name)
	if apiErr != nil {
		return RuleGroup{}, apiErr
	}
	return groups[index], nil
}

func createRuleGroup(path string, index int, group RuleGroup) *apiError {
	doc, err := loadTemplateDocument(path)
	if err != nil {
		return internalError("read template", err)
	}
	seq, err := rulesSequence(doc, true)
	if err != nil {
		return internalError("read rules", err)
	}
	groups := parseRuleGroups(seq)
	if hasGroup(groups, group.Name) {
		return conflict("rule group already exists")
	}
	if index <= 0 || index > len(groups) {
		return badRequest("index must insert after default group")
	}

	groups = append(groups, RuleGroup{})
	copy(groups[index+1:], groups[index:])
	groups[index] = group
	writeRuleGroups(seq, groups)

	if err := saveTemplateDocument(path, doc); err != nil {
		return internalError("write template", err)
	}
	return nil
}

func updateRuleGroup(path, name string, req updateRuleGroupRequest) (RuleGroup, *apiError) {
	if req.Name == nil && req.Rules == nil {
		return RuleGroup{}, badRequest("name or rules is required")
	}

	doc, err := loadTemplateDocument(path)
	if err != nil {
		return RuleGroup{}, internalError("read template", err)
	}
	seq, err := rulesSequence(doc, true)
	if err != nil {
		return RuleGroup{}, internalError("read rules", err)
	}
	groups := parseRuleGroups(seq)
	index, apiErr := uniqueGroupIndex(groups, name)
	if apiErr != nil {
		return RuleGroup{}, apiErr
	}

	if req.Name != nil {
		if name == defaultGroupName {
			return RuleGroup{}, badRequest("default group cannot be renamed")
		}
		if err := validateNonDefaultGroupName(*req.Name); err != nil {
			return RuleGroup{}, badRequest(err.Error())
		}
		if hasOtherGroup(groups, *req.Name, index) {
			return RuleGroup{}, conflict("rule group already exists")
		}
		groups[index].Name = *req.Name
	}
	if req.Rules != nil {
		if err := validateRules(*req.Rules); err != nil {
			return RuleGroup{}, badRequest(err.Error())
		}
		groups[index].Rules = *req.Rules
	}

	updated := groups[index]
	writeRuleGroups(seq, groups)
	if err := saveTemplateDocument(path, doc); err != nil {
		return RuleGroup{}, internalError("write template", err)
	}
	return updated, nil
}

func deleteRuleGroup(path, name string) *apiError {
	if name == defaultGroupName {
		return badRequest("default group cannot be deleted")
	}

	doc, err := loadTemplateDocument(path)
	if err != nil {
		return internalError("read template", err)
	}
	seq, err := rulesSequence(doc, true)
	if err != nil {
		return internalError("read rules", err)
	}
	groups := parseRuleGroups(seq)
	index, apiErr := uniqueGroupIndex(groups, name)
	if apiErr != nil {
		return apiErr
	}

	groups = append(groups[:index], groups[index+1:]...)
	writeRuleGroups(seq, groups)
	if err := saveTemplateDocument(path, doc); err != nil {
		return internalError("write template", err)
	}
	return nil
}

func addRuleToGroup(path, name string, req updateRuleRequest) (RuleGroup, *apiError) {
	doc, groups, index, apiErr := loadRuleGroupForEdit(path, name)
	if apiErr != nil {
		return RuleGroup{}, apiErr
	}

	insertIndex := len(groups[index].Rules)
	if req.Index != nil {
		insertIndex = *req.Index
	}
	if insertIndex < 0 || insertIndex > len(groups[index].Rules) {
		return RuleGroup{}, badRequest("invalid index")
	}
	groups[index].Rules = append(groups[index].Rules, "")
	copy(groups[index].Rules[insertIndex+1:], groups[index].Rules[insertIndex:])
	groups[index].Rules[insertIndex] = req.Rule

	if apiErr := saveRuleGroups(path, doc, groups); apiErr != nil {
		return RuleGroup{}, apiErr
	}
	return groups[index], nil
}

func getRuleFromGroup(path, name string, ruleIndex int) (string, *apiError) {
	group, apiErr := getRuleGroup(path, name)
	if apiErr != nil {
		return "", apiErr
	}
	if ruleIndex >= len(group.Rules) {
		return "", notFound("rule not found")
	}
	return group.Rules[ruleIndex], nil
}

func updateRuleInGroup(path, name string, ruleIndex int, rule string) (RuleGroup, *apiError) {
	doc, groups, index, apiErr := loadRuleGroupForEdit(path, name)
	if apiErr != nil {
		return RuleGroup{}, apiErr
	}
	if ruleIndex >= len(groups[index].Rules) {
		return RuleGroup{}, notFound("rule not found")
	}
	groups[index].Rules[ruleIndex] = rule

	if apiErr := saveRuleGroups(path, doc, groups); apiErr != nil {
		return RuleGroup{}, apiErr
	}
	return groups[index], nil
}

func deleteRuleFromGroup(path, name string, ruleIndex int) (RuleGroup, *apiError) {
	doc, groups, index, apiErr := loadRuleGroupForEdit(path, name)
	if apiErr != nil {
		return RuleGroup{}, apiErr
	}
	if ruleIndex >= len(groups[index].Rules) {
		return RuleGroup{}, notFound("rule not found")
	}
	groups[index].Rules = append(groups[index].Rules[:ruleIndex], groups[index].Rules[ruleIndex+1:]...)
	updated := groups[index]

	if apiErr := saveRuleGroups(path, doc, groups); apiErr != nil {
		return RuleGroup{}, apiErr
	}
	return updated, nil
}

func loadRuleGroupForEdit(path, name string) (*yaml.Node, []RuleGroup, int, *apiError) {
	doc, err := loadTemplateDocument(path)
	if err != nil {
		return nil, nil, 0, internalError("read template", err)
	}
	seq, err := rulesSequence(doc, true)
	if err != nil {
		return nil, nil, 0, internalError("read rules", err)
	}
	groups := parseRuleGroups(seq)
	index, apiErr := uniqueGroupIndex(groups, name)
	if apiErr != nil {
		return nil, nil, 0, apiErr
	}
	return doc, groups, index, nil
}

func saveRuleGroups(path string, doc *yaml.Node, groups []RuleGroup) *apiError {
	seq, err := rulesSequence(doc, true)
	if err != nil {
		return internalError("read rules", err)
	}
	writeRuleGroups(seq, groups)
	if err := saveTemplateDocument(path, doc); err != nil {
		return internalError("write template", err)
	}
	return nil
}

func loadTemplateDocument(path string) (*yaml.Node, error) {
	if err := ensureRegularFile(path); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc := &yaml.Node{}
	if len(data) == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		return doc, nil
	}
	if err := yaml.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	if _, err := documentMapping(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func saveTemplateDocument(path string, doc *yaml.Node) error {
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	perm := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	return writeFileAtomic(path, data, perm)
}

func documentMapping(doc *yaml.Node) (*yaml.Node, error) {
	if doc.Kind == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	if doc.Kind != yaml.DocumentNode {
		return nil, fmt.Errorf("template must be a YAML document")
	}
	if len(doc.Content) == 0 {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("template root must be a mapping")
	}
	return root, nil
}

func ruleProvidersMapping(doc *yaml.Node, create bool) (*yaml.Node, error) {
	root, err := documentMapping(doc)
	if err != nil {
		return nil, err
	}
	providers := mappingValue(root, "rule-providers")
	if providers == nil {
		if !create {
			return nil, nil
		}
		providers = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, stringNode("rule-providers"), providers)
	}
	if providers.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("rule-providers must be a mapping")
	}
	return providers, nil
}

func rulesSequence(doc *yaml.Node, create bool) (*yaml.Node, error) {
	root, err := documentMapping(doc)
	if err != nil {
		return nil, err
	}
	rules := mappingValue(root, "rules")
	if rules == nil {
		if !create {
			return nil, nil
		}
		rules = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		root.Content = append(root.Content, stringNode("rules"), rules)
	}
	if rules.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("rules must be a sequence")
	}
	return rules, nil
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	_, valueIndex := findMappingKey(mapping, key)
	if valueIndex < 0 {
		return nil
	}
	return mapping.Content[valueIndex]
}

func findMappingKey(mapping *yaml.Node, key string) (int, int) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return -1, -1
	}
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return i, i + 1
		}
	}
	return -1, -1
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func parseRuleGroups(seq *yaml.Node) []RuleGroup {
	groups := []RuleGroup{{Name: defaultGroupName}}
	if seq == nil {
		return groups
	}

	current := 0
	for _, item := range seq.Content {
		category := ruleCategory(item.HeadComment)
		if category != "" {
			groups = append(groups, RuleGroup{Name: category})
			current = len(groups) - 1
		}
		groups[current].Rules = append(groups[current].Rules, item.Value)
	}
	return groups
}

func writeRuleGroups(seq *yaml.Node, groups []RuleGroup) {
	seq.Kind = yaml.SequenceNode
	seq.Tag = "!!seq"
	seq.Content = nil

	for _, group := range groups {
		if len(group.Rules) == 0 {
			continue
		}
		for i, rule := range group.Rules {
			node := stringNode(rule)
			if group.Name != defaultGroupName && i == 0 {
				node.HeadComment = group.Name
			}
			seq.Content = append(seq.Content, node)
		}
	}
}

func ruleCategory(comment string) string {
	lines := strings.Split(comment, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if line != "" {
			return line
		}
	}
	return ""
}

func uniqueGroupIndex(groups []RuleGroup, name string) (int, *apiError) {
	found := -1
	for i, group := range groups {
		if group.Name != name {
			continue
		}
		if found >= 0 {
			return 0, conflict("duplicate rule group")
		}
		found = i
	}
	if found < 0 {
		return 0, notFound("rule group not found")
	}
	return found, nil
}

func hasGroup(groups []RuleGroup, name string) bool {
	for _, group := range groups {
		if group.Name == name {
			return true
		}
	}
	return false
}

func hasOtherGroup(groups []RuleGroup, name string, index int) bool {
	for i, group := range groups {
		if i != index && group.Name == name {
			return true
		}
	}
	return false
}

func validateRules(rules []string) error {
	for _, rule := range rules {
		if err := validateRuleString(rule); err != nil {
			return err
		}
	}
	return nil
}
