package composer

import (
	"clash-composer/config"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type RulesetStrategy string

const (
	UrlRuleset     RulesetStrategy = "url-ruleset"
	ReplaceRuleset RulesetStrategy = "replace-ruleset"
)

type MergeRule struct {
	Template             string                 `json:"template"`
	Configurations       map[string]ConfigGroup `json:"configurations"` // proxy group name: config
	RulesetStrategy      RulesetStrategy        `json:"rulesetStrategy"`
	CacheDurationSeconds int64                  `json:"cacheDurationSeconds,omitempty"`
}

type ConfigGroup struct {
	Sources       []ConfigSource `json:"sources,omitempty"`
	IncludeDirect bool           `json:"includeDirect,omitempty"`
	IncludeGroups []string       `json:"includeGroups,omitempty"`
}

func (g *ConfigGroup) UnmarshalJSON(data []byte) error {
	var sources []ConfigSource
	if err := json.Unmarshal(data, &sources); err == nil {
		*g = ConfigGroup{Sources: sources}
		return nil
	}

	type configGroup ConfigGroup
	var next configGroup
	if err := json.Unmarshal(data, &next); err != nil {
		return err
	}
	*g = ConfigGroup(next)
	return nil
}

// MergeOptions controls optional merge behavior that is not part of the JSON rule.
type MergeOptions struct {
	CommandDir string
}

func mergeProxies(template *config.RawConfig, configs []*config.RawConfig, seen map[string]bool) error {
	log.Printf("merge proxies start: sources=%d", len(configs))
	for _, cfg := range configs {
		for _, proxy := range cfg.Proxy {
			name, err := proxyName(proxy)
			if err != nil {
				return err
			}
			if seen[name] {
				log.Printf("skip duplicate proxy definition: name=%q", name)
				continue
			}
			seen[name] = true
			template.Proxy = append(template.Proxy, proxy)
		}
	}
	log.Printf("merge proxies complete: total=%d", len(template.Proxy))
	return nil
}

func proxyName(proxy map[string]any) (string, error) {
	value, ok := proxy["name"]
	if !ok {
		return "", fmt.Errorf("missing name in proxy:\n%v", proxy)
	}
	name, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid proxy name %v in proxy:\n%v", value, proxy)
	}
	return name, nil
}

