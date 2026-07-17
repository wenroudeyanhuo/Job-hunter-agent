package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/config"
)

func TestApplicationCrawlRunImportsConfiguredSourceURLs(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Tencent Go Backend Engineer 2027 Campus - Shenzhen</title><meta name="description" content="Campus recruitment for Go backend microservices in Shenzhen."></head></html>`))
	}))
	defer source.Close()

	application, err := NewApplication(config.Config{
		DBPath:     ":memory:",
		SourceURLs: []string{source.URL},
	})
	if err != nil {
		t.Fatalf("new application: %v", err)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/crawl/run", nil)
	runRec := httptest.NewRecorder()
	application.Handler.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected crawl 200, got %d: %s", runRec.Code, runRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	listRec := httptest.NewRecorder()
	application.Handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var jobs []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("decode jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one imported job, got %d", len(jobs))
	}
	if jobs[0].Title != "Tencent Go Backend Engineer 2027 Campus - Shenzhen" {
		t.Fatalf("unexpected title %q", jobs[0].Title)
	}
}
