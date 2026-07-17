package main

import (
	"log"
	"net/http"
	"os"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/app"
)

func main() {
	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("job-hunter-agent backend listening on %s", addr)
	if err := http.ListenAndServe(addr, app.NewServer()); err != nil {
		log.Fatal(err)
	}
}
