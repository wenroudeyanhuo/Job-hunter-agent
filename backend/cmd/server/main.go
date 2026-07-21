package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		stopAutomation, err := crawl.StartScheduledFunc(ctx, crawl.DefaultAutomationSpecs, func(ctx context.Context) {
			if application.Automation == nil {
				return
			}
			if _, err := application.Automation.Tick(ctx, time.Now().UTC()); err != nil {
				log.Printf("run automation tick: %v", err)
			}
		})
		if err != nil {
			log.Fatal(err)
		}
		defer stopAutomation()
	}
	log.Printf("job-hunter-agent backend listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, application.Handler); err != nil {
		log.Fatal(err)
	}
}
