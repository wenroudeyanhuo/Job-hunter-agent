package config

import "testing"

func TestParseSourceURLs(t *testing.T) {
	got := parseSourceURLs("https://a.example/jobs, https://b.example\nhttps://a.example/jobs")
	want := []string{"https://a.example/jobs", "https://b.example"}
	if len(got) != len(want) {
		t.Fatalf("expected %d URLs, got %#v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("at %d expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestLoadReadsDeepSeekModelConfig(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "deepseek-key")
	t.Setenv("DEEPSEEK_MODEL", "deepseek-chat")

	cfg := Load()

	if cfg.LLMProvider != "deepseek" {
		t.Fatalf("expected deepseek provider, got %#v", cfg)
	}
	if cfg.LLMAPIKey != "deepseek-key" || cfg.LLMModel != "deepseek-chat" {
		t.Fatalf("expected deepseek key and model, got %#v", cfg)
	}
	if cfg.LLMBaseURL != "https://api.deepseek.com" {
		t.Fatalf("expected deepseek base URL, got %#v", cfg)
	}
}
