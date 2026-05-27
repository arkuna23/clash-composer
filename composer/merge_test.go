package composer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeExample(t *testing.T) {
	t.Chdir("..")

	data, err := os.ReadFile("example/merge.json")
	if err != nil {
		t.Fatalf("read merge example: %v", err)
	}

	var rule MergeRule
	if err := json.Unmarshal(data, &rule); err != nil {
		t.Fatalf("unmarshal merge example: %v", err)
	}

	cfg, err := Merge(rule)
	if err != nil {
		t.Fatalf("merge example: %v", err)
	}

	if got, want := len(cfg.Proxy), 6; got != want {
		t.Fatalf("len(Proxies) = %d, want %d", got, want)
	}

	groups := map[string]bool{}
	for _, group := range cfg.ProxyGroup {
		name, _ := group["name"].(string)
		groups[name] = true
	}
	for _, name := range []string{"High", "High-UrlTest", "Common", "Common-UrlTest"} {
		if !groups[name] {
			t.Fatalf("missing generated proxy group %q in %#v", name, groups)
		}
	}

	hasRule := func(want string) bool {
		for _, rule := range cfg.Rule {
			if rule == want {
				return true
			}
		}
		return false
	}

	for _, want := range []string{
		"DOMAIN-SUFFIX,openai.com,High",
		"RULE-SET,google,High",
		"DOMAIN-SUFFIX,microsoft.com,High",
		"DOMAIN-SUFFIX,anthropic.com,High",
		"DOMAIN-SUFFIX,steampowered.com,DIRECT",
	} {
		if !hasRule(want) {
			t.Fatalf("missing rule %q in %#v", want, cfg.Rule)
		}
	}
}

func TestMergeLogs(t *testing.T) {
	t.Chdir("..")

	data, err := os.ReadFile("example/merge.json")
	if err != nil {
		t.Fatalf("read merge example: %v", err)
	}

	var rule MergeRule
	if err := json.Unmarshal(data, &rule); err != nil {
		t.Fatalf("unmarshal merge example: %v", err)
	}

	var buf bytes.Buffer
	restoreLogger(t, &buf)

	if _, err := Merge(rule); err != nil {
		t.Fatalf("merge example: %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		`merge start: template="example/template.yaml"`,
		`load config source start: path="example/high.yaml"`,
		`load config source complete: path="example/common.yaml"`,
		`merge complete:`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("log output %q does not contain %q", output, want)
		}
	}
}

