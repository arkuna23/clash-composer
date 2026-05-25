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

const (
	defaultDownloadTimeout = 5 * time.Second
	clashVergeUserAgent    = "clash-verge/v2.0.4"
)

var httpClient = &http.Client{Timeout: defaultDownloadTimeout}

type ConfigSource struct {
	Path string `json:"path"`
	Url  string `json:"url"`
	Cmd  string `json:"cmd"`
}

func loadConfigSource(source ConfigSource, options MergeOptions) (cfg *config.RawConfig, err error) {
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

	data, err := downloadConfigSource(source, options)
	if err != nil {
		return nil, err
	}

	cfg, err = config.UnmarshalRawConfig(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", label, err)
	}
	return cfg, nil
}

func downloadConfigSource(source ConfigSource, options MergeOptions) ([]byte, error) {
	if err := validateConfigSource(source); err != nil {
		return nil, err
	}

	switch {
	case source.Path != "":
		return os.ReadFile(source.Path)
	case source.Url != "":
		return downloadHTTPConfig(source.Url)
	default:
		return downloadCommandConfig(source.Cmd, options.CommandDir)
	}
}

func validateConfigSource(source ConfigSource) error {
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
		return fmt.Errorf("config source must set exactly one of path, url, or cmd: %+v", source)
	}

	return nil
}

func downloadHTTPConfig(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request %s: %w", url, err)
	}
	req.Header.Set("User-Agent", clashVergeUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch %s: unexpected status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DownloadURL(url string) ([]byte, error) {
	return downloadHTTPConfig(url)
}

func downloadCommandConfig(command string, dir string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultDownloadTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	if dir != "" {
		cmd.Dir = dir
	}
	data, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("run %q: command timed out", command)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("run %q: %w: %s", command, err, stderr)
			}
		}
		return nil, fmt.Errorf("run %q: %w", command, err)
	}

	return data, nil
}

func loadConfigurations(configs map[string][]ConfigSource, options MergeOptions) (map[string][]*config.RawConfig, error) {
	log.Printf("load configurations start: groups=%d", len(configs))
	result := make(map[string][]*config.RawConfig)
	for name, cfg := range configs {
		groupStart := time.Now()
		log.Printf("load configuration group start: group=%q sources=%d", name, len(cfg))
		result[name] = make([]*config.RawConfig, 0, len(cfg))
		for _, source := range cfg {
			c, err := loadConfigSource(source, options)
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
