package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestByteDanceCareerCollectorReturnsConcreteJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/job/posts" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": {
				"count": 1,
				"job_post_list": [{
					"id": "7525009396952582407",
					"title": "后端开发工程师-飞书-2027届校招",
					"city_info": {"name": "深圳"},
					"description": "负责飞书业务后端服务开发，参与高并发系统建设。",
					"requirement": "熟悉 Go 或 Java，具备扎实的数据结构和算法基础。"
				}]
			}
		}`))
	}))
	defer server.Close()

	collector := newByteDanceCareerCollector(server.URL, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect ByteDance careers: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one job, got %#v", jobs)
	}
	job := jobs[0]
	if job.Company != "ByteDance" {
		t.Fatalf("unexpected company %q", job.Company)
	}
	if job.Title != "后端开发工程师-飞书-2027届校招" {
		t.Fatalf("unexpected title %q", job.Title)
	}
	if job.City != "Shenzhen" {
		t.Fatalf("expected Shenzhen city, got %q", job.City)
	}
	if !strings.Contains(job.Description, "高并发") || !strings.Contains(job.Description, "Go") {
		t.Fatalf("expected description and requirement, got %q", job.Description)
	}
	if !strings.HasSuffix(job.ApplyURL, "/campus/position/7525009396952582407") {
		t.Fatalf("unexpected apply url %q", job.ApplyURL)
	}
}
