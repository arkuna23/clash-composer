package composer

import (
	"clash-composer/config"
	"fmt"
	"log"
	"os"
	"time"
)

type RulesetStrategy string

const (
	UrlRuleset     RulesetStrategy = "url-ruleset"
	ReplaceRuleset RulesetStrategy = "replace-ruleset"
)

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
	proxiesSelect = append(proxiesSelect, name+"-UrlTest")
	proxiesSelect = append(proxiesSelect, proxies...)
	template.ProxyGroup = append(template.ProxyGroup, map[string]any{
		"name":    name,
		"type":    "select",
		"proxies": proxiesSelect,
	})

	log.Printf("append proxy group complete: group=%q proxies=%d", name, len(proxiesSelect))
	return nil
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
