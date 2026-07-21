package httpapi

import (
	"bytes"
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

func TestProfileAPIPersistsUpdates(t *testing.T) {
	_, handler := testRouter(t, nil)
	body := bytes.NewBufferString(`{
		"target_cities":["Shenzhen","Guangzhou"],
		"target_directions":["backend","go"],
		"skills":["Go","React"],
		"education":"本科",
		"graduation_year":"2027",
		"internship_preference":"accept_conversion_clear",
		"preferred_companies":["Tencent"],
		"blocked_keywords":["外包","培训"],
		"notes":"Prefer backend platform roles."
	}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/profile", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var profile jobs.CandidateProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if len(profile.TargetDirections) != 2 || profile.TargetDirections[1] != "go" {
		t.Fatalf("unexpected profile: %#v", profile)
	}
	if profile.Notes != "Prefer backend platform roles." {
		t.Fatalf("expected notes to round trip, got %q", profile.Notes)
	}
}

func TestJobDetailAPIIncludesFitAndDecisions(t *testing.T) {
	repo, handler := testRouter(t, nil)
	ctx := context.Background()
	if _, err := repo.SaveCandidateProfile(ctx, jobs.CandidateProfile{
		TargetCities:       []string{"Shenzhen"},
		TargetDirections:   []string{"backend", "go"},
		Skills:             []string{"Go"},
		PreferredCompanies: []string{"Tencent"},
		BlockedKeywords:    []string{"外包"},
	}); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	job, err := repo.CreateJob(ctx, domain.Job{
		Company:       "Tencent",
		Title:         "Go Backend Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"backend", "go"},
		ApplyURL:      "https://example.com/apply",
		DiscoveredAt:  time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC),
		MatchScore:    82,
		Status:        domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := repo.UpdateStatus(ctx, job.ID, domain.StatusInterested); err != nil {
		t.Fatalf("update status: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/jobs/"+strconv.FormatInt(job.ID, 10)+"/detail", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var detail jobs.JobDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.Job.ID != job.ID || len(detail.Decisions) != 1 {
		t.Fatalf("unexpected detail response: %#v", detail)
	}
	if detail.Fit.Score <= job.MatchScore {
		t.Fatalf("expected profile-aware fit score, got %#v", detail.Fit)
	}
}
