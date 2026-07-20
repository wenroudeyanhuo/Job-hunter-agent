package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMeituanCareerCollectorReturnsConcreteJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/official/job/getJobList" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		_, _ = w.Write([]byte(`{
			"data": {
				"list": [{
					"jobUnionId": "3300336612",
					"name": "Merchant Platform Java Engineer",
					"jobFamily": "Technology",
					"jobFamilyGroup": "Software",
					"cityList": [{"name": "Shenzhen"}],
					"department": [{"name": "Core Local Business"}],
					"jobDuty": "Build backend services for merchant platforms.",
					"jobRequirement": "Strong Java or Go backend experience.",
					"highLight": "AI-enabled merchant service platform.",
					"refreshTime": 1748858517000
				}]
			}
		}`))
	}))
	defer server.Close()

	collector := newMeituanCareerCollector(server.URL, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect Meituan careers: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one job, got %#v", jobs)
	}
	job := jobs[0]
	if job.Company != "Meituan" {
		t.Fatalf("unexpected company %q", job.Company)
	}
	if job.Title != "Merchant Platform Java Engineer" {
		t.Fatalf("unexpected title %q", job.Title)
	}
	if job.City != "Shenzhen" {
		t.Fatalf("expected Shenzhen city, got %q", job.City)
	}
	if !strings.Contains(job.Description, "backend services") || !strings.Contains(job.Description, "Core Local Business") {
		t.Fatalf("expected rich description, got %q", job.Description)
	}
	if job.ApplyURL != server.URL+"/web/position/detail?jobUnionId=3300336612" {
		t.Fatalf("unexpected apply url %q", job.ApplyURL)
	}
	if job.PublishedAt == nil {
		t.Fatal("expected refresh time to be mapped")
	}
}
