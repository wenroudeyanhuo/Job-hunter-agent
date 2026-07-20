package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositorySettingsRoundTrip(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	settings := DefaultSettings()
	settings.TargetCities = []string{"Shenzhen", "Guangzhou"}
	settings.TargetDirections = []string{"backend", "go", "ai_application"}
	settings.ExcludedKeywords = []string{"outsourcing", "training"}
	settings.CrawlSchedule = []string{"09:00", "18:00"}
	settings.FeishuWebhookURL = " https://open.feishu.cn/open-apis/bot/v2/hook/test "

	saved, err := repo.SaveSettings(ctx, settings)
	if err != nil {
		t.Fatalf("save settings: %v", err)
	}
	loaded, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}

	if saved.UpdatedAt.IsZero() || loaded.UpdatedAt.IsZero() {
		t.Fatal("expected updated_at to be set")
	}
	assertStringSlice(t, loaded.TargetCities, settings.TargetCities)
	assertStringSlice(t, loaded.TargetDirections, settings.TargetDirections)
	assertStringSlice(t, loaded.ExcludedKeywords, settings.ExcludedKeywords)
	assertStringSlice(t, loaded.CrawlSchedule, settings.CrawlSchedule)
	if loaded.FeishuWebhookURL != "https://open.feishu.cn/open-apis/bot/v2/hook/test" {
		t.Fatalf("expected trimmed Feishu webhook URL, got %q", loaded.FeishuWebhookURL)
	}
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %#v, got %#v", want, got)
		}
	}
}
