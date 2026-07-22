package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	AgentActionRequestStatusPending   = "pending"
	AgentActionRequestStatusApproved  = "approved"
	AgentActionRequestStatusDismissed = "dismissed"
)

type AgentActionRequest struct {
	ID         int64      `json:"id"`
	Source     string     `json:"source"`
	ActionType string     `json:"action_type"`
	Target     string     `json:"target"`
	Detail     string     `json:"detail"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

func (r *Repository) RecordAgentActionRequests(ctx context.Context, source string, actions []AgentCommandAction) error {
	source = strings.TrimSpace(source)
	if source == "" {
		source = "agent"
	}
	for _, action := range actions {
		allowed, ok := allowedModelActionTypes[strings.TrimSpace(action.Type)]
		if !ok {
			continue
		}
		if strings.TrimSpace(action.Target) != "" {
			allowed.Target = strings.TrimSpace(action.Target)
		}
		if strings.TrimSpace(action.Detail) != "" {
			allowed.Detail = strings.TrimSpace(action.Detail)
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO agent_action_requests (source, action_type, target, detail, status)
			VALUES (?, ?, ?, ?, ?)
		`, source, allowed.Type, allowed.Target, allowed.Detail, AgentActionRequestStatusPending); err != nil {
			return fmt.Errorf("insert agent action request: %w", err)
		}
	}
	return nil
}

func (r *Repository) ListAgentActionRequests(ctx context.Context, status string) ([]AgentActionRequest, error) {
	status = normalizeAgentActionRequestStatus(status)
	query := selectAgentActionRequestSQL()
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY status ASC, created_at DESC, id DESC LIMIT 50`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agent action requests: %w", err)
	}
	defer rows.Close()

	requests := []AgentActionRequest{}
	for rows.Next() {
		request, err := scanAgentActionRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent action requests: %w", err)
	}
	return requests, nil
}

func (r *Repository) UpdateAgentActionRequestStatus(ctx context.Context, id int64, status string) (AgentActionRequest, error) {
	status = normalizeAgentActionRequestStatus(status)
	if status == "" {
		status = AgentActionRequestStatusPending
	}
	var resolvedAt any
	if status != AgentActionRequestStatusPending {
		resolvedAt = time.Now().UTC()
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE agent_action_requests
		SET status = ?, resolved_at = ?
		WHERE id = ?
	`, status, resolvedAt, id)
	if err != nil {
		return AgentActionRequest{}, fmt.Errorf("update agent action request: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AgentActionRequest{}, fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return AgentActionRequest{}, sql.ErrNoRows
	}
	row := r.db.QueryRowContext(ctx, selectAgentActionRequestSQL()+` WHERE id = ?`, id)
	return scanAgentActionRequest(row)
}

func normalizeAgentActionRequestStatus(status string) string {
	switch strings.TrimSpace(status) {
	case AgentActionRequestStatusPending:
		return AgentActionRequestStatusPending
	case AgentActionRequestStatusApproved:
		return AgentActionRequestStatusApproved
	case AgentActionRequestStatusDismissed:
		return AgentActionRequestStatusDismissed
	default:
		return ""
	}
}

func selectAgentActionRequestSQL() string {
	return `SELECT id, source, action_type, target, detail, status, created_at, resolved_at FROM agent_action_requests`
}

func scanAgentActionRequest(scanner jobScanner) (AgentActionRequest, error) {
	var request AgentActionRequest
	if err := scanner.Scan(
		&request.ID,
		&request.Source,
		&request.ActionType,
		&request.Target,
		&request.Detail,
		&request.Status,
		&request.CreatedAt,
		&request.ResolvedAt,
	); err != nil {
		return AgentActionRequest{}, fmt.Errorf("scan agent action request: %w", err)
	}
	request.Status = normalizeAgentActionRequestStatus(request.Status)
	if request.Status == "" {
		request.Status = AgentActionRequestStatusPending
	}
	return request, nil
}
