package jobs

import "strings"

type LLMConfig struct {
	APIKey  string `json:"-"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

type AgentChatStatus struct {
	Mode         string `json:"mode"`
	Model        string `json:"model"`
	Configured   bool   `json:"configured"`
	FallbackMode string `json:"fallback_mode"`
}

func NormalizeLLMConfig(config LLMConfig) LLMConfig {
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.Model = strings.TrimSpace(config.Model)
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	return config
}

func BuildAgentChatStatus(config LLMConfig) AgentChatStatus {
	config = NormalizeLLMConfig(config)
	configured := config.APIKey != "" && config.Model != ""
	mode := "local_rules"
	if configured {
		mode = "model"
	}
	return AgentChatStatus{
		Mode:         mode,
		Model:        config.Model,
		Configured:   configured,
		FallbackMode: "local_rules",
	}
}
