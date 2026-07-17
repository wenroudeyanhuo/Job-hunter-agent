package app

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/config"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	httpapi "github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/http"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func NewServer() http.Handler {
	cfg := config.Load()
	if dir := filepath.Dir(cfg.DBPath); dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		panic(err)
	}
	repo := jobs.NewRepository(conn)
	runner := crawl.NewRunner(repo, []crawl.Collector{crawl.SeedCollector{}})
	return httpapi.NewRouter(&httpapi.Handlers{
		Repo:             repo,
		Runner:           runner,
		FeishuWebhookURL: cfg.FeishuWebhookURL,
	})
}
