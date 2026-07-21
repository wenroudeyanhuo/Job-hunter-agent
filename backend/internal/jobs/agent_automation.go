package jobs

import (
	"fmt"
	"time"
)

type AgentAutomationState struct {
	DutyReportEnabled bool             `json:"duty_report_enabled"`
	DutyReportTime    string           `json:"duty_report_time"`
	LastReportSentAt  *time.Time       `json:"last_report_sent_at,omitempty"`
	NextDutyReportAt  string           `json:"next_duty_report_at"`
	TaskSLAHours      int              `json:"task_sla_hours"`
	StaleTaskCount    int              `json:"stale_task_count"`
	StaleTasks        []AgentStaleTask `json:"stale_tasks"`
}

type AgentStaleTask struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	AgeHours int    `json:"age_hours"`
}

func BuildAgentAutomationState(settings Settings, tasks []AgentTask, now time.Time) AgentAutomationState {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	settings = normalizeSettings(settings)
	state := AgentAutomationState{
		DutyReportEnabled: settings.AutoDutyReportEnabled,
		DutyReportTime:    settings.DutyReportTime,
		LastReportSentAt:  settings.LastDutyReportSentAt,
		NextDutyReportAt:  nextDutyReportAt(settings.DutyReportTime, now),
		TaskSLAHours:      settings.TaskSLAHours,
		StaleTasks:        []AgentStaleTask{},
	}
	for _, task := range tasks {
		if task.Status == AgentTaskStatusDone || task.CreatedAt.IsZero() {
			continue
		}
		ageHours := int(now.Sub(task.CreatedAt).Hours())
		if ageHours < settings.TaskSLAHours {
			continue
		}
		state.StaleTasks = append(state.StaleTasks, AgentStaleTask{
			ID:       task.ID,
			Title:    task.Title,
			Detail:   task.Detail,
			AgeHours: ageHours,
		})
	}
	state.StaleTaskCount = len(state.StaleTasks)
	return state
}

func nextDutyReportAt(reportTime string, now time.Time) string {
	hour := 18
	minute := 0
	_, _ = fmt.Sscanf(reportTime, "%d:%d", &hour, &minute)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Format(time.RFC3339)
}
