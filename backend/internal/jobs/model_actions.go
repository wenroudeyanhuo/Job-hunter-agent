package jobs

import (
	"encoding/json"
	"sort"
	"strings"
)

var allowedModelActionTypes = map[string]AgentCommandAction{
	"add_recommended_and_crawl": {Type: "add_recommended_and_crawl", Target: "sources", Detail: "Add recommended sources and run the first crawl."},
	"run_crawl":                 {Type: "run_crawl", Target: "sources", Detail: "Run a manual crawl."},
	"refresh_tasks":             {Type: "refresh_tasks", Target: "daily_tasks", Detail: "Refresh today's task queue."},
	"sync_application_plans":    {Type: "sync_application_plans", Target: "applications", Detail: "Sync application preparation plans."},
	"send_feishu_report":        {Type: "send_feishu_report", Target: "notification", Detail: "Send the current duty report to Feishu."},
	"discover_sources":          {Type: "discover_sources", Target: "sources", Detail: "Discover new source candidates."},
	"review_strong_matches":     {Type: "review_strong_matches", Target: "opportunities", Detail: "Review strong matched jobs."},
	"review_manual_check":       {Type: "review_manual_check", Target: "opportunities", Detail: "Review jobs that need manual decisions."},
}

func AllowedModelActionTypes() []AgentCommandAction {
	keys := make([]string, 0, len(allowedModelActionTypes))
	for key := range allowedModelActionTypes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	actions := make([]AgentCommandAction, 0, len(keys))
	for _, key := range keys {
		actions = append(actions, allowedModelActionTypes[key])
	}
	return actions
}

func ModelActionPromptList() string {
	actions := AllowedModelActionTypes()
	types := make([]string, 0, len(actions))
	for _, action := range actions {
		types = append(types, action.Type)
	}
	return strings.Join(types, ", ")
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
