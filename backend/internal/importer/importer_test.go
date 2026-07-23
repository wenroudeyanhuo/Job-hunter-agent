package importer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
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

func TestImportURLParsesOpenGraphAndChineseCity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head>
<meta property="og:title" content="Go 后端开发工程师 - 深圳 - 校园招聘">
<meta name="keywords" content="岗位职责, 任职要求, 立即申请">
</head><body></body></html>`))
	}))
	defer server.Close()

	job, err := ImportURL(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("import url: %v", err)
	}
	if job.Title != "Go 后端开发工程师 - 深圳 - 校园招聘" {
		t.Fatalf("unexpected title: %q", job.Title)
	}
	if job.City != "Shenzhen" {
		t.Fatalf("expected Shenzhen city, got %q", job.City)
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

func TestDiscoverLinksFindsChineseRecruitmentAnchors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<a href="/positions/algorithm">算法工程师（深圳）</a>
<a href="/about">关于我们</a>
</body></html>`))
	}))
	defer server.Close()

	links, err := DiscoverLinks(context.Background(), server.URL, server.Client(), 10)
	if err != nil {
		t.Fatalf("discover links: %v", err)
	}

	if len(links) != 1 || links[0] != server.URL+"/positions/algorithm" {
		t.Fatalf("unexpected links: %#v", links)
	}
}

func TestDiscoverLinksFindsRecruitmentURLsInScripts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<script>window.__jobs = [{"url":"\/campus\/position\/12345?lang=zh-CN"}]</script>
</body></html>`))
	}))
	defer server.Close()

	links, err := DiscoverLinks(context.Background(), server.URL, server.Client(), 10)
	if err != nil {
		t.Fatalf("discover links: %v", err)
	}

	if len(links) != 1 || links[0] != server.URL+"/campus/position/12345?lang=zh-CN" {
		t.Fatalf("unexpected links: %#v", links)
	}
}

func TestDiscoverJobCardsExtractsStructuredListingRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<section class="job-list">
  <a class="job-card" href="/jobs/go-backend-2027">
    <h3>Go 后端开发工程师 2027 校招</h3>
    <span>深圳</span>
    <p>负责云服务后端开发，要求 Go、微服务、MySQL。</p>
  </a>
  <a class="job-card" href="/jobs/ai-app-intern">
    <h3>AI 应用开发实习生</h3>
    <span>深圳</span>
    <p>参与 LLM 应用、RAG 和智能体开发。</p>
  </a>
  <a href="/about">关于我们</a>
</section>
</body></html>`))
	}))
	defer server.Close()

	cards, err := DiscoverJobCards(context.Background(), server.URL, server.Client(), 10)
	if err != nil {
		t.Fatalf("discover job cards: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected two job cards, got %#v", cards)
	}
	if cards[0].Title != "Go 后端开发工程师 2027 校招" || cards[0].City != "Shenzhen" {
		t.Fatalf("unexpected first card: %#v", cards[0])
	}
	if cards[1].ApplyURL != server.URL+"/jobs/ai-app-intern" {
		t.Fatalf("unexpected second card URL: %#v", cards[1])
	}
}

func TestDiscoverJobCardsFallsBackToAnchorText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<a href="/campus/position/1001">Java 后端工程师 深圳 校招 岗位职责 任职要求 投递</a>
<a href="/news/1">招聘季新闻</a>
</body></html>`))
	}))
	defer server.Close()

	cards, err := DiscoverJobCards(context.Background(), server.URL, server.Client(), 10)
	if err != nil {
		t.Fatalf("discover job cards: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected one text-only card, got %#v", cards)
	}
	if cards[0].City != "Shenzhen" || cards[0].ApplyURL != server.URL+"/campus/position/1001" {
		t.Fatalf("unexpected card: %#v", cards[0])
	}
}

func TestLooksLikeConcreteJobPostingRejectsRecruitmentLandingPage(t *testing.T) {
	if LooksLikeConcreteJobPosting(domain.Job{
		Title:       "华为应届生_实习生_留学生_海外本地最新招聘信息-华为校园招聘",
		Description: "校园招聘官网",
		ApplyURL:    "https://career.example.com/",
	}) {
		t.Fatal("expected recruitment landing page to be rejected")
	}
}

func TestLooksLikeConcreteJobPostingRejectsKnownRecruitmentPortals(t *testing.T) {
	cases := []domain.Job{
		{
			Title:       "校园招聘 - DJI 大疆招聘",
			Description: "从这里起飞，了解校园招聘、拓疆者计划、实习生招聘等最新校招资讯。",
			ApplyURL:    "https://we.dji.com/zh-CN/campus",
			SourceURL:   "https://we.dji.com/zh-CN/campus",
		},
		{
			Title:       "百度校园招聘",
			Description: "百度官方招聘平台邀请来自社会、校园、实习生、海外的精英加入百度。",
			ApplyURL:    "https://talent.baidu.com/jobs/list",
			SourceURL:   "https://talent.baidu.com/jobs/list",
		},
		{
			Title:       "百度招聘",
			Description: "百度官方招聘平台邀请来自社会、校园、实习生、海外的精英加入百度。",
			ApplyURL:    "https://talent.baidu.com/static/index.html",
			SourceURL:   "https://talent.baidu.com/static/index.html",
		},
	}

	for _, job := range cases {
		if LooksLikeConcreteJobPosting(job) {
			t.Fatalf("expected recruitment portal to be rejected: %#v", job)
		}
	}
}

func TestLooksLikeConcreteJobPostingAcceptsRolePage(t *testing.T) {
	if !LooksLikeConcreteJobPosting(domain.Job{
		Title:       "Go 后端开发工程师 2027 校招 - 深圳",
		Description: "岗位职责和任职要求：负责 Go 后端微服务开发。立即申请。",
		ApplyURL:    "https://career.example.com/jobs/backend-go",
	}) {
		t.Fatal("expected concrete role page to be accepted")
	}
}
