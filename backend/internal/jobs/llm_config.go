package jobs

import "strings"

type LLMConfig struct {
	Provider string `json:"provider"`
	APIKey   string `json:"-"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
}

type AgentChatStatus struct {
	Mode         string `json:"mode"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	BaseURL      string `json:"base_url"`
	Configured   bool   `json:"configured"`
	FallbackMode string `json:"fallback_mode"`
}

type AgentChatHealthcheck struct {
	Status     string `json:"status"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	BaseURL    string `json:"base_url"`
	Configured bool   `json:"configured"`
	Message    string `json:"message"`
}

func NormalizeLLMConfig(config LLMConfig) LLMConfig {
	config.Provider = normalizeLLMProvider(config.Provider)
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.Model = strings.TrimSpace(config.Model)
	if config.Provider == "deepseek" && config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com"
	}
	if config.Provider == "" && config.BaseURL != "" {
		config.Provider = "openai_compatible"
	}
	if config.Provider == "" {
		config.Provider = "openai_compatible"
	}
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
		Provider:     config.Provider,
		Model:        config.Model,
		BaseURL:      config.BaseURL,
		Configured:   configured,
		FallbackMode: "local_rules",
	}
}

func normalizeLLMProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "deepseek":
		return "deepseek"
	case "openai", "openai_compatible", "openai-compatible", "compatible":
		return "openai_compatible"
	default:
		return ""
	}
}
