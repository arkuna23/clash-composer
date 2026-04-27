package main

import (
	"clash-composer/composer"
	"clash-composer/config"
	"encoding/json"
	"flag"
	"os"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		println("Usage: clash-composer <command> [args]")
		return
	}

	if args[0] == "merge" {
		if len(args) != 2 {
			println("Usage: clash-composer merge <config-file>")
			return
		}

		bytes, err := os.ReadFile(args[1])
		if err != nil {
			println(err)
			return
		}

		var mergeRule composer.MergeRule
		if err := json.Unmarshal(bytes, &mergeRule); err != nil {
			println(err)
			return
		}

		merged, err := composer.Merge(mergeRule)
		if err != nil {
			println(err)
			return
		}

		bytes, err = config.MarshalRawConfig(merged)
		if err != nil {
			println(err)
			return
		}

		if err := os.WriteFile("merged.yaml", bytes, 0644); err != nil {
			println(err)
			return
		}

		println("merged.yaml written")
	}
}
