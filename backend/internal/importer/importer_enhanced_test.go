package importer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverJobCardsExtractsStructuredCardFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div class="job-card" data-href="/jobs/go-backend">
			<h3>Go 后端开发工程师 2027 校招</h3>
			<span>深圳</span>
			<span>截止时间：2026-09-30</span>
			<p>负责 Golang backend services and AI platform.</p>
		</div>`))
	}))
	defer server.Close()

	jobs, err := DiscoverJobCards(context.Background(), server.URL, server.Client(), 5)
	if err != nil {
		t.Fatalf("discover cards: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one structured card, got %#v", jobs)
	}
	if jobs[0].City != "Shenzhen" || jobs[0].DeadlineAt == nil {
		t.Fatalf("expected city and deadline, got %#v", jobs[0])
	}
	if !hasDirectionTag(jobs[0].DirectionTags, "go") || !hasDirectionTag(jobs[0].DirectionTags, "backend") {
		t.Fatalf("expected direction tags, got %#v", jobs[0].DirectionTags)
	}
}

func TestDiscoverPaginationLinksFindsNextPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<a rel="next" href="/jobs?page=2">下一页</a>`))
	}))
	defer server.Close()

	links, err := DiscoverPaginationLinks(context.Background(), server.URL, server.Client(), 3)
	if err != nil {
		t.Fatalf("discover pagination: %v", err)
	}
	if len(links) != 1 || links[0] != server.URL+"/jobs?page=2" {
		t.Fatalf("unexpected pagination links: %#v", links)
	}
}

func TestDiscoverJobCardsExtractsJSONLDJobPosting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head>
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "JobPosting",
  "title": "AI Application Backend Engineer 2027 Campus",
  "description": "Build LLM agent platform services with Go and RAG. Responsibilities and requirements are listed here.",
  "hiringOrganization": {"name": "AgentWorks"},
  "jobLocation": {"address": {"addressLocality": "Shenzhen"}},
  "validThrough": "2026-10-15",
  "url": "/apply/ai-backend"
}
</script></head><body></body></html>`))
	}))
	defer server.Close()

	jobs, err := DiscoverJobCards(context.Background(), server.URL, server.Client(), 5)
	if err != nil {
		t.Fatalf("discover cards: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected JSON-LD job posting, got %#v", jobs)
	}
	if jobs[0].Company != "AgentWorks" || jobs[0].City != "Shenzhen" || jobs[0].DeadlineAt == nil {
		t.Fatalf("expected company, city, and deadline from JSON-LD, got %#v", jobs[0])
	}
	if jobs[0].ApplyURL != server.URL+"/apply/ai-backend" {
		t.Fatalf("expected resolved apply URL, got %q", jobs[0].ApplyURL)
	}
	if !hasDirectionTag(jobs[0].DirectionTags, "go") || !hasDirectionTag(jobs[0].DirectionTags, "ai_application") {
		t.Fatalf("expected direction tags from JSON-LD, got %#v", jobs[0].DirectionTags)
	}
}

func hasDirectionTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}
