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
