package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/app"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/config"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
)

func main() {
	cfg := config.Load()
	application, err := app.NewApplication(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if !cfg.DisableScheduler {
		stopScheduler, err := crawl.StartScheduler(ctx, application.Runner, crawl.DefaultScheduleSpecs)
		if err != nil {
			log.Fatal(err)
		}
		defer stopScheduler()
	}
	log.Printf("job-hunter-agent backend listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, application.Handler); err != nil {
		log.Fatal(err)
	}
}
