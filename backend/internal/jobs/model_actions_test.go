package jobs

import "testing"

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
