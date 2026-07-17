package notify

import (
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestBuildFeishuSummary(t *testing.T) {
	text := BuildFeishuSummary(crawl.RunSummary{
		JobsCreated:      3,
		ManualCheckCount: 1,
		SourcesFailed:    2,
	}, []domain.Job{{
		Company:          "Tencent",
		Title:            "Backend Engineer",
		City:             "Shenzhen",
		MatchScore:       92,
		RecommendReasons: []string{"Shenzhen role", "Clear application URL"},
		ApplyURL:         "https://example.com/apply",
	}})

	wants := []string{
		"Jobs created: 3",
		"Strong matches: 1",
		"Manual check: 1",
		"Failed sources: 2",
		"Tencent - Backend Engineer - Shenzhen - 92",
		"Shenzhen role",
		"https://example.com/apply",
	}
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("expected summary to contain %q, got:\n%s", want, text)
		}
	}
}
