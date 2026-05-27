package composer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

const testAPIToken = "secret-token"

func TestHTTPAPIConfigCRUDAndSubscription(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", `
mixed-port: 7890
proxy-groups: []
rules:
  - MATCH,DIRECT
`)
	writeTestFile(t, dir, "proxy.yaml", `
proxies:
  - name: Test-Proxy
    type: ss
    server: 127.0.0.1
    port: 8388
    cipher: aes-128-gcm
    password: change-me
`)

	api := newTestAPI(t, dir)
	rule := MergeRule{
		Template: "template.yaml",
		Configurations: map[string]ConfigGroup{
			"Auto": {
				Sources:       []ConfigSource{{Path: "proxy.yaml"}},
				IncludeDirect: true,
			},
		},
		RulesetStrategy: UrlRuleset,
	}

	resp := performJSONRequest(t, api, http.MethodPost, "/api/configs/demo", rule, true)
	assertStatus(t, resp, http.StatusCreated)

	resp = performRequest(t, api, http.MethodGet, "/api/configs", nil, true)
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(resp.Body.String(), `"demo"`) {
		t.Fatalf("config list missing demo: %s", resp.Body.String())
	}

	resp = performRequest(t, api, http.MethodGet, "/api/configs/demo", nil, true)
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(resp.Body.String(), `"template":"template.yaml"`) {
		t.Fatalf("config body mismatch: %s", resp.Body.String())
	}

	resp = performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml", nil, true)
	assertStatus(t, resp, http.StatusUnauthorized)

	resp = performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml?token="+testAPIToken, nil, false)
	assertStatus(t, resp, http.StatusOK)
	body := resp.Body.String()
	for _, want := range []string{"Test-Proxy", "Auto-UrlTest", "DIRECT", "MATCH,DIRECT"} {
		if !strings.Contains(body, want) {
			t.Fatalf("subscription body missing %q:\n%s", want, body)
		}
	}

	resp = performJSONRequest(t, api, http.MethodPost, "/api/configs/..%2Fbad", rule, true)
	assertStatus(t, resp, http.StatusBadRequest)

	resp = performRequest(t, api, http.MethodDelete, "/api/configs/demo", nil, true)
	assertStatus(t, resp, http.StatusNoContent)

	resp = performRequest(t, api, http.MethodGet, "/api/configs/demo", nil, true)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestHTTPAPINormalizesLegacyConfigurationGroups(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", "rules: []\n")
	writeTestFile(t, dir, "demo.json", `{
  "template": "template.yaml",
  "configurations": {
    "Legacy": [
      {
        "path": "proxy.yaml"
      }
    ]
  }
}
`)

	api := newTestAPI(t, dir)
	resp := performRequest(t, api, http.MethodGet, "/api/configs/demo", nil, true)
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(resp.Body.String(), `"sources":[{"path":"proxy.yaml"`) {
		t.Fatalf("legacy group was not normalized: %s", resp.Body.String())
	}
}

func TestHTTPAPIRejectsInvalidGroupIncludes(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", "rules: []\n")

	api := newTestAPI(t, dir)
	rule := MergeRule{
		Template: "template.yaml",
		Configurations: map[string]ConfigGroup{
			"Auto": {IncludeGroups: []string{"Missing"}},
		},
	}

	resp := performJSONRequest(t, api, http.MethodPost, "/api/configs/demo", rule, true)
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(resp.Body.String(), "unknown proxy group") {
		t.Fatalf("unexpected response: %s", resp.Body.String())
	}
}

func TestHTTPAPIRuleProvidersCRUD(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", `
rule-providers:
  reject:
    type: http
    behavior: domain
    url: https://example.com/reject.txt
    path: ./ruleset/reject.yaml
    interval: 86400
rules:
  - MATCH,DIRECT
`)
	writeMergeRule(t, dir, "demo", MergeRule{Template: "template.yaml"})
	api := newTestAPI(t, dir)

	resp := performRequest(t, api, http.MethodGet, "/api/configs/demo/template/rule-providers/reject", nil, true)
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(resp.Body.String(), `"behavior":"domain"`) {
		t.Fatalf("provider body mismatch: %s", resp.Body.String())
	}

	provider := map[string]any{
		"type":     "http",
		"behavior": "ipcidr",
		"url":      "https://example.com/cncidr.txt",
		"path":     "./ruleset/cncidr.yaml",
		"interval": 86400,
	}
	resp = performJSONRequest(t, api, http.MethodPost, "/api/configs/demo/template/rule-providers/cncidr", provider, true)
	assertStatus(t, resp, http.StatusCreated)

	provider["behavior"] = "classical"
	resp = performJSONRequest(t, api, http.MethodPut, "/api/configs/demo/template/rule-providers/cncidr", provider, true)
	assertStatus(t, resp, http.StatusOK)

	resp = performRequest(t, api, http.MethodGet, "/api/configs/demo/template/rule-providers/cncidr", nil, true)
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(resp.Body.String(), `"behavior":"classical"`) {
		t.Fatalf("updated provider body mismatch: %s", resp.Body.String())
	}

	resp = performRequest(t, api, http.MethodDelete, "/api/configs/demo/template/rule-providers/reject", nil, true)
	assertStatus(t, resp, http.StatusNoContent)

	resp = performRequest(t, api, http.MethodGet, "/api/configs/demo/template/rule-providers/reject", nil, true)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestHTTPAPIRuleGroupsCRUD(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", `
rules:
  - RULE-SET,private,DIRECT

  # steam
  - DOMAIN-SUFFIX,steampowered.com,DIRECT
  - RULE-SET,reject,REJECT

  # openai
  - DOMAIN-SUFFIX,openai.com,High
`)
	writeMergeRule(t, dir, "demo", MergeRule{Template: "template.yaml"})
	api := newTestAPI(t, dir)

	resp := performRequest(t, api, http.MethodGet, "/api/configs/demo/template/rule-groups", nil, true)
	assertStatus(t, resp, http.StatusOK)
	var groups []RuleGroup
	decodeResponse(t, resp, &groups)
	assertGroupNames(t, groups, []string{"default", "steam", "openai"})

	create := createRuleGroupRequest{
		Name:  "google",
		Index: intPtr(2),
		Rules: []string{"RULE-SET,google,High"},
	}
	resp = performJSONRequest(t, api, http.MethodPost, "/api/configs/demo/template/rule-groups", create, true)
	assertStatus(t, resp, http.StatusCreated)

	resp = performRequest(t, api, http.MethodGet, "/api/configs/demo/template/rule-groups", nil, true)
	assertStatus(t, resp, http.StatusOK)
	decodeResponse(t, resp, &groups)
	assertGroupNames(t, groups, []string{"default", "steam", "google", "openai"})

	resp = performJSONRequest(t, api, http.MethodPost, "/api/configs/demo/template/rule-groups/google/rules", updateRuleRequest{
		Rule:  "DOMAIN-SUFFIX,google.com,High",
		Index: intPtr(1),
	}, true)
	assertStatus(t, resp, http.StatusCreated)

	resp = performJSONRequest(t, api, http.MethodPut, "/api/configs/demo/template/rule-groups/google/rules/0", updateRuleRequest{
		Rule: "RULE-SET,google,Auto",
	}, true)
	assertStatus(t, resp, http.StatusOK)

	resp = performJSONRequest(t, api, http.MethodPut, "/api/configs/demo/template/rule-groups/google", updateRuleGroupRequest{
		Name: stringPtr("search"),
	}, true)
	assertStatus(t, resp, http.StatusOK)

	resp = performRequest(t, api, http.MethodDelete, "/api/configs/demo/template/rule-groups/steam", nil, true)
	assertStatus(t, resp, http.StatusNoContent)

	resp = performRequest(t, api, http.MethodDelete, "/api/configs/demo/template/rule-groups/default", nil, true)
	assertStatus(t, resp, http.StatusBadRequest)

	resp = performRequest(t, api, http.MethodGet, "/api/configs/demo/template/rule-groups", nil, true)
	assertStatus(t, resp, http.StatusOK)
	decodeResponse(t, resp, &groups)
	assertGroupNames(t, groups, []string{"default", "search", "openai"})

	data, err := os.ReadFile(filepath.Join(dir, "template.yaml"))
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "# search") || strings.Contains(text, "# steam") {
		t.Fatalf("template comments not updated as expected:\n%s", text)
	}
}

func TestHTTPAPIRejectsEscapingManagedPaths(t *testing.T) {
	dir := t.TempDir()
	api := newTestAPI(t, dir)

	rule := MergeRule{Template: "../template.yaml"}
	resp := performJSONRequest(t, api, http.MethodPost, "/api/configs/demo", rule, true)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestHTTPAPIStaticDisabledWithoutWebappFS(t *testing.T) {
	dir := t.TempDir()
	api := newTestAPI(t, dir)

	resp := performRequest(t, api, http.MethodGet, "/", nil, false)
	assertStatus(t, resp, http.StatusNotFound)
	if !strings.Contains(resp.Body.String(), "/api/") {
		t.Fatalf("expected hint mentioning /api/, got: %s", resp.Body.String())
	}
}

func TestHTTPAPIStaticServesIndexAndFallback(t *testing.T) {
	dir := t.TempDir()
	api, _, err := newHTTPAPI(ServeOptions{
		ConfigDir: dir,
		Token:     testAPIToken,
		WebappFS:  fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html>hi</html>")}, "assets/app.js": &fstest.MapFile{Data: []byte("console.log(1)")}},
	})
	if err != nil {
		t.Fatalf("new api: %v", err)
	}

	cases := []struct {
		path string
		body string
	}{
		{"/", "<html>hi</html>"},
		{"/assets/app.js", "console.log(1)"},
		{"/configs/demo", "<html>hi</html>"}, // SPA fallback for unknown routes (no /api prefix)
	}
	for _, tc := range cases {
		resp := performRequest(t, api, http.MethodGet, tc.path, nil, false)
		assertStatus(t, resp, http.StatusOK)
		if !strings.Contains(resp.Body.String(), tc.body) {
			t.Fatalf("path=%s body mismatch: %s", tc.path, resp.Body.String())
		}
	}
}

func newTestAPI(t *testing.T, dir string) *httpAPI {
	t.Helper()

	api, _, err := newHTTPAPI(ServeOptions{
		ConfigDir: dir,
		Token:     testAPIToken,
	})
	if err != nil {
		t.Fatalf("new api handler: %v", err)
	}
	return api
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(content, "\n")), 0644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func writeMergeRule(t *testing.T, dir, id string, rule MergeRule) {
	t.Helper()

	data, err := json.MarshalIndent(rule, "", "  ")
	if err != nil {
		t.Fatalf("marshal merge rule: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), append(data, '\n'), 0644); err != nil {
		t.Fatalf("write merge rule: %v", err)
	}
}

func performJSONRequest(t *testing.T, api http.Handler, method, target string, body any, auth bool) *httptest.ResponseRecorder {
	t.Helper()

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return performRequest(t, api, method, target, data, auth)
}

func performRequest(t *testing.T, api http.Handler, method, target string, body []byte, auth bool) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+testAPIToken)
	}
	resp := httptest.NewRecorder()
	api.ServeHTTP(resp, req)
	return resp
}

func decodeResponse(t *testing.T, resp *httptest.ResponseRecorder, out any) {
	t.Helper()

	if err := json.Unmarshal(resp.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response %q: %v", resp.Body.String(), err)
	}
}

func assertStatus(t *testing.T, resp *httptest.ResponseRecorder, want int) {
	t.Helper()

	if resp.Code != want {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, want, resp.Body.String())
	}
}

func assertGroupNames(t *testing.T, groups []RuleGroup, want []string) {
	t.Helper()

	if len(groups) != len(want) {
		t.Fatalf("len(groups) = %d, want %d: %#v", len(groups), len(want), groups)
	}
	for i := range want {
		if groups[i].Name != want[i] {
			t.Fatalf("groups[%d].Name = %q, want %q: %#v", i, groups[i].Name, want[i], groups)
		}
	}
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
