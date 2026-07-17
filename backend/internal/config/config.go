package config

import "os"

type Config struct {
	Addr             string
	DBPath           string
	FeishuWebhookURL string
	DisableScheduler bool
}

func Load() Config {
	cfg := Config{
		Addr:             os.Getenv("APP_ADDR"),
		DBPath:           os.Getenv("APP_DB_PATH"),
		FeishuWebhookURL: os.Getenv("FEISHU_WEBHOOK_URL"),
		DisableScheduler: os.Getenv("DISABLE_SCHEDULER") == "1",
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "data/job-hunter-agent.db"
	}
	return cfg
}
