package main

import (
	"clash-composer/composer"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
)

var version = "v0.1.0"

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "version":
		log.Printf("command start: version")
		_, _ = os.Stdout.WriteString(version + "\n")
	case "serve":
		log.Printf("command start: serve args=%v", args[1:])
		serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)
		addr := serveFlags.String("addr", "127.0.0.1:8080", "HTTP listen address")
		configDir := serveFlags.String("config-dir", "", "directory for API-managed merge config files")
		token := serveFlags.String("token", "", "HTTP API token; defaults to CLASH_COMPOSER_TOKEN")
		if err := serveFlags.Parse(args[1:]); err != nil {
			log.Printf("parse serve flags failed: %v", err)
			return
		}
		if serveFlags.NArg() != 0 {
			log.Println("Usage: clash-composer serve -config-dir <dir> [-addr 127.0.0.1:8080] [-token <token>]")
			return
		}
		if err := composer.ServeHTTPAPI(composer.ServeOptions{
			Addr:      *addr,
			ConfigDir: *configDir,
			Token:     *token,
			WebappFS:  webappFS(),
		}); err != nil {
			log.Printf("serve failed: %v", err)
		}
	case "merge":
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

		mergeFile, err := filepath.Abs(args[1])
		if err != nil {
			log.Printf("resolve merge rule path failed: %v", err)
			return
		}

		log.Printf("merge execution start: %s", args[1])
		merged, bytes, err := composer.MergeYAMLWithOptions(mergeRule, composer.MergeOptions{
			CommandDir: filepath.Dir(mergeFile),
		})
		if err != nil {
			log.Printf("merge execution failed: %v", err)
			return
		}
		log.Printf("merge execution complete: proxies=%d groups=%d rules=%d", len(merged.Proxy), len(merged.ProxyGroup), len(merged.Rule))

		log.Printf("marshal merged config complete: bytes=%d", len(bytes))

		log.Println("write merged config start: merged.yaml")
		if err := os.WriteFile("merged.yaml", bytes, 0644); err != nil {
			log.Printf("write merged config failed: %v", err)
			return
		}

		log.Println("write merged config complete: merged.yaml")
	case "download":
		log.Printf("command start: download args=%v", args[1:])
		if len(args) != 2 {
			log.Println("Usage: clash-composer download <subscription-url>")
			return
		}

		subscriptionURL := args[1]
		log.Printf("download start: %s", subscriptionURL)
		bytes, err := composer.DownloadURL(subscriptionURL)
		if err != nil {
			log.Printf("download failed: %v", err)
			return
		}
		log.Printf("download complete: bytes=%d", len(bytes))
		if _, err := os.Stdout.Write(bytes); err != nil {
			log.Printf("write stdout failed: %v", err)
		}
	default:
		log.Printf("unknown command: %s", args[0])
		printUsage()
	}
}

func printUsage() {
	log.Printf("clash-composer %s", version)
	log.Println("Usage: clash-composer <command> [args]")
	log.Println("Commands: version, serve -config-dir <dir> [-addr 127.0.0.1:8080] [-token <token>], merge <config-file>, download <subscription-url>")
}
