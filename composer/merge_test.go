package composer

import (
	"encoding/json"
	"os"
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
