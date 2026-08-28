package jobs

import "testing"

func TestValidateAgentToolCallsKeepsOnlyRegistryBackedCalls(t *testing.T) {
	calls := []AgentStructuredToolCall{
		{
			Name:                "run_crawl",
			Arguments:           map[string]string{"scope": "enabled_sources"},
			Reason:              "Need fresh evidence before planning applications.",
			ExpectedObservation: "New run summary with created jobs and source health.",
		},
		{
			Name:                "submit_resume",
			Arguments:           map[string]string{"job_id": "42"},
			Reason:              "This should never be allowed by the personal agent.",
			ExpectedObservation: "Resume submitted.",
		},
		{
			Name:   "send_feishu_report",
			Reason: "Missing observation should make this call invalid.",
		},
	}

	result := ValidateAgentToolCalls(calls, NewDefaultAgentToolRegistry())

	if len(result.Valid) != 1 {
		t.Fatalf("expected one valid call, got %#v", result)
	}
	if result.Valid[0].Name != "run_crawl" || !result.Valid[0].RequiresApproval || result.Valid[0].RiskLevel != AgentToolRiskMedium {
		t.Fatalf("expected registry metadata on valid call, got %#v", result.Valid[0])
	}
	if len(result.Rejected) != 2 {
		t.Fatalf("expected two rejected calls, got %#v", result.Rejected)
	}
	if result.Actions[0].Type != "run_crawl" || result.Actions[0].Target != "sources" {
		t.Fatalf("expected normalized action from valid call, got %#v", result.Actions)
	}
}

func TestParseModelToolCallReplyReturnsStructuredValidation(t *testing.T) {
	raw := `{"tool_calls":[{"name":"refresh_tasks","arguments":{"date":"today"},"reason":"No daily queue exists.","expected_observation":"Fresh daily tasks are visible."},{"name":"unknown_tool","reason":"bad","expected_observation":"bad"}]}`

	result := ParseModelStructuredToolCallReply(raw)

	if len(result.Valid) != 1 || result.Valid[0].Name != "refresh_tasks" {
		t.Fatalf("expected one validated refresh_tasks call, got %#v", result)
	}
	if len(result.Rejected) != 1 {
		t.Fatalf("expected rejected unknown tool, got %#v", result)
	}
}

func TestApplyModelAgentInsightsRebuildsAutonomyPlanFromStructuredToolCalls(t *testing.T) {
	cycle := RunRecruitingAgentCycle(MultiAgentCycleInput{})
	insights := []ModelAgentInsight{{
		AgentKey: MultiAgentPlanner,
		Decision: "Need a fresh crawl.",
		ToolCalls: []AgentStructuredToolCall{{
			Name:                "run_crawl",
			Reason:              "Planner needs fresh opportunities.",
			ExpectedObservation: "A run summary with job and source health counts.",
		}},
	}}

	enhanced := ApplyModelAgentInsights(cycle, insights)

	if !containsAgentAction(enhanced.Actions, "run_crawl") {
		t.Fatalf("expected model tool call to become action, got %#v", enhanced.Actions)
	}
	if !containsAutonomyStep(enhanced.AutonomyPlan.Steps, "run_crawl") {
		t.Fatalf("expected autonomy plan to be rebuilt from model tool call, got %#v", enhanced.AutonomyPlan)
	}
}

func containsAutonomyStep(steps []AgentAutonomyStep, tool string) bool {
	for _, step := range steps {
		if step.Tool == tool {
			return true
		}
	}
	return false
}
