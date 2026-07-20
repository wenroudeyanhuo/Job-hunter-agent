package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
)

func TestSourcesAPI(t *testing.T) {
	_, handler := testRouter(t, nil)

	createBody := bytes.NewBufferString(`{"name":"Tencent Campus","url":"https://example.com/tencent","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sources", createBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID       int64  `json:"id"`
		Enabled  bool   `json:"enabled"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created source: %v", err)
	}
	if created.ID == 0 || !created.Enabled {
		t.Fatalf("unexpected created source: %#v", created)
	}
	if created.Category != "general" {
		t.Fatalf("expected default category, got %#v", created)
	}

	toggleBody := bytes.NewBufferString(`{"enabled":false}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/sources/"+strconv.FormatInt(created.ID, 10), toggleBody)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var sources []struct {
		ID      int64 `json:"id"`
		Enabled bool  `json:"enabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sources); err != nil {
		t.Fatalf("decode sources: %v", err)
	}
	if len(sources) != 1 || sources[0].Enabled {
		t.Fatalf("expected one disabled source, got %#v", sources)
	}
}

func TestSeedRecommendedSourcesAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sources/recommended", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Created int `json:"created"`
		Total   int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Created == 0 || response.Total == 0 {
		t.Fatalf("expected recommended sources to be created, got %#v", response)
	}

	sources, err := repo.ListSources(req.Context(), false)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != response.Total {
		t.Fatalf("expected %d sources, got %d", response.Total, len(sources))
	}
}

func TestRecommendedCrawlSeedsSourcesAndRuns(t *testing.T) {
	repo, handler := testRouter(t, fakeRunner{summary: crawl.RunSummary{SourcesSuccess: 1}})

	req := httptest.NewRequest(http.MethodPost, "/api/crawl/recommended", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Seeded  int              `json:"seeded"`
		Summary crawl.RunSummary `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Seeded == 0 || response.Summary.SourcesSuccess != 1 {
		t.Fatalf("unexpected recommended crawl response: %#v", response)
	}

	sources, err := repo.ListSources(req.Context(), false)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != response.Seeded {
		t.Fatalf("expected seeded sources to be stored, got %d vs %d", len(sources), response.Seeded)
	}
}
