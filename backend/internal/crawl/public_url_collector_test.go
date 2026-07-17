package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicURLCollectorImportsConfiguredURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Tencent Go Backend Engineer 2027 Campus - Shenzhen</title><meta name="description" content="Campus recruitment for Go backend microservices in Shenzhen."></head></html>`))
	}))
	defer server.Close()

	collector := NewPublicURLCollector([]string{server.URL}, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one job, got %d", len(jobs))
	}
	if jobs[0].Title != "Tencent Go Backend Engineer 2027 Campus - Shenzhen" {
		t.Fatalf("unexpected job title %q", jobs[0].Title)
	}
}

func TestPublicURLCollectorKeepsInvalidURLForManualCheck(t *testing.T) {
	collector := NewPublicURLCollector([]string{"not-a-url"}, nil)
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one manual-check job, got %d", len(jobs))
	}
	if jobs[0].Status != "manual_check" {
		t.Fatalf("expected manual_check, got %q", jobs[0].Status)
	}
}

func TestPublicURLCollectorImportsDiscoveredLinks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Campus Jobs</title></head><body>
<a href="/jobs/backend">Go Backend Engineer 2027 Campus</a>
<a href="/about">About us</a>
</body></html>`))
	})
	mux.HandleFunc("/jobs/backend", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Tencent Go Backend Engineer 2027 Campus - Shenzhen</title><meta name="description" content="Go backend role in Shenzhen."></head></html>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	collector := NewPublicURLCollector([]string{server.URL}, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("expected source page and discovered job, got %d", len(jobs))
	}
	found := false
	for _, job := range jobs {
		if job.Title == "Tencent Go Backend Engineer 2027 Campus - Shenzhen" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected discovered backend job, got %#v", jobs)
	}
}
