package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryRecordsAndListsAgentChatMessages(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	if _, err := repo.RecordAgentChatMessage(ctx, AgentChatMessageInput{Role: "user", Content: "帮我看看今天该投什么"}); err != nil {
		t.Fatalf("record user message: %v", err)
	}
	if _, err := repo.RecordAgentChatMessage(ctx, AgentChatMessageInput{Role: "assistant", Content: "我建议先看强匹配岗位", Source: "local"}); err != nil {
		t.Fatalf("record assistant message: %v", err)
	}

	messages, err := repo.ListAgentChatMessages(ctx, 10)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected two messages, got %d", len(messages))
	}
	if messages[0].Role != "user" || messages[1].Role != "assistant" {
		t.Fatalf("messages should be returned oldest first, got %#v", messages)
	}
}

func TestBuildLocalAgentChatReplyUsesContext(t *testing.T) {
	reply := BuildLocalAgentChatReply("今天该做什么", AgentChatContext{
		OpenTasks:       3,
		StrongMatches:   5,
		ManualDecisions: 2,
		SourceIssues:    1,
		ModelEnabled:    false,
	})
	if reply.Content == "" || reply.Source != "local" {
		t.Fatalf("expected local reply, got %#v", reply)
	}
	if len(reply.Actions) == 0 {
		t.Fatalf("expected suggested actions, got %#v", reply)
	}
}

func TestBuildLocalAgentChatReplySuggestsApplicationPlanSync(t *testing.T) {
	reply := BuildLocalAgentChatReply("帮我准备投递计划", AgentChatContext{})
	if !containsCommandAction(reply.Actions, "sync_application_plans") {
		t.Fatalf("expected sync application action, got %#v", reply.Actions)
	}
}
