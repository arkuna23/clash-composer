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
	"sync/atomic"
	"testing"
	"time"

	"clash-composer/config"
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

func TestLoadConfigurationsLoadsSourcesConcurrentlyWithLimit(t *testing.T) {
	const totalSources = maxConcurrentSources + 4
	started := make(chan struct{}, totalSources)
	release := make(chan struct{})
	var active int32
	var maxActive int32

	restoreHTTPClient(t, &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			current := atomic.AddInt32(&active, 1)
			for {
				previous := atomic.LoadInt32(&maxActive)
				if current <= previous || atomic.CompareAndSwapInt32(&maxActive, previous, current) {
					break
				}
			}
			started <- struct{}{}
			<-release
			atomic.AddInt32(&active, -1)

			name := req.URL.Query().Get("name")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`
proxies:
  - name: %s
    type: socks5
    server: 127.0.0.1
    port: 1080
`, name))),
				Header: make(http.Header),
			}, nil
		}),
	})

	sources := make([]ConfigSource, 0, totalSources)
	for i := range totalSources {
		sources = append(sources, ConfigSource{
			Url: fmt.Sprintf("https://example.com/config.yaml?name=Proxy-%02d", i),
		})
	}

	type loadResult struct {
		configs map[string][]*config.RawConfig
		err     error
	}
	done := make(chan loadResult, 1)
	go func() {
		configs, err := loadConfigurations(map[string]ConfigGroup{
			"Concurrent": {Sources: sources},
		}, MergeOptions{})
		done <- loadResult{configs: configs, err: err}
	}()

	for i := 0; i < maxConcurrentSources; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for source %d to start", i)
		}
	}
	select {
	case <-started:
		t.Fatalf("more than %d sources started before release", maxConcurrentSources)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	result := <-done
	if result.err != nil {
		t.Fatalf("load configurations: %v", result.err)
	}
	if got := atomic.LoadInt32(&maxActive); got != maxConcurrentSources {
		t.Fatalf("max active = %d, want %d", got, maxConcurrentSources)
	}

	got := result.configs["Concurrent"]
	if len(got) != totalSources {
		t.Fatalf("len(Concurrent) = %d, want %d", len(got), totalSources)
	}
	for i, cfg := range got {
		want := fmt.Sprintf("Proxy-%02d", i)
		if cfg.Proxy[0]["name"] != want {
			t.Fatalf("proxy[%d] = %q, want %q", i, cfg.Proxy[0]["name"], want)
		}
	}
}

