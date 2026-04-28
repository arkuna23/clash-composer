package composer

import (
	"clash-composer/config"
	"context"
	"fmt"
	"io"
	"log"
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
	log.Printf("merge proxies start: sources=%d", len(configs))
	for _, cfg := range configs {
		for _, proxy := range cfg.Proxy {
			template.Proxy = append(template.Proxy, proxy)
		}
	}
	log.Printf("merge proxies complete: total=%d", len(template.Proxy))
}

func appendProxyGroup(template *config.RawConfig, name string, configs []*config.RawConfig) error {
	log.Printf("append proxy group start: group=%q sources=%d", name, len(configs))
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

	log.Printf("append proxy group complete: group=%q proxies=%d", name, len(proxiesSelect))
	return nil
}

func loadConfigSource(source ConfigSource) (cfg *config.RawConfig, err error) {
	hasPath := source.Path != ""
	hasURL := source.Url != ""
	hasCmd := source.Cmd != ""
	label := sourceLabel(source)
	start := time.Now()
	log.Printf("load config source start: %s", label)
	defer func() {
		if err != nil {
			log.Printf("load config source failed: %s elapsed=%s err=%v", label, time.Since(start), err)
			return
		}
		log.Printf("load config source complete: %s elapsed=%s proxies=%d", label, time.Since(start), len(cfg.Proxy))
	}()

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

	var data []byte
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

	cfg, err = config.UnmarshalRawConfig(data)
	return cfg, err
}

func loadConfigurations(configs map[string][]ConfigSource) (map[string][]*config.RawConfig, error) {
	log.Printf("load configurations start: groups=%d", len(configs))
	result := make(map[string][]*config.RawConfig)
	for name, cfg := range configs {
		groupStart := time.Now()
		log.Printf("load configuration group start: group=%q sources=%d", name, len(cfg))
		result[name] = make([]*config.RawConfig, 0, len(cfg))
		for _, source := range cfg {
			c, err := loadConfigSource(source)
			if err != nil {
				return nil, err
			}
			result[name] = append(result[name], c)
		}
		log.Printf("load configuration group complete: group=%q elapsed=%s configs=%d", name, time.Since(groupStart), len(result[name]))
	}
	log.Printf("load configurations complete: groups=%d", len(result))
	return result, nil
}

func Merge(rule MergeRule) (*config.RawConfig, error) {
	start := time.Now()
	log.Printf("merge start: template=%q groups=%d strategy=%s", rule.Template, len(rule.Configurations), rule.RulesetStrategy)
	log.Printf("read template start: %s", rule.Template)
	template, err := os.ReadFile(rule.Template)
	if err != nil {
		log.Printf("read template failed: %s err=%v", rule.Template, err)
		return nil, err
	}
	log.Printf("read template complete: %s bytes=%d", rule.Template, len(template))

	log.Printf("parse template start: %s", rule.Template)
	newConfig, err := config.UnmarshalRawConfig(template)
	if err != nil {
		log.Printf("parse template failed: %s err=%v", rule.Template, err)
		return nil, err
	}
	log.Printf("parse template complete: %s proxies=%d rules=%d", rule.Template, len(newConfig.Proxy), len(newConfig.Rule))

	configurations, err := loadConfigurations(rule.Configurations)
	if err != nil {
		log.Printf("load configurations failed: err=%v", err)
		return nil, err
	}

	for _, cfg := range configurations {
		mergeProxies(newConfig, cfg)
	}

	for name, cfg := range configurations {
		if err := appendProxyGroup(newConfig, name, cfg); err != nil {
			log.Printf("append proxy group failed: group=%q err=%v", name, err)
			return nil, err
		}
	}

	log.Printf("merge complete: elapsed=%s proxies=%d groups=%d rules=%d", time.Since(start), len(newConfig.Proxy), len(newConfig.ProxyGroup), len(newConfig.Rule))
	return newConfig, nil
}

func sourceLabel(source ConfigSource) string {
	switch {
	case source.Path != "":
		return fmt.Sprintf("path=%q", source.Path)
	case source.Url != "":
		return fmt.Sprintf("url=%q", source.Url)
	case source.Cmd != "":
		return summarizeCmd(source.Cmd)
	default:
		return "empty-source"
	}
}

func summarizeCmd(cmd string) string {
	const maxPreview = 32
	cleaned := strings.TrimSpace(cmd)
	if len(cleaned) > maxPreview {
		cleaned = cleaned[:maxPreview] + "..."
	}
	return fmt.Sprintf("cmd(len=%d preview=%q)", len(cmd), cleaned)
}
