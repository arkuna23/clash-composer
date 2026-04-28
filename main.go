package main

import (
	"clash-composer/composer"
	"clash-composer/config"
	"encoding/json"
	"flag"
	"log"
	"os"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		log.Println("Usage: clash-composer <command> [args]")
		return
	}

	if args[0] == "merge" {
		log.Printf("command start: merge args=%v", args[1:])
		if len(args) != 2 {
			log.Println("Usage: clash-composer merge <config-file>")
			return
		}

		log.Printf("read merge rule start: %s", args[1])
		bytes, err := os.ReadFile(args[1])
		if err != nil {
			log.Printf("read merge rule failed: %v", err)
			return
		}
		log.Printf("read merge rule complete: %s bytes=%d", args[1], len(bytes))

		var mergeRule composer.MergeRule
		log.Printf("parse merge rule start: %s", args[1])
		if err := json.Unmarshal(bytes, &mergeRule); err != nil {
			log.Printf("parse merge rule failed: %v", err)
			return
		}
		log.Printf("parse merge rule complete: template=%q groups=%d", mergeRule.Template, len(mergeRule.Configurations))

		log.Printf("merge execution start: %s", args[1])
		merged, err := composer.Merge(mergeRule)
		if err != nil {
			log.Printf("merge execution failed: %v", err)
			return
		}
		log.Printf("merge execution complete: proxies=%d groups=%d rules=%d", len(merged.Proxy), len(merged.ProxyGroup), len(merged.Rule))

		log.Println("marshal merged config start")
		bytes, err = config.MarshalRawConfig(merged)
		if err != nil {
			log.Printf("marshal merged config failed: %v", err)
			return
		}
		log.Printf("marshal merged config complete: bytes=%d", len(bytes))

		log.Println("write merged config start: merged.yaml")
		if err := os.WriteFile("merged.yaml", bytes, 0644); err != nil {
			log.Printf("write merged config failed: %v", err)
			return
		}

		log.Println("write merged config complete: merged.yaml")
	}
}
