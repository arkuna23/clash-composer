package composer

import (
	"clash-composer/config"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type RulesetStrategy string

const (
	UrlRuleset     RulesetStrategy = "url-ruleset"
	ReplaceRuleset RulesetStrategy = "replace-ruleset"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

type ConfigSource struct {
	Path string `json:"path"`
	Url  string `json:"url"`
	Cmd  string `json:"cmd"`
}

type MergeRule struct {
	Template        string                    `json:"template"`
	Configurations  map[string][]ConfigSource `json:"configurations"` // proxy group name: config
	RulesetStrategy RulesetStrategy           `json:"rulesetStrategy"`
}

func mergeProxies(template *config.RawConfig, configs []*config.RawConfig) {
	for _, cfg := range configs {
		for _, proxy := range cfg.Proxy {
			template.Proxy = append(template.Proxy, proxy)
		}
	}
}

func appendProxyGroup(template *config.RawConfig, name string, configs []*config.RawConfig) error {
	length := 0
	for _, cfg := range configs {
		length += len(cfg.Proxy)
	}

	proxies := make([]string, 0, length)
	for _, cfg := range configs {
		for _, proxy := range cfg.Proxy {
			if proxy["name"] != nil {
				proxies = append(proxies, proxy["name"].(string))
			} else {
				return fmt.Errorf("missing name in proxy:\n%v", proxy)
			}
		}
	}
	template.ProxyGroup = append(template.ProxyGroup, map[string]any{
		"name":      name + "-UrlTest",
		"type":      "url-test",
		"url":       "http://www.gstatic.com/generate_204",
		"interval":  300,
		"tolerance": 50,
		"proxies":   proxies,
	})

	proxiesSelect := make([]string, 0, len(proxies)+1)
	proxiesSelect = append(proxiesSelect, proxies...)
	proxiesSelect = append(proxiesSelect, name+"-UrlTest")
	template.ProxyGroup = append(template.ProxyGroup, map[string]any{
		"name":    name,
		"type":    "select",
		"proxies": proxiesSelect,
	})

	return nil
}

func loadConfigSource(source ConfigSource) (*config.RawConfig, error) {
	hasPath := source.Path != ""
	hasURL := source.Url != ""
	hasCmd := source.Cmd != ""

	selected := 0
	if hasPath {
		selected++
	}
	if hasURL {
		selected++
	}
	if hasCmd {
		selected++
	}
	if selected != 1 {
		return nil, fmt.Errorf("config source must set exactly one of path, url, or cmd: %+v", source)
	}

	var (
		data []byte
		err  error
	)
	if hasPath {
		data, err = os.ReadFile(source.Path)
		if err != nil {
			return nil, err
		}
	} else if hasURL {
		resp, err := httpClient.Get(source.Url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("fetch %s: unexpected status %d", source.Url, resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "bash", "-lc", source.Cmd)
		data, err = cmd.Output()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("run %q: command timed out", source.Cmd)
			}
			if exitErr, ok := err.(*exec.ExitError); ok {
				stderr := strings.TrimSpace(string(exitErr.Stderr))
				if stderr != "" {
					return nil, fmt.Errorf("run %q: %w: %s", source.Cmd, err, stderr)
				}
			}
			return nil, fmt.Errorf("run %q: %w", source.Cmd, err)
		}
	}

	return config.UnmarshalRawConfig(data)
}

func loadConfigurations(configs map[string][]ConfigSource) (map[string][]*config.RawConfig, error) {
	result := make(map[string][]*config.RawConfig)
	for name, cfg := range configs {
		result[name] = make([]*config.RawConfig, 0, len(cfg))
		for _, source := range cfg {
			c, err := loadConfigSource(source)
			if err != nil {
				return nil, err
			}
			result[name] = append(result[name], c)
		}
	}
	return result, nil
}

func Merge(rule MergeRule) (*config.RawConfig, error) {
	template, err := os.ReadFile(rule.Template)
	if err != nil {
		return nil, err
	}

	newConfig, err := config.UnmarshalRawConfig(template)
	if err != nil {
		return nil, err
	}

	configurations, err := loadConfigurations(rule.Configurations)
	if err != nil {
		return nil, err
	}

	for _, cfg := range configurations {
		mergeProxies(newConfig, cfg)
	}

	for name, cfg := range configurations {
		if err := appendProxyGroup(newConfig, name, cfg); err != nil {
			return nil, err
		}
	}

	return newConfig, nil
}
