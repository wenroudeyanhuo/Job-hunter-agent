package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTencentCareerCollectorReturnsConcreteJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tencentcareer/api/post/Query" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("keyword") == "" {
			t.Fatal("expected keyword query")
		}
		_, _ = w.Write([]byte(`{
			"Code": 200,
			"Data": {
				"Count": 1,
				"Posts": [{
					"PostId": "2076872257417949184",
					"RecruitPostName": "QQ 经典农场服务器开发工程师-新星引力计划",
					"LocationName": "深圳",
					"ProductName": "NQF-手游小程序",
					"CategoryName": "技术",
					"Responsibility": "负责游戏后端体系的维护工作，持续优化架构。",
					"PostURL": "http://careers.tencent.com/jobdesc.html?postId=2076872257417949184",
					"LastUpdateTime": "2026年07月14日"
				}]
			}
		}`))
	}))
	defer server.Close()

	collector := newTencentCareerCollector(server.URL, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect Tencent careers: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one job, got %#v", jobs)
	}
	job := jobs[0]
	if job.Company != "Tencent" {
		t.Fatalf("unexpected company %q", job.Company)
	}
	if job.Title != "QQ 经典农场服务器开发工程师-新星引力计划" {
		t.Fatalf("unexpected title %q", job.Title)
	}
	if job.City != "Shenzhen" {
		t.Fatalf("expected Shenzhen city, got %q", job.City)
	}
	if !strings.Contains(job.Description, "负责游戏后端体系") {
		t.Fatalf("expected responsibility in description, got %q", job.Description)
	}
	if job.ApplyURL != "https://careers.tencent.com/jobdesc.html?postId=2076872257417949184" {
		t.Fatalf("unexpected apply url %q", job.ApplyURL)
	}
}
