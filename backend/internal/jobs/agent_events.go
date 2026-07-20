package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type AgentEvent struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	Level     string    `json:"level"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentEventInput struct {
	Type    string
	Title   string
	Summary string
	Level   string
}

func (r *Repository) CreateAgentEvent(ctx context.Context, input AgentEventInput) (AgentEvent, error) {
	input = normalizeAgentEventInput(input)
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_events (type, title, summary, level)
		VALUES (?, ?, ?, ?)
	`, input.Type, input.Title, input.Summary, input.Level)
	if err != nil {
		return AgentEvent{}, fmt.Errorf("insert agent event: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return AgentEvent{}, fmt.Errorf("read agent event id: %w", err)
	}
	return r.GetAgentEvent(ctx, id)
}

func (r *Repository) GetAgentEvent(ctx context.Context, id int64) (AgentEvent, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, type, title, summary, level, created_at
		FROM agent_events
		WHERE id = ?
	`, id)
	event, err := scanAgentEvent(row)
	if err != nil {
		return AgentEvent{}, fmt.Errorf("get agent event %d: %w", id, err)
	}
	return event, nil
}

func (r *Repository) ListAgentEvents(ctx context.Context, limit int) ([]AgentEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, title, summary, level, created_at
		FROM agent_events
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list agent events: %w", err)
	}
	defer rows.Close()

	events := []AgentEvent{}
	for rows.Next() {
		event, err := scanAgentEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent events: %w", err)
	}
	return events, nil
}

func normalizeAgentEventInput(input AgentEventInput) AgentEventInput {
	input.Type = strings.TrimSpace(input.Type)
	input.Title = strings.TrimSpace(input.Title)
	input.Summary = strings.TrimSpace(input.Summary)
	input.Level = strings.TrimSpace(input.Level)
	if input.Type == "" {
		input.Type = "agent_note"
	}
	if input.Title == "" {
		input.Title = "Agent activity"
	}
	if input.Level == "" {
		input.Level = "info"
	}
	return input
}

func scanAgentEvent(scanner jobScanner) (AgentEvent, error) {
	var event AgentEvent
	if err := scanner.Scan(
		&event.ID,
		&event.Type,
		&event.Title,
		&event.Summary,
		&event.Level,
		&event.CreatedAt,
	); err != nil {
		return AgentEvent{}, fmt.Errorf("scan agent event: %w", err)
	}
	return event, nil
}
