package jobs

import "strings"

type AgentCommandResult struct {
	Input   string               `json:"input"`
	Intent  string               `json:"intent"`
	Summary string               `json:"summary"`
	Actions []AgentCommandAction `json:"actions"`
	Needs   []string             `json:"needs"`
}

type AgentCommandAction struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	Detail string `json:"detail"`
}

type AgentCommandPlan struct {
	Result               AgentCommandResult
	TargetCities         []string
	TargetDirections     []string
	RefreshTasks         bool
	RunCrawl             bool
	SendFeishuReport     bool
	SyncApplicationPlans bool
}

func PlanAgentCommand(text string, current Settings) AgentCommandPlan {
	cleaned := strings.TrimSpace(text)
	plan := AgentCommandPlan{
		Result: AgentCommandResult{
			Input:  cleaned,
			Intent: "update_workflow",
		},
	}
	if cleaned == "" {
		plan.Result.Intent = "empty"
		plan.Result.Summary = "No command was provided."
		plan.Result.Needs = append(plan.Result.Needs, "Type what you want the agent to change or do.")
		return plan
	}

	lower := strings.ToLower(cleaned)
	plan.TargetCities = commandCities(cleaned, current.TargetCities)
	plan.TargetDirections = commandDirections(lower, current.TargetDirections)
	if !sameStringSet(plan.TargetCities, current.TargetCities) || !sameStringSet(plan.TargetDirections, current.TargetDirections) {
		plan.Result.Actions = append(plan.Result.Actions, AgentCommandAction{
			Type:   "update_settings",
			Target: "preferences",
			Detail: "Updated target cities or directions.",
		})
	}
	plan.RefreshTasks = containsAny(lower, []string{"刷新任务", "更新任务", "任务", "refresh task", "work queue"})
	if plan.RefreshTasks {
		plan.Result.Actions = append(plan.Result.Actions, AgentCommandAction{
			Type:   "refresh_tasks",
			Target: "daily_tasks",
			Detail: "Rebuilt today's task queue.",
		})
	}
	plan.SyncApplicationPlans = containsAny(lower, []string{"同步投递", "投递计划", "准备投递", "申请计划", "application plan", "applications"})
	if plan.SyncApplicationPlans {
		plan.Result.Actions = append(plan.Result.Actions, AgentCommandAction{
			Type:   "sync_application_plans",
			Target: "applications",
			Detail: "Synced human-approved application preparation plans.",
		})
	}
	plan.RunCrawl = containsAny(lower, []string{"采集", "抓取", "crawl", "run crawl"})
	if plan.RunCrawl {
		plan.Result.Actions = append(plan.Result.Actions, AgentCommandAction{
			Type:   "run_crawl",
			Target: "sources",
			Detail: "Started a manual crawl.",
		})
	}
	plan.SendFeishuReport = containsAny(lower, []string{"飞书", "feishu", "日报", "报告", "report"})
	if plan.SendFeishuReport {
		plan.Result.Actions = append(plan.Result.Actions, AgentCommandAction{
			Type:   "send_feishu_report",
			Target: "notification",
			Detail: "Sent the current duty report when Feishu is configured.",
		})
	}
	if len(plan.Result.Actions) == 0 {
		plan.Result.Intent = "needs_clarification"
		plan.Result.Summary = "I could not map this command to a supported action yet."
		plan.Result.Needs = append(plan.Result.Needs, "Try commands like: only watch Shenzhen Go backend, refresh tasks, sync application plans, run crawl, or send Feishu report.")
		return plan
	}
	plan.Result.Summary = "Command accepted. I applied the supported workflow changes."
	return plan
}

func commandCities(text string, fallback []string) []string {
	candidates := []string{"深圳", "广州", "上海", "北京", "杭州", "成都", "武汉", "Shenzhen", "Guangzhou", "Shanghai", "Beijing", "Hangzhou", "Chengdu", "Wuhan"}
	out := []string{}
	for _, city := range candidates {
		if strings.Contains(text, city) {
			out = append(out, city)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return cleanStringList(out)
}

func commandDirections(lower string, fallback []string) []string {
	candidates := []string{"frontend", "backend", "java", "go", "algorithm", "ai_application"}
	aliases := map[string][]string{
		"frontend":       {"前端", "frontend"},
		"backend":        {"后端", "backend"},
		"java":           {"java"},
		"go":             {"go", "golang"},
		"algorithm":      {"算法", "algorithm"},
		"ai_application": {"ai应用", "ai 应用", "ai_application", "llm", "大模型"},
	}
	out := []string{}
	for _, direction := range candidates {
		for _, alias := range aliases[direction] {
			if strings.Contains(lower, strings.ToLower(alias)) {
				out = append(out, direction)
				break
			}
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return cleanStringList(out)
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func sameStringSet(left []string, right []string) bool {
	left = cleanStringList(left)
	right = cleanStringList(right)
	if len(left) != len(right) {
		return false
	}
	seen := map[string]bool{}
	for _, item := range left {
		seen[strings.ToLower(item)] = true
	}
	for _, item := range right {
		if !seen[strings.ToLower(item)] {
			return false
		}
	}
	return true
}
