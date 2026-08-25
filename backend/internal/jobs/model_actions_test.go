package jobs

import (
	"strings"
	"testing"
)

func TestParseModelActionReplyAllowsWhitelistedActions(t *testing.T) {
	reply := `{"content":"我建议先同步投递计划，再刷新任务。","actions":[{"type":"sync_application_plans","target":"applications","detail":"准备投递工作台"},{"type":"refresh_tasks","target":"tasks","detail":"更新今日任务"}]}`

	parsed := ParseModelActionReply(reply)
	if parsed.Content != "我建议先同步投递计划，再刷新任务。" {
		t.Fatalf("unexpected content: %#v", parsed)
	}
	if len(parsed.Actions) != 2 {
		t.Fatalf("expected two whitelisted actions, got %#v", parsed.Actions)
	}
	if parsed.Actions[0].Type != "sync_application_plans" || parsed.Actions[1].Type != "refresh_tasks" {
		t.Fatalf("unexpected actions: %#v", parsed.Actions)
	}
}

func TestParseModelActionReplyRejectsUnsafeActions(t *testing.T) {
	reply := `{"content":"我不能直接替你投递。","actions":[{"type":"auto_apply_resume","target":"external","detail":"submit resume"},{"type":"send_feishu_report","target":"feishu","detail":"发送日报"}]}`

	parsed := ParseModelActionReply(reply)
	if len(parsed.Actions) != 1 {
		t.Fatalf("expected only one safe action, got %#v", parsed.Actions)
	}
	if parsed.Actions[0].Type != "send_feishu_report" {
		t.Fatalf("expected safe feishu action, got %#v", parsed.Actions)
	}
}

func TestParseModelActionReplyAllowsRecommendedSourceBootstrap(t *testing.T) {
	reply := `{"content":"I should bootstrap the source pool first.","actions":[{"type":"add_recommended_and_crawl","target":"sources","detail":"Add recommended sources and run the first crawl."}]}`

	parsed := ParseModelActionReply(reply)

	if len(parsed.Actions) != 1 || parsed.Actions[0].Type != "add_recommended_and_crawl" {
		t.Fatalf("expected recommended source bootstrap action, got %#v", parsed.Actions)
	}
}

func TestModelActionPromptListIncludesEveryAllowedAction(t *testing.T) {
	actions := AllowedModelActionTypes()
	prompt := ModelActionPromptList()

	if len(actions) == 0 {
		t.Fatal("expected allowed model actions")
	}
	for _, action := range actions {
		if !strings.Contains(prompt, action.Type) {
			t.Fatalf("expected prompt list %q to include action %q", prompt, action.Type)
		}
	}
	if !strings.Contains(prompt, "add_recommended_and_crawl") {
		t.Fatalf("expected recommended bootstrap action in prompt list, got %q", prompt)
	}
}

func TestParseModelToolCallReplyUsesRegisteredToolSchema(t *testing.T) {
	raw := `{"tool_calls":[{"name":"run_crawl","target":"sources","detail":"Run a crawl.","arguments":{}},{"name":"submit_resume","target":"external","detail":"unsafe","arguments":{}}]}`

	actions := ParseModelToolCallReply(raw)

	if len(actions) != 1 || actions[0].Type != "run_crawl" {
		t.Fatalf("expected only registered safe tool call, got %#v", actions)
	}
	if schema := ModelToolSchemaPrompt(); !strings.Contains(schema, "run_crawl") || strings.Contains(schema, "submit_resume") {
		t.Fatalf("unexpected tool schema prompt: %s", schema)
	}
}
