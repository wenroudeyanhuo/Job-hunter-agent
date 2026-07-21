package jobs

import "testing"

func TestPlanAgentCommandRecognizesWorkflowActions(t *testing.T) {
	plan := PlanAgentCommand("只看深圳 Go 后端，刷新任务并发送飞书日报", DefaultSettings())

	if plan.Result.Intent != "update_workflow" {
		t.Fatalf("unexpected intent: %#v", plan.Result)
	}
	if !containsTestString(plan.TargetCities, "深圳") {
		t.Fatalf("expected Shenzhen target city, got %#v", plan.TargetCities)
	}
	if !containsTestString(plan.TargetDirections, "go") || !containsTestString(plan.TargetDirections, "backend") {
		t.Fatalf("expected go/backend directions, got %#v", plan.TargetDirections)
	}
	if !plan.RefreshTasks || !plan.SendFeishuReport {
		t.Fatalf("expected refresh and feishu actions, got %#v", plan)
	}
}

func TestPlanAgentCommandAsksForClarification(t *testing.T) {
	plan := PlanAgentCommand("帮我看看", DefaultSettings())

	if plan.Result.Intent != "needs_clarification" {
		t.Fatalf("expected clarification intent, got %#v", plan.Result)
	}
	if len(plan.Result.Needs) == 0 {
		t.Fatalf("expected guidance for unsupported command")
	}
}

func containsTestString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
