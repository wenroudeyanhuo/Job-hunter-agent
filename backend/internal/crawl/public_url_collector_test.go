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

func TestPublicURLCollectorRetriesTransientImportFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`<html><head><title>Tencent Go Backend Engineer 2027 Campus - Shenzhen</title><meta name="description" content="Campus recruitment for Go backend microservices in Shenzhen."></head></html>`))
	}))
	defer server.Close()

	collector := NewPublicURLCollector([]string{server.URL}, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if attempts < 2 {
		t.Fatalf("expected retry, got %d attempts", attempts)
	}
	if len(jobs) != 1 || jobs[0].Status == "manual_check" {
		t.Fatalf("expected successful retry import, got %#v", jobs)
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

	if len(jobs) != 1 {
		t.Fatalf("expected only discovered concrete job, got %d", len(jobs))
	}
	if jobs[0].Title != "Tencent Go Backend Engineer 2027 Campus - Shenzhen" {
		t.Fatalf("expected discovered backend job, got %#v", jobs)
	}
}

func TestPublicURLCollectorImportsStructuredJobCards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<div class="job-list">
  <a class="position-card" href="/positions/go-backend">
    <h3>Go 后端开发工程师 2027 校招</h3>
    <span>深圳</span>
    <p>负责 Go 后端服务、微服务和数据库开发。</p>
  </a>
  <a class="position-card" href="/positions/algorithm">
    <h3>算法工程师 2027 校招</h3>
    <span>深圳</span>
    <p>负责推荐算法、机器学习和模型优化。</p>
  </a>
</div>
</body></html>`))
	}))
	defer server.Close()

	collector := NewPublicURLCollector([]string{server.URL}, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected two job cards, got %#v", jobs)
	}
	if jobs[0].ApplyURL != server.URL+"/positions/go-backend" || jobs[0].City != "Shenzhen" {
		t.Fatalf("unexpected first job: %#v", jobs[0])
	}
}

func TestPublicURLCollectorImportsJobsFromEmbeddedJSONState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<script id="__NEXT_DATA__" type="application/json">
{
  "props": {
    "pageProps": {
      "jobs": [
        {
          "company": "DeepAI",
          "title": "AI Application Backend Engineer",
          "city": "Shenzhen",
          "url": "/jobs/ai-backend",
          "description": "Build agent workflow, RAG services, and Go backend systems.",
          "deadline": "2026-10-31"
        },
        {
          "company": "DeepAI",
          "name": "Algorithm Intern",
          "location": "Guangzhou",
          "applyUrl": "/jobs/algorithm-intern",
          "requirements": "Machine learning and recommendation algorithm internship."
        }
      ]
    }
  }
}
</script>
</body></html>`))
	}))
	defer server.Close()

	collector := NewPublicURLCollector([]string{server.URL}, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected two jobs from embedded JSON, got %#v", jobs)
	}
	if jobs[0].Company != "DeepAI" || jobs[0].City != "Shenzhen" || jobs[0].ApplyURL != server.URL+"/jobs/ai-backend" {
		t.Fatalf("unexpected embedded JSON job: %#v", jobs[0])
	}
	if jobs[0].DeadlineAt == nil {
		t.Fatalf("expected deadline to be parsed, got %#v", jobs[0])
	}
}

func TestPublicURLCollectorSkipsRecruitmentLandingPageWithoutConcreteJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>华为应届生_实习生_留学生_海外本地最新招聘信息-华为校园招聘</title></head><body>校园招聘官网</body></html>`))
	}))
	defer server.Close()

	collector := NewPublicURLCollector([]string{server.URL}, server.Client())
	jobs, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	if len(jobs) != 0 {
		t.Fatalf("expected landing page to be skipped, got %#v", jobs)
	}
}
