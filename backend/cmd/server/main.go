package main

import (
	"log"
	"net/http"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/app"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("job-hunter-agent backend listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, app.NewServer()); err != nil {
		log.Fatal(err)
	}
}
