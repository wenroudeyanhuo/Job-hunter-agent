package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOPPOCareerCollectorReturnsConcreteJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openapi/position/pageNew" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": {
				"records": [{
					"idProjPosition": 1745,
					"positionName": "Senior AI Researcher",
					"workCityName": "Shenzhen",
					"positionDesc": "Build AI agents for personalized content services.",
					"positionRequire": "Strong NLP and recommendation background.",
					"releaseTime": "2026-07-01 10:00:00",
					"recruitmentTypeName": "Campus",
					"projectName": "2027"
				}],
				"total": 1
			},
			"msg": "success"
		}`))
	}))
	defer server.Close()

	collector := newOPPOCareerCollector(server.URL, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect OPPO careers: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one job, got %#v", jobs)
	}
	job := jobs[0]
	if job.Company != "OPPO" {
		t.Fatalf("unexpected company %q", job.Company)
	}
	if job.Title != "Senior AI Researcher" {
		t.Fatalf("unexpected title %q", job.Title)
	}
	if job.City != "Shenzhen" {
		t.Fatalf("expected Shenzhen city, got %q", job.City)
	}
	if !strings.Contains(job.Description, "AI agents") || !strings.Contains(job.Description, "NLP") {
		t.Fatalf("expected combined description, got %q", job.Description)
	}
	if job.ApplyURL != server.URL+"/university/oppo/campus/post/1745" {
		t.Fatalf("unexpected apply url %q", job.ApplyURL)
	}
	if job.PublishedAt == nil {
		t.Fatal("expected published time")
	}
}
