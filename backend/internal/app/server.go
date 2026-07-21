package app

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/config"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	httpapi "github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/http"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

type Application struct {
	Handler    http.Handler
	DB         *sql.DB
	Runner     crawl.Runnable
	Automation *automationRunner
}

func NewServer() http.Handler {
	cfg := config.Load()
	application, err := NewApplication(cfg)
	if err != nil {
		panic(err)
	}
	return application.Handler
}

func NewApplication(cfg config.Config) (*Application, error) {
	if dir := filepath.Dir(cfg.DBPath); dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	repo := jobs.NewRepository(conn)
	if len(cfg.SourceURLs) > 0 {
		if err := repo.SeedPublicURLSources(context.Background(), cfg.SourceURLs); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	collectors := []crawl.Collector{crawl.SeedCollector{}, crawl.NewDBSourceCollector(repo, nil)}
	baseRunner := crawl.NewRunner(repo, collectors)
	runner := newNotifyingRunner(baseRunner, repo, cfg.FeishuWebhookURL)
	automation := newAutomationRunner(repo, cfg.FeishuWebhookURL)
	handler := httpapi.NewRouter(&httpapi.Handlers{
		Repo:             repo,
		Runner:           runner,
		FeishuWebhookURL: cfg.FeishuWebhookURL,
	})
	return &Application{Handler: handler, DB: conn, Runner: runner, Automation: automation}, nil
}
