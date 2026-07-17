package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestListRunSourcesAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	run, err := repo.CreateRun(context.Background(), "manual", time.Now().UTC())
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if _, err := repo.CreateRunSourceResult(context.Background(), jobs.RunSourceResultInput{
		JobRunID:    run.ID,
		SourceName:  "example.com",
		SourceURL:   "https://example.com",
		Status:      "success",
		JobsFound:   1,
		JobsCreated: 1,
	}); err != nil {
		t.Fatalf("create source result: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/crawl/runs/"+strconv.FormatInt(run.ID, 10)+"/sources", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var results []domain.JobRunSource
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode source results: %v", err)
	}
	if len(results) != 1 || results[0].JobsCreated != 1 {
		t.Fatalf("unexpected source results: %#v", results)
	}
}
