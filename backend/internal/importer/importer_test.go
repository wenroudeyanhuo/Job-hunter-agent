package importer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImportURLParsesTitleAndDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
<head>
  <title>Tencent Go Backend Engineer 2027 Campus - Shenzhen</title>
  <meta name="description" content="Campus recruitment for Go backend microservices in Shenzhen.">
</head>
<body>Apply now</body>
</html>`))
	}))
	defer server.Close()

	job, err := ImportURL(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("import url: %v", err)
	}

	if job.Title != "Tencent Go Backend Engineer 2027 Campus - Shenzhen" {
		t.Fatalf("unexpected title: %q", job.Title)
	}
	if job.Description != "Campus recruitment for Go backend microservices in Shenzhen." {
		t.Fatalf("unexpected description: %q", job.Description)
	}
	if job.ApplyURL != server.URL || job.SourceURL != server.URL {
		t.Fatalf("expected source/apply URL to round trip")
	}
	if job.Status != "" {
		t.Fatalf("successful import should let scorer choose status, got %q", job.Status)
	}
}

func TestImportURLKeepsFailedFetchAsManualCheck(t *testing.T) {
	job, err := ImportURL(context.Background(), "https://127.0.0.1:1/jobs/backend", http.DefaultClient)
	if err != nil {
		t.Fatalf("failed fetch should still create manual-check job: %v", err)
	}
	if job.Status != "manual_check" {
		t.Fatalf("expected manual_check, got %q", job.Status)
	}
	if !strings.Contains(job.Description, "Fetch failed") {
		t.Fatalf("expected fetch failure in description, got %q", job.Description)
	}
}

func TestImportURLRejectsInvalidURL(t *testing.T) {
	_, err := ImportURL(context.Background(), "not-a-url", http.DefaultClient)
	if err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestDiscoverLinksFindsRecruitmentAnchors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<a href="/jobs/backend-go">Go Backend Engineer 2027 Campus</a>
<a href="/about">About us</a>
<a href="/jobs/backend-go#apply">Duplicate apply link</a>
<a href="mailto:hr@example.com">Email HR</a>
</body></html>`))
	}))
	defer server.Close()

	links, err := DiscoverLinks(context.Background(), server.URL, server.Client(), 10)
	if err != nil {
		t.Fatalf("discover links: %v", err)
	}

	if len(links) != 1 {
		t.Fatalf("expected one recruitment link, got %d: %#v", len(links), links)
	}
	if links[0] != server.URL+"/jobs/backend-go" {
		t.Fatalf("unexpected link %q", links[0])
	}
}
