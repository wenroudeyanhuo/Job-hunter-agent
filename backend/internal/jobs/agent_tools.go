package jobs

import (
	"sort"
	"strings"
)

const (
	AgentToolRiskLow    = "low"
	AgentToolRiskMedium = "medium"
	AgentToolRiskHigh   = "high"
)

type AgentToolDefinition struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	InputSchema      string `json:"input_schema"`
	RiskLevel        string `json:"risk_level"`
	RequiresApproval bool   `json:"requires_approval"`
	Preview          string `json:"preview"`
}

type AgentStructuredToolCall struct {
	Name                string            `json:"name"`
	Target              string            `json:"target"`
	Arguments           map[string]string `json:"arguments"`
	Reason              string            `json:"reason"`
	ExpectedObservation string            `json:"expected_observation"`
	RiskLevel           string            `json:"risk_level"`
	RequiresApproval    bool              `json:"requires_approval"`
	RejectedReason      string            `json:"rejected_reason,omitempty"`
}

type AgentToolCallValidationResult struct {
	Valid    []AgentStructuredToolCall `json:"valid"`
	Rejected []AgentStructuredToolCall `json:"rejected"`
	Actions  []AgentCommandAction      `json:"actions"`
}

type AgentToolRegistry struct {
	tools map[string]AgentToolDefinition
}

func NewDefaultAgentToolRegistry() AgentToolRegistry {
	registry := AgentToolRegistry{tools: map[string]AgentToolDefinition{}}
	for _, tool := range []AgentToolDefinition{
		{
			Name:             "add_recommended_and_crawl",
			Description:      "Seed the recommended public recruiting source pool and run a crawl.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskMedium,
			RequiresApproval: true,
			Preview:          "Adds recommended sources, runs a crawl, refreshes tasks, and records a review snapshot.",
		},
		{
			Name:             "run_crawl",
			Description:      "Run a manual crawl against enabled recruiting sources.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskMedium,
			RequiresApproval: true,
			Preview:          "Fetches enabled sources, imports new jobs, deduplicates results, and updates source health.",
		},
		{
			Name:             "refresh_tasks",
			Description:      "Rebuild today's agent task queue from jobs, decisions, sources, and plans.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Recomputes local daily tasks without contacting external services.",
		},
		{
			Name:             "sync_application_plans",
			Description:      "Create or update application preparation plans for promising jobs.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Creates local application plans and does not submit any resume.",
		},
		{
			Name:             "send_feishu_report",
			Description:      "Send the current duty report to the configured Feishu incoming bot.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskMedium,
			RequiresApproval: true,
			Preview:          "Posts a duty report summary to your configured Feishu webhook.",
		},
		{
			Name:             "discover_sources",
			Description:      "Discover new recruiting source candidates from the current profile.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Generates and stores source candidates for later validation and acceptance.",
		},
		{
			Name:             "generate_daily_plan",
			Description:      "Generate today's recruiting work plan from the latest review.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Creates a local plan and suggested actions; it does not contact external services.",
		},
		{
			Name:             "inspect_source_health",
			Description:      "Inspect enabled source health and summarize broken or warning sources.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Reads local source health and records an audit event without crawling.",
		},
		{
			Name:             "validate_source_candidates",
			Description:      "Validate pending recruiting source candidates and update confidence.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskMedium,
			RequiresApproval: true,
			Preview:          "Fetches pending candidate pages, checks recruitment signals, and updates local validation status.",
		},
		{
			Name:             "review_parser_gaps",
			Description:      "Open the parser-gap review workflow for low-confidence or manual-check pages.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Keeps work local and helps decide which parser/source needs improvement.",
		},
		{
			Name:             "rebuild_semantic_memory",
			Description:      "Rebuild local semantic memory from tracked jobs, profile, and decisions.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Reindexes local memory only; it does not contact external services by default.",
		},
		{
			Name:             "review_strong_matches",
			Description:      "Navigate the user to strong job matches for manual review.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Marks the action complete after opening the relevant review workflow.",
		},
		{
			Name:             "review_manual_check",
			Description:      "Navigate the user to jobs that need manual verification.",
			InputSchema:      `{"type":"object","properties":{},"additionalProperties":false}`,
			RiskLevel:        AgentToolRiskLow,
			RequiresApproval: true,
			Preview:          "Marks the action complete after opening the manual-check workflow.",
		},
	} {
		registry.Register(tool)
	}
	return registry
}

func (r AgentToolRegistry) Register(tool AgentToolDefinition) {
	if r.tools == nil || tool.Name == "" {
		return
	}
	tool.RiskLevel = normalizeAgentToolRisk(tool.RiskLevel)
	r.tools[tool.Name] = tool
}

func (r AgentToolRegistry) Get(name string) (AgentToolDefinition, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r AgentToolRegistry) List() []AgentToolDefinition {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]AgentToolDefinition, 0, len(names))
	for _, name := range names {
		out = append(out, r.tools[name])
	}
	return out
}

func ValidateAgentToolCalls(calls []AgentStructuredToolCall, registry AgentToolRegistry) AgentToolCallValidationResult {
	if registry.tools == nil {
		registry = NewDefaultAgentToolRegistry()
	}
	result := AgentToolCallValidationResult{
		Valid:    []AgentStructuredToolCall{},
		Rejected: []AgentStructuredToolCall{},
		Actions:  []AgentCommandAction{},
	}
	for _, call := range calls {
		call.Name = strings.TrimSpace(call.Name)
		call.Target = strings.TrimSpace(call.Target)
		call.Reason = strings.TrimSpace(call.Reason)
		call.ExpectedObservation = strings.TrimSpace(call.ExpectedObservation)
		tool, ok := registry.Get(call.Name)
		if !ok {
			call.RejectedReason = "tool is not registered"
			result.Rejected = append(result.Rejected, call)
			continue
		}
		if call.Reason == "" {
			call.RejectedReason = "reason is required"
			result.Rejected = append(result.Rejected, call)
			continue
		}
		if call.ExpectedObservation == "" {
			call.RejectedReason = "expected_observation is required"
			result.Rejected = append(result.Rejected, call)
			continue
		}
		allowed := allowedModelActionTypes[call.Name]
		if call.Target == "" {
			call.Target = allowed.Target
		}
		call.RiskLevel = tool.RiskLevel
		call.RequiresApproval = tool.RequiresApproval
		result.Valid = append(result.Valid, call)
		detail := call.Reason
		if strings.TrimSpace(allowed.Detail) != "" {
			detail = allowed.Detail + " " + call.Reason
		}
		result.Actions = append(result.Actions, AgentCommandAction{
			Type:   call.Name,
			Target: call.Target,
			Detail: strings.TrimSpace(detail),
		})
	}
	result.Actions = safeAgentActions(result.Actions)
	return result
}

func AttachAgentToolMetadata(request AgentActionRequest, registry AgentToolRegistry) AgentActionRequest {
	tool, ok := registry.Get(request.ActionType)
	if !ok {
		return request
	}
	request.ToolName = tool.Name
	request.ToolDescription = tool.Description
	request.ToolInputSchema = tool.InputSchema
	request.RiskLevel = tool.RiskLevel
	request.RequiresApproval = tool.RequiresApproval
	request.ToolPreview = tool.Preview
	return request
}

func normalizeAgentToolRisk(risk string) string {
	switch risk {
	case AgentToolRiskLow, AgentToolRiskMedium, AgentToolRiskHigh:
		return risk
	default:
		return AgentToolRiskLow
	}
}
