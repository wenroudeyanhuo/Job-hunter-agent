package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestListJobsAndUpdateStatus(t *testing.T) {
	repo, handler := testRouter(t, nil)
	created, err := repo.CreateJob(context.Background(), domain.Job{
		Company:    "Tencent",
		Title:      "Backend Engineer",
		City:       "Shenzhen",
		ApplyURL:   "https://example.com/apply",
		Status:     domain.StatusNew,
		MatchScore: 88,
	})
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}

	body := bytes.NewBufferString(`{"status":"interested"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+strconv.FormatInt(created.ID, 10)+"/status", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/jobs?status=interested", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var list []domain.Job
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode jobs: %v", err)
	}
	if len(list) != 1 || list[0].Status != domain.StatusInterested {
		t.Fatalf("unexpected jobs response: %#v", list)
	}
}

func TestListJobsReturnsEmptyArray(t *testing.T) {
	_, handler := testRouter(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "[]" {
		t.Fatalf("expected empty JSON array, got %q", rec.Body.String())
	}
}

func TestRunCrawlReturnsSummary(t *testing.T) {
	_, handler := testRouter(t, fakeRunner{summary: crawl.RunSummary{JobsCreated: 1}})

	req := httptest.NewRequest(http.MethodPost, "/api/crawl/run", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var summary crawl.RunSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.JobsCreated != 1 {
		t.Fatalf("expected jobs_created 1, got %d", summary.JobsCreated)
	}
}

func testRouter(t *testing.T, runner CrawlRunner) (*jobs.Repository, http.Handler) {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	if runner == nil {
		runner = crawl.NewRunner(repo, []crawl.Collector{crawl.SeedCollector{}})
	}
	handler := NewRouter(&Handlers{Repo: repo, Runner: runner})
	return repo, handler
}

type fakeRunner struct {
	summary crawl.RunSummary
	err     error
}

func (f fakeRunner) Run(context.Context, string) (crawl.RunSummary, error) {
	return f.summary, f.err
}
