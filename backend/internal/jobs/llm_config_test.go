package jobs

import "testing"

func TestBuildAgentChatStatusReportsDeepSeekProvider(t *testing.T) {
	status := BuildAgentChatStatus(LLMConfig{
		Provider: "deepseek",
		APIKey:   "test-key",
		Model:    "deepseek-chat",
	})

	if status.Mode != "model" || !status.Configured {
		t.Fatalf("expected configured model mode, got %#v", status)
	}
	if status.Provider != "deepseek" {
		t.Fatalf("expected deepseek provider, got %#v", status)
	}
	if status.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("expected default deepseek base URL, got %#v", status)
	}
}

func TestNormalizeLLMConfigInfersOpenAICompatibleProvider(t *testing.T) {
	config := NormalizeLLMConfig(LLMConfig{
		APIKey:  "test-key",
		BaseURL: "https://example.com/v1/",
		Model:   "custom-model",
	})

	if config.Provider != "openai_compatible" {
		t.Fatalf("expected openai-compatible provider, got %#v", config)
	}
	if config.BaseURL != "https://example.com/v1" {
		t.Fatalf("expected trimmed base URL, got %#v", config)
	}
}
