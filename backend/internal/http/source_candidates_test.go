package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestSourceCandidateDiscoveryAPI(t *testing.T) {
	_, handler := testRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sources/discovery/run", strings.NewReader(`{"target_cities":["Shenzhen"],"target_directions":["go"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var result jobs.SourceDiscoveryResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Created == 0 {
		t.Fatalf("expected discovered candidates, got %#v", result)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/sources/candidates?status=pending", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var candidates []jobs.SourceCandidate
	if err := json.Unmarshal(rec.Body.Bytes(), &candidates); err != nil {
		t.Fatalf("decode candidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected candidates")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sources/candidates/"+strconv.FormatInt(candidates[0].ID, 10)+"/accept", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 accept, got %d: %s", rec.Code, rec.Body.String())
	}
	var accepted struct {
		Candidate jobs.SourceCandidate `json:"candidate"`
		Source    jobs.Source          `json:"source"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode accepted: %v", err)
	}
	if accepted.Candidate.Status != jobs.SourceCandidateStatusAccepted || accepted.Source.ID == 0 {
		t.Fatalf("expected accepted candidate and source, got %#v", accepted)
	}
}

func TestRejectSourceCandidateAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.DiscoverSourceCandidates(t.Context(), jobs.SourceDiscoveryInput{TargetCities: []string{"Shenzhen"}}); err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	candidates, err := repo.ListSourceCandidates(t.Context(), jobs.SourceCandidateFilter{})
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sources/candidates/"+strconv.FormatInt(candidates[0].ID, 10)+"/reject", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 reject, got %d: %s", rec.Code, rec.Body.String())
	}
	var rejected jobs.SourceCandidate
	if err := json.Unmarshal(rec.Body.Bytes(), &rejected); err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	if rejected.Status != jobs.SourceCandidateStatusRejected {
		t.Fatalf("expected rejected candidate, got %#v", rejected)
	}
}
