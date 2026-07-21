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
	LLMAPIKey        string
	LLMBaseURL       string
	LLMModel         string
}

func Load() Config {
	cfg := Config{
		Addr:             os.Getenv("APP_ADDR"),
		DBPath:           os.Getenv("APP_DB_PATH"),
		FeishuWebhookURL: os.Getenv("FEISHU_WEBHOOK_URL"),
		DisableScheduler: os.Getenv("DISABLE_SCHEDULER") == "1",
		SourceURLs:       parseSourceURLs(os.Getenv("SOURCE_URLS")),
		LLMAPIKey:        firstNonEmpty(os.Getenv("LLM_API_KEY"), os.Getenv("OPENAI_API_KEY")),
		LLMBaseURL:       firstNonEmpty(os.Getenv("LLM_BASE_URL"), os.Getenv("OPENAI_BASE_URL")),
		LLMModel:         firstNonEmpty(os.Getenv("LLM_MODEL"), os.Getenv("OPENAI_MODEL")),
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "data/job-hunter-agent.db"
	}
	if cfg.LLMBaseURL == "" {
		cfg.LLMBaseURL = "https://api.openai.com/v1"
	}
	return cfg
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
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
