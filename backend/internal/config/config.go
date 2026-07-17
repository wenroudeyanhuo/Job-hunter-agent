package config

import (
	"os"
	"strings"
)

type Config struct {
	Addr             string
	DBPath           string
	FeishuWebhookURL string
	DisableScheduler bool
	SourceURLs       []string
}

func Load() Config {
	cfg := Config{
		Addr:             os.Getenv("APP_ADDR"),
		DBPath:           os.Getenv("APP_DB_PATH"),
		FeishuWebhookURL: os.Getenv("FEISHU_WEBHOOK_URL"),
		DisableScheduler: os.Getenv("DISABLE_SCHEDULER") == "1",
		SourceURLs:       parseSourceURLs(os.Getenv("SOURCE_URLS")),
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "data/job-hunter-agent.db"
	}
	return cfg
}

func parseSourceURLs(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t'
	})
	out := []string{}
	seen := map[string]bool{}
	for _, field := range fields {
		value := strings.TrimSpace(field)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
