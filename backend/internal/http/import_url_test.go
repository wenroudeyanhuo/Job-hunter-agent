package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestImportURLCreatesScoredJob(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Tencent Go Backend Engineer 2027 Campus - Shenzhen</title><meta name="description" content="Campus recruitment for Go backend microservices in Shenzhen."></head></html>`))
	}))
	defer source.Close()

	repo, handler := testRouter(t, nil)
	body, _ := json.Marshal(map[string]string{"url": source.URL})
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/import-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Job        domain.Job `json:"job"`
		Duplicate  bool       `json:"duplicate"`
		ManualOnly bool       `json:"manual_only"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Job.ID == 0 || response.Job.MatchScore == 0 {
		t.Fatalf("expected persisted scored job, got %#v", response.Job)
	}

	list, err := repo.ListJobs(req.Context(), jobs.ListFilter{})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one job, got %d", len(list))
	}
}

func TestImportURLRejectsBadPayload(t *testing.T) {
	_, handler := testRouter(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/import-url", bytes.NewReader([]byte(`{"url":"not-a-url"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
