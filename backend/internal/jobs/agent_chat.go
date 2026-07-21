package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	AgentChatRoleUser      = "user"
	AgentChatRoleAssistant = "assistant"
)

type AgentChatMessage struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentChatMessageInput struct {
	Role    string
	Content string
	Source  string
}

type AgentChatContext struct {
	OpenTasks       int
	StrongMatches   int
	ManualDecisions int
	SourceIssues    int
	ActiveView      string
	ModelEnabled    bool
}

type AgentChatReply struct {
	Content string               `json:"content"`
	Source  string               `json:"source"`
	Actions []AgentCommandAction `json:"actions"`
}

func (r *Repository) RecordAgentChatMessage(ctx context.Context, input AgentChatMessageInput) (AgentChatMessage, error) {
	input.Role = normalizeAgentChatRole(input.Role)
	input.Content = strings.TrimSpace(input.Content)
	input.Source = strings.TrimSpace(input.Source)
	if input.Source == "" {
		input.Source = "local"
	}
	if input.Content == "" {
		return AgentChatMessage{}, fmt.Errorf("chat content is required")
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_chat_messages (role, content, source)
		VALUES (?, ?, ?)
	`, input.Role, input.Content, input.Source)
	if err != nil {
		return AgentChatMessage{}, fmt.Errorf("record agent chat message: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return AgentChatMessage{}, fmt.Errorf("read agent chat message id: %w", err)
	}
	return r.GetAgentChatMessage(ctx, id)
}

func (r *Repository) GetAgentChatMessage(ctx context.Context, id int64) (AgentChatMessage, error) {
	row := r.db.QueryRowContext(ctx, selectAgentChatMessageSQL()+` WHERE id = ?`, id)
	message, err := scanAgentChatMessage(row)
	if err != nil {
		return AgentChatMessage{}, fmt.Errorf("get agent chat message %d: %w", id, err)
	}
	return message, nil
}

func (r *Repository) ListAgentChatMessages(ctx context.Context, limit int) ([]AgentChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, role, content, source, created_at
		FROM (
			SELECT id, role, content, source, created_at
			FROM agent_chat_messages
			ORDER BY created_at DESC, id DESC
			LIMIT ?
		)
		ORDER BY created_at ASC, id ASC
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list agent chat messages: %w", err)
	}
	defer rows.Close()
	out := []AgentChatMessage{}
	for rows.Next() {
		message, err := scanAgentChatMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent chat messages: %w", err)
	}
	return out, nil
}

func BuildLocalAgentChatReply(input string, context AgentChatContext) AgentChatReply {
	text := strings.ToLower(strings.TrimSpace(input))
	reply := AgentChatReply{Source: "local"}
	if strings.Contains(text, "模型") || strings.Contains(text, "model") {
		if context.ModelEnabled {
			reply.Content = "我已经检测到模型配置，可以用模型来理解更自由的对话；同时我仍会保留本地规则作为兜底。"
		} else {
			reply.Content = "现在我处于本地规则模式，还没有检测到模型密钥。配置 LLM_API_KEY 和 LLM_MODEL 后，我就能切换到模型对话。"
		}
		return reply
	}
	if strings.Contains(text, "今天") || strings.Contains(text, "做什么") || strings.Contains(text, "推荐") || strings.Contains(text, "投") {
		reply.Content = fmt.Sprintf("我建议先处理今天的闭环：%d 个开放任务、%d 个强匹配岗位、%d 个需要你决策的岗位、%d 个来源异常。优先级是先看强匹配，再处理人工决策，最后修来源。", context.OpenTasks, context.StrongMatches, context.ManualDecisions, context.SourceIssues)
		reply.Actions = append(reply.Actions, AgentCommandAction{Type: "review_strong_matches", Target: "opportunities", Detail: "Open strong opportunities first."})
		if context.ManualDecisions > 0 {
			reply.Actions = append(reply.Actions, AgentCommandAction{Type: "review_manual_check", Target: "opportunities", Detail: "Resolve manual decisions."})
		}
		return reply
	}
	if strings.Contains(text, "采集") || strings.Contains(text, "crawl") {
		reply.Content = "可以，我可以帮你发起采集。为了避免误触，我会把它作为建议动作展示，你确认后执行。"
		reply.Actions = append(reply.Actions, AgentCommandAction{Type: "run_crawl", Target: "sources", Detail: "Run a manual crawl."})
		return reply
	}
	reply.Content = "我在。你可以问我今天该投哪些岗位、为什么某个岗位适合你、哪些任务快过期，或者让我刷新任务/运行采集。"
	return reply
}

func normalizeAgentChatRole(role string) string {
	role = strings.TrimSpace(strings.ToLower(role))
	if role == AgentChatRoleAssistant {
		return AgentChatRoleAssistant
	}
	return AgentChatRoleUser
}

func selectAgentChatMessageSQL() string {
	return `SELECT id, role, content, source, created_at FROM agent_chat_messages`
}

func scanAgentChatMessage(scanner interface {
	Scan(dest ...any) error
}) (AgentChatMessage, error) {
	var message AgentChatMessage
	if err := scanner.Scan(&message.ID, &message.Role, &message.Content, &message.Source, &message.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return AgentChatMessage{}, err
		}
		return AgentChatMessage{}, fmt.Errorf("scan agent chat message: %w", err)
	}
	return message, nil
}
