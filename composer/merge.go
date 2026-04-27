package composer

import (
	"clash-composer/config"
	"fmt"
	"os"
)

type RulesetStrategy string

const (
	UrlRuleset     RulesetStrategy = "url-ruleset"
	ReplaceRuleset RulesetStrategy = "replace-ruleset"
)

type MergeRule struct {
	Template        string              `json:"template"`
	Configurations  map[string][]string `json:"configurations"` // proxy group name: config
	RulesetStrategy RulesetStrategy     `json:"rulesetStrategy"`
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

func loadConfigurations(configs map[string][]string) (map[string][]*config.RawConfig, error) {
	result := make(map[string][]*config.RawConfig)
	for name, cfg := range configs {
		result[name] = make([]*config.RawConfig, 0, len(cfg))
		for _, path := range cfg {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			c, err := config.UnmarshalRawConfig(data)
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
