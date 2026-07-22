package jobs

import (
	"encoding/json"
	"strings"
)

var allowedModelActionTypes = map[string]AgentCommandAction{
	"run_crawl":              {Type: "run_crawl", Target: "sources", Detail: "Run a manual crawl."},
	"refresh_tasks":          {Type: "refresh_tasks", Target: "daily_tasks", Detail: "Refresh today's task queue."},
	"sync_application_plans": {Type: "sync_application_plans", Target: "applications", Detail: "Sync application preparation plans."},
	"send_feishu_report":     {Type: "send_feishu_report", Target: "notification", Detail: "Send the current duty report to Feishu."},
	"discover_sources":       {Type: "discover_sources", Target: "sources", Detail: "Discover new source candidates."},
	"review_strong_matches":  {Type: "review_strong_matches", Target: "opportunities", Detail: "Review strong matched jobs."},
	"review_manual_check":    {Type: "review_manual_check", Target: "opportunities", Detail: "Review jobs that need manual decisions."},
}

func ParseModelActionReply(raw string) AgentChatReply {
	raw = strings.TrimSpace(raw)
	reply := AgentChatReply{Content: raw, Source: "model"}
	if raw == "" {
		return reply
	}
	var payload struct {
		Content string               `json:"content"`
		Actions []AgentCommandAction `json:"actions"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return reply
	}
	reply.Content = strings.TrimSpace(payload.Content)
	if reply.Content == "" {
		reply.Content = raw
	}
	for _, action := range payload.Actions {
		actionType := strings.TrimSpace(action.Type)
		allowed, ok := allowedModelActionTypes[actionType]
		if !ok {
			continue
		}
		if strings.TrimSpace(action.Target) != "" {
			allowed.Target = strings.TrimSpace(action.Target)
		}
		if strings.TrimSpace(action.Detail) != "" {
			allowed.Detail = strings.TrimSpace(action.Detail)
		}
		reply.Actions = append(reply.Actions, allowed)
	}
	return reply
}
