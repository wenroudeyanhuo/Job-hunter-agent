package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestDBSourceCollectorReadsEnabledSources(t *testing.T) {
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Tencent Go Backend Engineer 2027 Campus - Shenzhen</title></head></html>`))
	}))
	defer sourceServer.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	if _, err := repo.CreateSource(context.Background(), jobs.SourceInput{
		Name:    "enabled",
		URL:     sourceServer.URL,
		Enabled: true,
	}); err != nil {
		t.Fatalf("create enabled source: %v", err)
	}
	if _, err := repo.CreateSource(context.Background(), jobs.SourceInput{
		Name:    "disabled",
		URL:     "https://example.com/disabled",
		Enabled: false,
	}); err != nil {
		t.Fatalf("create disabled source: %v", err)
	}

	collector := NewDBSourceCollector(repo, sourceServer.Client())
	collected, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(collected) != 1 {
		t.Fatalf("expected one collected job, got %d", len(collected))
	}
	if collected[0].Title != "Tencent Go Backend Engineer 2027 Campus - Shenzhen" {
		t.Fatalf("unexpected title %q", collected[0].Title)
	}
}

func TestDBSourceCollectorUsesOfficialParserType(t *testing.T) {
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": {
				"count": 1,
				"job_post_list": [{
					"id": "7525009396952582407",
					"title": "后端开发工程师-飞书-2027届校招",
					"city_info": {"name": "深圳"},
					"description": "负责飞书业务后端服务开发。",
					"requirement": "熟悉 Go 或 Java。"
				}]
			}
		}`))
	}))
	defer sourceServer.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	if _, err := repo.CreateSource(context.Background(), jobs.SourceInput{
		Name:       "ByteDance Jobs",
		URL:        sourceServer.URL + "/campus/",
		Enabled:    true,
		ParserType: "bytedance_api",
	}); err != nil {
		t.Fatalf("create source: %v", err)
	}

	collector := NewDBSourceCollector(repo, sourceServer.Client())
	collected, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(collected) != 1 || collected[0].Company != "ByteDance" {
		t.Fatalf("expected ByteDance job, got %#v", collected)
	}
}

func TestDBSourceCollectorKeepsGoingWhenOfficialParserFails(t *testing.T) {
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary failure", http.StatusBadGateway)
	}))
	defer sourceServer.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	if _, err := repo.CreateSource(context.Background(), jobs.SourceInput{
		Name:       "ByteDance Jobs",
		URL:        sourceServer.URL,
		Enabled:    true,
		ParserType: "bytedance_api",
	}); err != nil {
		t.Fatalf("create source: %v", err)
	}

	collector := NewDBSourceCollector(repo, sourceServer.Client())
	collected, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect should not fail the whole run: %v", err)
	}
	if len(collected) != 1 || collected[0].Status != domain.StatusManualCheck {
		t.Fatalf("expected manual-check diagnostic job, got %#v", collected)
	}
	if !strings.Contains(collected[0].Description, "Source parser failed") {
		t.Fatalf("expected parser failure description, got %q", collected[0].Description)
	}
}