func appendProxyGroup(template *config.RawConfig, name string, group ConfigGroup, configs []*config.RawConfig) error {
	log.Printf("append proxy group start: group=%q sources=%d", name, len(configs))
	length := 0
	for _, cfg := range configs {
		length += len(cfg.Proxy)
	}

	proxies := make([]string, 0, length)
	seen := map[string]bool{}
	for _, cfg := range configs {
		for _, proxy := range cfg.Proxy {
			name, err := proxyName(proxy)
			if err != nil {
				return err
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			proxies = append(proxies, name)
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
	proxiesSelect = append(proxiesSelect, proxyGroupIncludes(group)...)
	proxiesSelect = append(proxiesSelect, proxies...)
	template.ProxyGroup = append(template.ProxyGroup, map[string]any{
		"name":    name,
		"type":    "select",
		"proxies": proxiesSelect,
	})

	log.Printf("append proxy group complete: group=%q proxies=%d", name, len(proxiesSelect))
	return nil
}

func proxyGroupIncludes(group ConfigGroup) []string {
	includes := []string{}
	seen := map[string]bool{}
	if group.IncludeDirect {
		includes = append(includes, "DIRECT")
		seen["DIRECT"] = true
	}
	for _, name := range group.IncludeGroups {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		includes = append(includes, name)
		seen[name] = true
	}
	return includes
}

func Merge(rule MergeRule) (*config.RawConfig, error) {
	return MergeWithOptions(rule, MergeOptions{})
}

// MergeYAML returns a merged config plus YAML rendered from the template document.
func MergeYAML(rule MergeRule) (*config.RawConfig, []byte, error) {
	return MergeYAMLWithOptions(rule, MergeOptions{})
}

// MergeYAMLWithOptions returns merged YAML without adding fields absent from the template.
func MergeYAMLWithOptions(rule MergeRule, options MergeOptions) (*config.RawConfig, []byte, error) {
	merged, err := MergeWithOptions(rule, options)
	if err != nil {
		return nil, nil, err
	}

	data, err := renderMergedTemplateYAML(rule.Template, merged)
	if err != nil {
		return nil, nil, err
	}
	return merged, data, nil
}

func MergeWithOptions(rule MergeRule, options MergeOptions) (*config.RawConfig, error) {
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
	if err := validateGroupIncludes(rule.Configurations, newConfig.ProxyGroup); err != nil {
		return nil, err
	}

	configurations, err := loadConfigurations(rule.Configurations, options)
	if err != nil {
		log.Printf("load configurations failed: err=%v", err)
		return nil, err
	}

	seenProxyNames := map[string]bool{}
	for _, cfg := range configurations {
		if err := mergeProxies(newConfig, cfg, seenProxyNames); err != nil {
			log.Printf("merge proxies failed: err=%v", err)
			return nil, err
		}
	}

	for name, cfg := range configurations {
		if err := appendProxyGroup(newConfig, name, rule.Configurations[name], cfg); err != nil {
			log.Printf("append proxy group failed: group=%q err=%v", name, err)
			return nil, err
		}
	}

	log.Printf("merge complete: elapsed=%s proxies=%d groups=%d rules=%d", time.Since(start), len(newConfig.Proxy), len(newConfig.ProxyGroup), len(newConfig.Rule))
	return newConfig, nil
}

func validateGroupIncludes(groups map[string]ConfigGroup, templateGroups []map[string]any) error {
	available := map[string]bool{
		"DIRECT": true,
		"REJECT": true,
	}
	for name := range groups {
		available[name] = true
	}
	for _, group := range templateGroups {
		if name, ok := group["name"].(string); ok && strings.TrimSpace(name) != "" {
			available[name] = true
		}
	}

	for groupName, group := range groups {
		for _, include := range group.IncludeGroups {
			include = strings.TrimSpace(include)
			if include == "" {
				return fmt.Errorf("configuration group %q includes an empty proxy group name", groupName)
			}
			if include == groupName {
				return fmt.Errorf("configuration group %q cannot include itself", groupName)
			}
			if !available[include] {
				return fmt.Errorf("configuration group %q includes unknown proxy group %q", groupName, include)
			}
		}
	}
	return nil
}

func renderMergedTemplateYAML(templatePath string, merged *config.RawConfig) ([]byte, error) {
	doc, err := loadTemplateDocument(templatePath)
	if err != nil {
		return nil, err
	}
	root, err := documentMapping(doc)
	if err != nil {
		return nil, err
	}

	proxies, err := encodeYAMLNode(merged.Proxy)
	if err != nil {
		return nil, fmt.Errorf("encode proxies: %w", err)
	}
	setTopLevelYAMLValue(root, "proxies", proxies, "proxy-groups", "rule-providers", "rules")

	proxyGroups, err := encodeYAMLNode(merged.ProxyGroup)
	if err != nil {
		return nil, fmt.Errorf("encode proxy groups: %w", err)
	}
	setTopLevelYAMLValue(root, "proxy-groups", proxyGroups, "rule-providers", "rules")

	return yaml.Marshal(doc)
}

func encodeYAMLNode(value any) (*yaml.Node, error) {
	node := &yaml.Node{}
	if err := node.Encode(value); err != nil {
		return nil, err
	}
	return node, nil
}

func setTopLevelYAMLValue(root *yaml.Node, key string, value *yaml.Node, anchors ...string) {
	_, valueIndex := findMappingKey(root, key)
	if valueIndex >= 0 {
		root.Content[valueIndex] = value
		return
	}

	insertIndex := len(root.Content)
	for _, anchor := range anchors {
		keyIndex, _ := findMappingKey(root, anchor)
		if keyIndex >= 0 && keyIndex < insertIndex {
			insertIndex = keyIndex
		}
	}

	content := make([]*yaml.Node, 0, len(root.Content)+2)
	content = append(content, root.Content[:insertIndex]...)
	content = append(content, stringNode(key), value)
	content = append(content, root.Content[insertIndex:]...)
	root.Content = content
}
