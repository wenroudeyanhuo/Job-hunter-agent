package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestApplicationPlansAPIListsAndUpdatesPlans(t *testing.T) {
	repo, handler := testRouter(t, nil)
	_, err := repo.CreateJob(t.Context(), domain.Job{
		Company:      "Tencent",
		Title:        "Go Backend Engineer",
		City:         "Shenzhen",
		ApplyURL:     "https://careers.example.com/jobs/1",
		DiscoveredAt: time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC),
		MatchScore:   88,
		Status:       domain.StatusInterested,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/applications/sync", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 sync, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/applications", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d: %s", rec.Code, rec.Body.String())
	}
	var plans []jobs.ApplicationPlan
	if err := json.Unmarshal(rec.Body.Bytes(), &plans); err != nil {
		t.Fatalf("decode plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one plan, got %#v", plans)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/applications/"+strconv.FormatInt(plans[0].ID, 10), strings.NewReader(`{"status":"ready","next_action":"Submit after resume review","checklist":["Resume ready"],"target_apply_date":"2026-07-23"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 update, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated jobs.ApplicationPlan
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated: %v", err)
	}
	if updated.Status != jobs.ApplicationPlanStatusReady || updated.NextAction != "Submit after resume review" {
		t.Fatalf("unexpected updated plan: %#v", updated)
	}
}