func TestLoadConfigSourcePathURLAndCmd(t *testing.T) {
	pathConfig := []byte(`
proxies:
  - name: Path-Proxy
    type: ss
    server: 127.0.0.1
    port: 8388
    cipher: aes-128-gcm
    password: change-me
`)
	pathFile := t.TempDir() + "/path.yaml"
	if err := os.WriteFile(pathFile, pathConfig, 0644); err != nil {
		t.Fatalf("write path fixture: %v", err)
	}

	cmdConfig := []byte(`
proxies:
  - name: Cmd-Proxy
    type: socks5
    server: 127.0.0.1
    port: 1080
    udp: true
`)
	cmdFile := t.TempDir() + "/cmd.yaml"
	if err := os.WriteFile(cmdFile, cmdConfig, 0644); err != nil {
		t.Fatalf("write cmd fixture: %v", err)
	}

	restoreHTTPClient(t, &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://example.com/url.yaml" {
				t.Fatalf("unexpected URL %q", req.URL.String())
			}
			if got, want := req.Header.Get("User-Agent"), clashVergeUserAgent; got != want {
				t.Fatalf("User-Agent = %q, want %q", got, want)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
proxies:
  - name: URL-Proxy
    type: trojan
    server: remote.example.com
    port: 443
    password: change-me
`)),
				Header: make(http.Header),
			}, nil
		}),
	})

	configs, err := loadConfigurations(map[string]ConfigGroup{
		"Mixed": {
			Sources: []ConfigSource{
				{Path: pathFile},
				{Url: "https://example.com/url.yaml"},
				{Cmd: fmt.Sprintf("cat %q", cmdFile)},
			},
		},
	}, MergeOptions{})
	if err != nil {
		t.Fatalf("load configurations: %v", err)
	}

	got := configs["Mixed"]
	if len(got) != 3 {
		t.Fatalf("len(Mixed) = %d, want 3", len(got))
	}
	if got[0].Proxy[0]["name"] != "Path-Proxy" {
		t.Fatalf("path config not loaded: %#v", got[0].Proxy)
	}
	if got[1].Proxy[0]["name"] != "URL-Proxy" {
		t.Fatalf("url config not loaded: %#v", got[1].Proxy)
	}
	if got[2].Proxy[0]["name"] != "Cmd-Proxy" {
		t.Fatalf("cmd config not loaded: %#v", got[2].Proxy)
	}
}

func TestLoadConfigSourceParseFailure(t *testing.T) {
	restoreHTTPClient(t, &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("not: [valid")),
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, err := loadConfigSource(ConfigSource{Url: "https://example.com/invalid.yaml"}, MergeOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `parse url="https://example.com/invalid.yaml"`) {
		t.Fatalf("error %q does not contain parse context", err)
	}
}

func TestLoadConfigSourceValidation(t *testing.T) {
	restoreHTTPClient(t, &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("bad gateway")),
				Header:     make(http.Header),
			}, nil
		}),
	})

	testCases := []struct {
		name string
		src  ConfigSource
		want string
	}{
		{
			name: "empty source",
			src:  ConfigSource{},
			want: "exactly one of path, url, or cmd",
		},
		{
			name: "path and url",
			src: ConfigSource{
				Path: "example/high.yaml",
				Url:  "https://example.com/config.yaml",
			},
			want: "exactly one of path, url, or cmd",
		},
		{
			name: "path and cmd",
			src: ConfigSource{
				Path: "example/high.yaml",
				Cmd:  "cat example/high.yaml",
			},
			want: "exactly one of path, url, or cmd",
		},
		{
			name: "url and cmd",
			src: ConfigSource{
				Url: "https://example.com/config.yaml",
				Cmd: "cat example/high.yaml",
			},
			want: "exactly one of path, url, or cmd",
		},
		{
			name: "bad status",
			src:  ConfigSource{Url: "https://example.com/bad.yaml"},
			want: "unexpected status 502",
		},
		{
			name: "bad command",
			src:  ConfigSource{Cmd: `printf 'bad cmd' >&2; exit 7`},
			want: "bad cmd",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadConfigSource(tc.src, MergeOptions{})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestLoadConfigSourceCmdUsesCommandDir(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "relative.yaml")
	if err := os.WriteFile(configFile, []byte(`
proxies:
  - name: Relative-Cmd-Proxy
    type: socks5
    server: 127.0.0.1
    port: 1080
`), 0644); err != nil {
		t.Fatalf("write cmd fixture: %v", err)
	}

	cfg, err := loadConfigSource(ConfigSource{Cmd: "cat relative.yaml"}, MergeOptions{
		CommandDir: dir,
	})
	if err != nil {
		t.Fatalf("load command config: %v", err)
	}
	if got := cfg.Proxy[0]["name"]; got != "Relative-Cmd-Proxy" {
		t.Fatalf("proxy name = %q, want Relative-Cmd-Proxy", got)
	}
}

func TestMergeWithOptionsPassesCommandDir(t *testing.T) {
	dir := t.TempDir()
	templateFile := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(templateFile, []byte(`
proxy-groups:
  - name: Existing
    type: select
    proxies:
      - DIRECT
rules: []
`), 0644); err != nil {
		t.Fatalf("write template fixture: %v", err)
	}

	cmdFile := filepath.Join(dir, "cmd.yaml")
	if err := os.WriteFile(cmdFile, []byte(`
proxies:
  - name: Merge-Cmd-Proxy
    type: socks5
    server: 127.0.0.1
    port: 1080
`), 0644); err != nil {
		t.Fatalf("write cmd fixture: %v", err)
	}

	cfg, err := MergeWithOptions(MergeRule{
		Template: templateFile,
		Configurations: map[string]ConfigGroup{
			"Cmd": {
				Sources: []ConfigSource{{Cmd: "cat cmd.yaml"}},
			},
		},
	}, MergeOptions{
		CommandDir: dir,
	})
	if err != nil {
		t.Fatalf("merge with command dir: %v", err)
	}
	if got := cfg.Proxy[0]["name"]; got != "Merge-Cmd-Proxy" {
		t.Fatalf("proxy name = %q, want Merge-Cmd-Proxy", got)
	}
}

func TestMergeYAMLWithOptionsPreservesTemplateFields(t *testing.T) {
	dir := t.TempDir()
	templateFile := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(templateFile, []byte(`
mixed-port: 7890
custom-field:
  enabled: yes
dns:
  enabled: true
rules:
  # keep-comment
  - MATCH,DIRECT
`), 0644); err != nil {
		t.Fatalf("write template fixture: %v", err)
	}

	proxyFile := filepath.Join(dir, "proxy.yaml")
	if err := os.WriteFile(proxyFile, []byte(`
proxies:
  - name: Template-Proxy
    type: socks5
    server: 127.0.0.1
    port: 1080
`), 0644); err != nil {
		t.Fatalf("write proxy fixture: %v", err)
	}

	_, data, err := MergeYAMLWithOptions(MergeRule{
		Template: templateFile,
		Configurations: map[string]ConfigGroup{
			"Auto": {
				Sources: []ConfigSource{{Path: proxyFile}},
			},
		},
	}, MergeOptions{})
	if err != nil {
		t.Fatalf("merge yaml: %v", err)
	}

	output := string(data)
	for _, want := range []string{
		"mixed-port: 7890",
		"custom-field:",
		"enabled: true",
		"# keep-comment",
		"proxies:",
		"name: Template-Proxy",
		"proxy-groups:",
		"name: Auto-UrlTest",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("merged YAML missing %q:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{
		"bind-address:",
		"mode:",
		"ntp:",
		"tun:",
		"use-hosts:",
		"default-nameserver:",
	} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("merged YAML contains template-absent field %q:\n%s", unwanted, output)
		}
	}
}

func TestConfigGroupUnmarshalLegacyArray(t *testing.T) {
	var rule MergeRule
	if err := json.Unmarshal([]byte(`{
		"template": "template.yaml",
		"configurations": {
			"Legacy": [
				{ "path": "legacy.yaml" }
			]
		}
	}`), &rule); err != nil {
		t.Fatalf("unmarshal legacy rule: %v", err)
	}

	got := rule.Configurations["Legacy"].Sources
	if len(got) != 1 || got[0].Path != "legacy.yaml" {
		t.Fatalf("legacy sources = %#v", got)
	}
}

func TestMergeGroupIncludesDirectAndGroups(t *testing.T) {
	dir := t.TempDir()
	templateFile := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(templateFile, []byte(`
proxy-groups:
  - name: TemplateGroup
    type: select
    proxies:
      - DIRECT
rules: []
`), 0644); err != nil {
		t.Fatalf("write template fixture: %v", err)
	}

	proxyFile := filepath.Join(dir, "proxy.yaml")
	if err := os.WriteFile(proxyFile, []byte(`
proxies:
  - name: Include-Proxy
    type: socks5
    server: 127.0.0.1
    port: 1080
`), 0644); err != nil {
		t.Fatalf("write proxy fixture: %v", err)
	}

	cfg, err := Merge(MergeRule{
		Template: templateFile,
		Configurations: map[string]ConfigGroup{
			"Auto": {
				Sources:       []ConfigSource{{Path: proxyFile}},
				IncludeDirect: true,
				IncludeGroups: []string{"TemplateGroup", "Other"},
			},
			"Other": {},
		},
	})
	if err != nil {
		t.Fatalf("merge includes: %v", err)
	}

	var auto map[string]any
	for _, group := range cfg.ProxyGroup {
		if group["name"] == "Auto" {
			auto = group
			break
		}
	}
	if auto == nil {
		t.Fatalf("missing Auto group: %#v", cfg.ProxyGroup)
	}
	got, ok := auto["proxies"].([]string)
	if !ok {
		t.Fatalf("Auto proxies type = %T", auto["proxies"])
	}
	want := []string{"Auto-UrlTest", "DIRECT", "TemplateGroup", "Other", "Include-Proxy"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Auto proxies = %#v, want %#v", got, want)
	}
}

func TestMergeRejectsInvalidGroupIncludes(t *testing.T) {
	dir := t.TempDir()
	templateFile := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(templateFile, []byte("rules: []\n"), 0644); err != nil {
		t.Fatalf("write template fixture: %v", err)
	}

	testCases := []struct {
		name   string
		groups map[string]ConfigGroup
		want   string
	}{
		{
			name: "empty",
			groups: map[string]ConfigGroup{
				"Auto": {IncludeGroups: []string{""}},
			},
			want: "empty proxy group name",
		},
		{
			name: "self",
			groups: map[string]ConfigGroup{
				"Auto": {IncludeGroups: []string{"Auto"}},
			},
			want: "cannot include itself",
		},
		{
			name: "missing",
			groups: map[string]ConfigGroup{
				"Auto": {IncludeGroups: []string{"Missing"}},
			},
			want: "unknown proxy group",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Merge(MergeRule{
				Template:       templateFile,
				Configurations: tc.groups,
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func restoreHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()

	oldClient := httpClient
	httpClient = client
	t.Cleanup(func() {
		httpClient = oldClient
	})
}

func restoreLogger(t *testing.T, output io.Writer) {
	t.Helper()

	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(output)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
