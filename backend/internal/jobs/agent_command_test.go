package jobs

import "testing"

func TestPlanAgentCommandRecognizesWorkflowActions(t *testing.T) {
	plan := PlanAgentCommand("只看深圳 Go 后端，刷新任务并发送飞书日报", DefaultSettings())

	if plan.Result.Intent != "update_workflow" {
		t.Fatalf("unexpected intent: %#v", plan.Result)
	}
	if !containsTestString(plan.TargetCities, "Shenzhen") {
		t.Fatalf("expected Shenzhen target city, got %#v", plan.TargetCities)
	}
	if !containsTestString(plan.TargetDirections, "go") || !containsTestString(plan.TargetDirections, "backend") {
		t.Fatalf("expected go/backend directions, got %#v", plan.TargetDirections)
	}
	if !plan.RefreshTasks || !plan.SendFeishuReport {
		t.Fatalf("expected refresh and feishu actions, got %#v", plan)
	}
}

func TestPlanAgentCommandRecognizesNormalChineseWorkflowActions(t *testing.T) {
	plan := PlanAgentCommand("\u53ea\u770b\u6df1\u5733 Go \u540e\u7aef\uff0c\u5237\u65b0\u4efb\u52a1\u5e76\u91c7\u96c6\u6700\u65b0\u5c97\u4f4d", DefaultSettings())

	if !containsTestString(plan.TargetCities, "Shenzhen") {
		t.Fatalf("expected Shenzhen target city, got %#v", plan.TargetCities)
	}
	if !containsTestString(plan.TargetDirections, "go") || !containsTestString(plan.TargetDirections, "backend") {
		t.Fatalf("expected go/backend directions, got %#v", plan.TargetDirections)
	}
	if !plan.RefreshTasks || !plan.RunCrawl {
		t.Fatalf("expected refresh and crawl actions, got %#v", plan)
	}
}

func TestPlanAgentCommandRecognizesApplicationPlanning(t *testing.T) {
	plan := PlanAgentCommand("帮我同步投递计划，准备投递这些感兴趣岗位", DefaultSettings())

	if !plan.SyncApplicationPlans {
		t.Fatalf("expected application plan sync, got %#v", plan)
	}
	if !containsCommandAction(plan.Result.Actions, "sync_application_plans") {
		t.Fatalf("expected sync action, got %#v", plan.Result.Actions)
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

func containsCommandAction(actions []AgentCommandAction, want string) bool {
	for _, action := range actions {
		if action.Type == want {
			return true
		}
	}
	return false
}
