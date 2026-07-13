package composer

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
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
		Configurations: ConfigGroups{
			{
				Name:          "Auto",
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

func TestHTTPAPISubscriptionCache(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", `
proxy-groups: []
rules:
  - MATCH,DIRECT
`)
	writeTestFile(t, dir, "proxy.yaml", `
proxies:
  - name: Cached-Proxy
    type: socks5
    server: first.example.com
    port: 1080
`)
	writeMergeRule(t, dir, "demo", MergeRule{
		Template:             "template.yaml",
		CacheDurationSeconds: 3600,
		Configurations: ConfigGroups{
			{Name: "Auto", Sources: []ConfigSource{{Path: "proxy.yaml"}}},
		},
	})
	api := newTestAPI(t, dir)

	resp := performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml?token="+testAPIToken, nil, false)
	assertStatus(t, resp, http.StatusOK)
	if got := resp.Header().Get("X-Clash-Composer-Cache"); got != "miss" {
		t.Fatalf("cache header = %q, want miss", got)
	}
	first := resp.Body.String()
	if !strings.Contains(first, "first.example.com") {
		t.Fatalf("first subscription body mismatch:\n%s", first)
	}

	writeTestFile(t, dir, "proxy.yaml", `
proxies:
  - name: Cached-Proxy
    type: socks5
    server: second.example.com
    port: 1080
`)
	resp = performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml?token="+testAPIToken, nil, false)
	assertStatus(t, resp, http.StatusOK)
	if got := resp.Header().Get("X-Clash-Composer-Cache"); got != "hit" {
		t.Fatalf("cache header = %q, want hit", got)
	}
	if body := resp.Body.String(); body != first || strings.Contains(body, "second.example.com") {
		t.Fatalf("expected cached body, got:\n%s", body)
	}
}

func TestHTTPAPISubscriptionCacheExpires(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", "proxy-groups: []\nrules: []\n")
	writeTestFile(t, dir, "proxy.yaml", `
proxies:
  - name: Expiring-Proxy
    type: socks5
    server: old.example.com
    port: 1080
`)
	writeMergeRule(t, dir, "demo", MergeRule{
		Template:             "template.yaml",
		CacheDurationSeconds: 1,
		Configurations: ConfigGroups{
			{Name: "Auto", Sources: []ConfigSource{{Path: "proxy.yaml"}}},
		},
	})
	api := newTestAPI(t, dir)

	resp := performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml?token="+testAPIToken, nil, false)
	assertStatus(t, resp, http.StatusOK)
	cachePath := filepath.Join(dir, ".cache", "subscriptions", "demo.yaml")
	expired := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(cachePath, expired, expired); err != nil {
		t.Fatalf("expire cache: %v", err)
	}
	writeTestFile(t, dir, "proxy.yaml", `
proxies:
  - name: Expiring-Proxy
    type: socks5
    server: fresh.example.com
    port: 1080
`)

	resp = performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml?token="+testAPIToken, nil, false)
	assertStatus(t, resp, http.StatusOK)
	if got := resp.Header().Get("X-Clash-Composer-Cache"); got != "miss" {
		t.Fatalf("cache header = %q, want miss", got)
	}
	if body := resp.Body.String(); !strings.Contains(body, "fresh.example.com") {
		t.Fatalf("expected refreshed body, got:\n%s", body)
	}
}

func TestHTTPAPISubscriptionCacheInvalidatedOnConfigWriteAndDelete(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", "proxy-groups: []\nrules: []\n")
	writeTestFile(t, dir, "proxy.yaml", `
proxies:
  - name: Invalidated-Proxy
    type: socks5
    server: first.example.com
    port: 1080
`)
	rule := MergeRule{
		Template:             "template.yaml",
		CacheDurationSeconds: 3600,
		Configurations: ConfigGroups{
			{Name: "Auto", Sources: []ConfigSource{{Path: "proxy.yaml"}}},
		},
	}
	writeMergeRule(t, dir, "demo", rule)
	api := newTestAPI(t, dir)

	resp := performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml?token="+testAPIToken, nil, false)
	assertStatus(t, resp, http.StatusOK)
	cachePath := filepath.Join(dir, ".cache", "subscriptions", "demo.yaml")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("stat cache: %v", err)
	}

	writeTestFile(t, dir, "proxy.yaml", `
proxies:
  - name: Invalidated-Proxy
    type: socks5
    server: second.example.com
    port: 1080
`)
	resp = performJSONRequest(t, api, http.MethodPut, "/api/configs/demo", rule, true)
	assertStatus(t, resp, http.StatusOK)
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("cache still exists after config write: %v", err)
	}

	resp = performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml?token="+testAPIToken, nil, false)
	assertStatus(t, resp, http.StatusOK)
	if body := resp.Body.String(); !strings.Contains(body, "second.example.com") {
		t.Fatalf("expected invalidated body, got:\n%s", body)
	}

	resp = performRequest(t, api, http.MethodDelete, "/api/configs/demo", nil, true)
	assertStatus(t, resp, http.StatusNoContent)
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("cache still exists after config delete: %v", err)
	}
}

func TestHTTPAPISubscriptionCacheInvalidatedOnTemplateEdit(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", `
proxy-groups: []
rules:
  - MATCH,DIRECT
`)
	writeMergeRule(t, dir, "demo", MergeRule{
		Template:             "template.yaml",
		CacheDurationSeconds: 3600,
	})
	api := newTestAPI(t, dir)

	resp := performRequest(t, api, http.MethodGet, "/api/subscriptions/demo.yaml?token="+testAPIToken, nil, false)
	assertStatus(t, resp, http.StatusOK)
	cachePath := filepath.Join(dir, ".cache", "subscriptions", "demo.yaml")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("stat cache: %v", err)
	}

	resp = performJSONRequest(t, api, http.MethodPost, "/api/configs/demo/template/rule-groups", createRuleGroupRequest{
		Name:  "custom",
		Index: intPtr(1),
		Rules: []string{"DOMAIN-SUFFIX,example.com,DIRECT"},
	}, true)
	assertStatus(t, resp, http.StatusCreated)
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("cache still exists after template edit: %v", err)
	}
}

func TestHTTPAPIRejectsNegativeCacheDuration(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", "rules: []\n")
	api := newTestAPI(t, dir)

	resp := performJSONRequest(t, api, http.MethodPost, "/api/configs/demo", MergeRule{
		Template:             "template.yaml",
		CacheDurationSeconds: -1,
	}, true)
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(resp.Body.String(), "cacheDurationSeconds") {
		t.Fatalf("unexpected response: %s", resp.Body.String())
	}
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
	if !strings.Contains(resp.Body.String(), `"configurations":[{"name":"Legacy","sources":[{"path":"proxy.yaml"`) {
		t.Fatalf("legacy group was not normalized: %s", resp.Body.String())
	}
}

func TestHTTPAPIRejectsInvalidGroupIncludes(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", "rules: []\n")

	api := newTestAPI(t, dir)
	rule := MergeRule{
		Template: "template.yaml",
		Configurations: ConfigGroups{
			{Name: "Auto", IncludeGroups: []string{"Missing"}},
		},
	}

	resp := performJSONRequest(t, api, http.MethodPost, "/api/configs/demo", rule, true)
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(resp.Body.String(), "unknown proxy group") {
		t.Fatalf("unexpected response: %s", resp.Body.String())
	}
}

func TestHTTPAPIUploadFile(t *testing.T) {
	dir := t.TempDir()
	api := newTestAPI(t, dir)

	resp := performUploadRequest(t, api, "/api/files?path=templates/base.yaml", "base.yaml", []byte("rules: []\n"), true)
	assertStatus(t, resp, http.StatusCreated)

	var result uploadFileResponse
	decodeResponse(t, resp, &result)
	if result.Path != "templates/base.yaml" || result.Size != len("rules: []\n") {
		t.Fatalf("upload response = %#v", result)
	}

	data, err := os.ReadFile(filepath.Join(dir, "templates", "base.yaml"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(data) != "rules: []\n" {
		t.Fatalf("uploaded content = %q", string(data))
	}

	resp = performUploadRequest(t, api, "/api/files?path=templates/base.yaml", "base.yaml", []byte("mixed-port: 7890\n"), true)
	assertStatus(t, resp, http.StatusConflict)

	resp = performUploadRequest(t, api, "/api/files?path=templates/base.yaml&overwrite=true", "base.yaml", []byte("mixed-port: 7890\n"), true)
	assertStatus(t, resp, http.StatusCreated)
	data, err = os.ReadFile(filepath.Join(dir, "templates", "base.yaml"))
	if err != nil {
		t.Fatalf("read overwritten file: %v", err)
	}
	if string(data) != "mixed-port: 7890\n" {
		t.Fatalf("overwritten content = %q", string(data))
	}
}

func TestHTTPAPIUploadFileValidation(t *testing.T) {
	dir := t.TempDir()
	api := newTestAPI(t, dir)

	testCases := []struct {
		name   string
		target string
		body   []byte
		want   int
	}{
		{
			name:   "empty path",
			target: "/api/files?path=",
			body:   []byte("rules: []\n"),
			want:   http.StatusBadRequest,
		},
		{
			name:   "escaping path",
			target: "/api/files?path=../escape.yaml",
			body:   []byte("rules: []\n"),
			want:   http.StatusBadRequest,
		},
		{
			name:   "json extension",
			target: "/api/files?path=config.json",
			body:   []byte("{}\n"),
			want:   http.StatusBadRequest,
		},
		{
			name:   "missing extension",
			target: "/api/files?path=config",
			body:   []byte("rules: []\n"),
			want:   http.StatusBadRequest,
		},
		{
			name:   "empty file",
			target: "/api/files?path=empty.yaml",
			body:   []byte{},
			want:   http.StatusBadRequest,
		},
		{
			name:   "invalid overwrite",
			target: "/api/files?path=base.yaml&overwrite=maybe",
			body:   []byte("rules: []\n"),
			want:   http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := performUploadRequest(t, api, tc.target, "config.yaml", tc.body, true)
			assertStatus(t, resp, tc.want)
		})
	}
}

func TestHTTPAPIUploadRejectsSymlinkParent(t *testing.T) {
	dir := t.TempDir()
	targetDir := t.TempDir()
	if err := os.Symlink(targetDir, filepath.Join(dir, "linked")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	api := newTestAPI(t, dir)
	resp := performUploadRequest(t, api, "/api/files?path=linked/base.yaml", "base.yaml", []byte("rules: []\n"), true)
	assertStatus(t, resp, http.StatusBadRequest)
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

func TestHTTPAPIRuleBatchAndRenameReferences(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", `
rules:
  - DOMAIN-SUFFIX,default.example,Target

  # Target
  - IP-CIDR,192.0.2.0/24,Target,no-resolve

  # unrelated
  - RULE-SET,Target,DIRECT
`)
	writeMergeRule(t, dir, "demo", MergeRule{Template: "template.yaml"})
	api := newTestAPI(t, dir)

	resp := performJSONRequest(t, api, http.MethodPost, "/api/configs/demo/template/rule-groups/Target/rules", addRulesRequest{
		RulesYAML: `- DOMAIN-SUFFIX,one.example,Target
- DOMAIN-SUFFIX,two.example,Target
`,
		Index: intPtr(1),
	}, true)
	assertStatus(t, resp, http.StatusCreated)
	var group RuleGroup
	decodeResponse(t, resp, &group)
	wantRules := []string{
		"IP-CIDR,192.0.2.0/24,Target,no-resolve",
		"DOMAIN-SUFFIX,one.example,Target",
		"DOMAIN-SUFFIX,two.example,Target",
	}
	if strings.Join(group.Rules, "\n") != strings.Join(wantRules, "\n") {
		t.Fatalf("batch rules = %#v, want %#v", group.Rules, wantRules)
	}

	resp = performJSONRequest(t, api, http.MethodPut, "/api/configs/demo/template/rule-groups/Target", updateRuleGroupRequest{
		Name: stringPtr("Renamed"),
	}, true)
	assertStatus(t, resp, http.StatusOK)

	data, err := os.ReadFile(filepath.Join(dir, "template.yaml"))
	if err != nil {
		t.Fatalf("read renamed template: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"# Renamed",
		"DOMAIN-SUFFIX,default.example,Renamed",
		"IP-CIDR,192.0.2.0/24,Renamed,no-resolve",
		"DOMAIN-SUFFIX,one.example,Renamed",
		"DOMAIN-SUFFIX,two.example,Renamed",
		"RULE-SET,Target,DIRECT",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("renamed template missing %q:\n%s", want, text)
		}
	}
}

func TestHTTPAPIProxyGroupTargets(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "template.yaml", `
proxy-groups:
  - name: Existing
    type: select
    proxies:
      - DIRECT
rules: []
`)
	disabled := false
	writeMergeRule(t, dir, "demo", MergeRule{
		Template: "template.yaml",
		Configurations: ConfigGroups{
			{Name: "Auto"},
			{Name: "Manual", EnableURLTest: &disabled},
		},
	})
	api := newTestAPI(t, dir)

	resp := performRequest(t, api, http.MethodGet, "/api/configs/demo/template/proxy-groups", nil, true)
	assertStatus(t, resp, http.StatusOK)
	var targets []string
	decodeResponse(t, resp, &targets)
	want := []string{"DIRECT", "REJECT", "Auto", "Auto-UrlTest", "Manual", "Existing"}
	if strings.Join(targets, ",") != strings.Join(want, ",") {
		t.Fatalf("targets = %#v, want %#v", targets, want)
	}
}

func TestParseRuleList(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name: "yaml list",
			input: `
- DOMAIN-SUFFIX,one.example,DIRECT
- DOMAIN-SUFFIX,two.example,DIRECT
`,
			want: []string{"DOMAIN-SUFFIX,one.example,DIRECT", "DOMAIN-SUFFIX,two.example,DIRECT"},
		},
		{
			name:  "plain lines",
			input: "DOMAIN,one.example,DIRECT\nMATCH,REJECT",
			want:  []string{"DOMAIN,one.example,DIRECT", "MATCH,REJECT"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRuleList(tc.input)
			if err != nil {
				t.Fatalf("parse rule list: %v", err)
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("rules = %#v, want %#v", got, tc.want)
			}
		})
	}

	invalidCases := []string{
		"rules: [",
		"[]",
		"rules: []",
		"rules:\n  - DOMAIN,one.example,DIRECT",
		"mixed-port: 7890\nrules:\n  - MATCH,REJECT",
	}
	for _, input := range invalidCases {
		if _, err := parseRuleList(input); err == nil {
			t.Fatalf("parseRuleList(%q) unexpectedly succeeded", input)
		}
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

func performUploadRequest(t *testing.T, api http.Handler, target, filename string, content []byte, auth bool) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create upload field: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("write upload field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
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