func TestLoadConfigurationsReturnsConcurrentSourceError(t *testing.T) {
	restoreHTTPClient(t, &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			status := http.StatusOK
			body := `
proxies:
  - name: OK
    type: socks5
    server: 127.0.0.1
    port: 1080
`
			if strings.Contains(req.URL.RawQuery, "bad=true") {
				status = http.StatusBadGateway
				body = "bad gateway"
			}
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, err := loadConfigurations(map[string]ConfigGroup{
		"Concurrent": {
			Sources: []ConfigSource{
				{Url: "https://example.com/ok.yaml"},
				{Url: "https://example.com/bad.yaml?bad=true"},
				{Url: "https://example.com/also-ok.yaml"},
			},
		},
	}, MergeOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unexpected status 502") {
		t.Fatalf("error %q does not contain status context", err)
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

func TestMergeDedupesProxiesByNameWithinGroup(t *testing.T) {
	dir := t.TempDir()
	templateFile := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(templateFile, []byte("proxy-groups: []\nrules: []\n"), 0644); err != nil {
		t.Fatalf("write template fixture: %v", err)
	}

	firstFile := filepath.Join(dir, "first.yaml")
	if err := os.WriteFile(firstFile, []byte(`
proxies:
  - name: Duplicate
    type: ss
    server: first.example.com
    port: 8388
    cipher: aes-128-gcm
    password: change-me
  - name: Unique
    type: socks5
    server: unique.example.com
    port: 1080
`), 0644); err != nil {
		t.Fatalf("write first fixture: %v", err)
	}

	secondFile := filepath.Join(dir, "second.yaml")
	if err := os.WriteFile(secondFile, []byte(`
proxies:
  - name: Duplicate
    type: trojan
    server: second.example.com
    port: 443
    password: change-me
`), 0644); err != nil {
		t.Fatalf("write second fixture: %v", err)
	}

	cfg, err := Merge(MergeRule{
		Template: templateFile,
		Configurations: map[string]ConfigGroup{
			"Auto": {
				Sources: []ConfigSource{{Path: firstFile}, {Path: secondFile}},
			},
		},
	})
	if err != nil {
		t.Fatalf("merge dedupe: %v", err)
	}

	if got, want := len(cfg.Proxy), 2; got != want {
		t.Fatalf("len(Proxy) = %d, want %d: %#v", got, want, cfg.Proxy)
	}
	if cfg.Proxy[0]["name"] != "Duplicate" || cfg.Proxy[0]["server"] != "first.example.com" {
		t.Fatalf("first duplicate was not retained: %#v", cfg.Proxy[0])
	}

	for _, groupName := range []string{"Auto-UrlTest", "Auto"} {
		proxies := proxyGroupNames(t, cfg, groupName)
		if strings.Count(strings.Join(proxies, ","), "Duplicate") != 1 {
			t.Fatalf("group %s proxies = %#v, want one Duplicate", groupName, proxies)
		}
	}
}

func TestMergeDedupesProxiesByNameAcrossGroups(t *testing.T) {
	dir := t.TempDir()
	templateFile := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(templateFile, []byte("proxy-groups: []\nrules: []\n"), 0644); err != nil {
		t.Fatalf("write template fixture: %v", err)
	}

	leftFile := filepath.Join(dir, "left.yaml")
	if err := os.WriteFile(leftFile, []byte(`
proxies:
  - name: Shared
    type: socks5
    server: left.example.com
    port: 1080
`), 0644); err != nil {
		t.Fatalf("write left fixture: %v", err)
	}

	rightFile := filepath.Join(dir, "right.yaml")
	if err := os.WriteFile(rightFile, []byte(`
proxies:
  - name: Shared
    type: socks5
    server: right.example.com
    port: 1080
`), 0644); err != nil {
		t.Fatalf("write right fixture: %v", err)
	}

	cfg, err := Merge(MergeRule{
		Template: templateFile,
		Configurations: map[string]ConfigGroup{
			"Left":  {Sources: []ConfigSource{{Path: leftFile}}},
			"Right": {Sources: []ConfigSource{{Path: rightFile}}},
		},
	})
	if err != nil {
		t.Fatalf("merge dedupe across groups: %v", err)
	}

	if got := proxyNameCount(cfg.Proxy, "Shared"); got != 1 {
		t.Fatalf("Shared proxy count = %d, want 1: %#v", got, cfg.Proxy)
	}
	references := 0
	for _, groupName := range []string{"Left-UrlTest", "Left", "Right-UrlTest", "Right"} {
		references += proxyNameCountInList(proxyGroupNames(t, cfg, groupName), "Shared")
	}
	if references != 4 {
		t.Fatalf("Shared proxy group references = %d, want 4", references)
	}
}

func TestMergeRejectsInvalidProxyNames(t *testing.T) {
	dir := t.TempDir()
	templateFile := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(templateFile, []byte("proxy-groups: []\nrules: []\n"), 0644); err != nil {
		t.Fatalf("write template fixture: %v", err)
	}

	testCases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing",
			body: `
proxies:
  - type: socks5
    server: missing.example.com
    port: 1080
`,
			want: "missing name",
		},
		{
			name: "non-string",
			body: `
proxies:
  - name: 123
    type: socks5
    server: invalid.example.com
    port: 1080
`,
			want: "invalid proxy name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proxyFile := filepath.Join(dir, tc.name+".yaml")
			if err := os.WriteFile(proxyFile, []byte(tc.body), 0644); err != nil {
				t.Fatalf("write proxy fixture: %v", err)
			}

			_, err := Merge(MergeRule{
				Template: templateFile,
				Configurations: map[string]ConfigGroup{
					"Auto": {Sources: []ConfigSource{{Path: proxyFile}}},
				},
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

func proxyGroupNames(t *testing.T, cfg *config.RawConfig, groupName string) []string {
	t.Helper()

	for _, group := range cfg.ProxyGroup {
		if group["name"] != groupName {
			continue
		}
		proxies, ok := group["proxies"].([]string)
		if !ok {
			t.Fatalf("proxy group %q proxies type = %T", groupName, group["proxies"])
		}
		return proxies
	}
	t.Fatalf("missing proxy group %q in %#v", groupName, cfg.ProxyGroup)
	return nil
}

func proxyNameCount(proxies []map[string]any, name string) int {
	count := 0
	for _, proxy := range proxies {
		if proxy["name"] == name {
			count++
		}
	}
	return count
}

func proxyNameCountInList(proxies []string, name string) int {
	count := 0
	for _, proxy := range proxies {
		if proxy == name {
			count++
		}
	}
	return count
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
