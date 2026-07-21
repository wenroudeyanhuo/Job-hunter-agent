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

func ShouldSendDutyReport(settings Settings, now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	settings = normalizeSettings(settings)
	if !settings.AutoDutyReportEnabled {
		return false
	}
	hour, minute := parseClock(settings.DutyReportTime, 18, 0)
	dueAt := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if now.Before(dueAt) {
		return false
	}
	if settings.LastDutyReportSentAt == nil {
		return true
	}
	last := settings.LastDutyReportSentAt.In(now.Location())
	return last.Year() != now.Year() || last.YearDay() != now.YearDay()
}

func nextDutyReportAt(reportTime string, now time.Time) string {
	hour, minute := parseClock(reportTime, 18, 0)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Format(time.RFC3339)
}

func parseClock(value string, fallbackHour int, fallbackMinute int) (int, int) {
	hour := fallbackHour
	minute := fallbackMinute
	if _, err := fmt.Sscanf(value, "%d:%d", &hour, &minute); err != nil {
		return fallbackHour, fallbackMinute
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return fallbackHour, fallbackMinute
	}
	return hour, minute
}
