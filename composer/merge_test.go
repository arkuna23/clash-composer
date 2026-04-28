package composer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

	configs, err := loadConfigurations(map[string][]ConfigSource{
		"Mixed": {
			{Path: pathFile},
			{Url: "https://example.com/url.yaml"},
			{Cmd: fmt.Sprintf("cat %q", cmdFile)},
		},
	})
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
			_, err := loadConfigSource(tc.src)
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
