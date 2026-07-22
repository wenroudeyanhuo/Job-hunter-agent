package jobs

import (
	"fmt"
	"strings"
	"time"
)

type AgentAutomationState struct {
	DutyReportEnabled        bool             `json:"duty_report_enabled"`
	SourceDiscoveryEnabled   bool             `json:"source_discovery_enabled"`
	DutyReportTime           string           `json:"duty_report_time"`
	LastReportSentAt         *time.Time       `json:"last_report_sent_at,omitempty"`
	NextDutyReportAt         string           `json:"next_duty_report_at"`
	SourceDiscoveryInterval  int              `json:"source_discovery_interval_hours"`
	LastSourceDiscoveryAt    *time.Time       `json:"last_source_discovery_at,omitempty"`
	NextSourceDiscoveryDueAt string           `json:"next_source_discovery_due_at"`
	TaskSLAHours             int              `json:"task_sla_hours"`
	StaleTaskCount           int              `json:"stale_task_count"`
	StaleTasks               []AgentStaleTask `json:"stale_tasks"`
}

type AgentAutomationDiagnostics struct {
	GeneratedAt             time.Time  `json:"generated_at"`
	SchedulerExpected       bool       `json:"scheduler_expected"`
	WebhookConfigured       bool       `json:"webhook_configured"`
	DutyReportEnabled       bool       `json:"duty_report_enabled"`
	DutyReportTime          string     `json:"duty_report_time"`
	TimeZone                string     `json:"time_zone"`
	NextDutyReportAt        string     `json:"next_duty_report_at"`
	LastDutyReportSentAt    *time.Time `json:"last_duty_report_sent_at,omitempty"`
	SourceDiscoveryEnabled  bool       `json:"source_discovery_enabled"`
	NextSourceDiscoveryAt   string     `json:"next_source_discovery_at"`
	LastSourceDiscoveryAt   *time.Time `json:"last_source_discovery_at,omitempty"`
	ReadyForAutomaticReport bool       `json:"ready_for_automatic_report"`
	Reason                  string     `json:"reason"`
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
		DutyReportEnabled:        settings.AutoDutyReportEnabled,
		SourceDiscoveryEnabled:   settings.AutoSourceDiscoveryEnabled,
		DutyReportTime:           settings.DutyReportTime,
		LastReportSentAt:         settings.LastDutyReportSentAt,
		NextDutyReportAt:         nextDutyReportAt(settings, now),
		SourceDiscoveryInterval:  settings.SourceDiscoveryIntervalHours,
		LastSourceDiscoveryAt:    settings.LastSourceDiscoveryAt,
		NextSourceDiscoveryDueAt: nextSourceDiscoveryDueAt(settings, now),
		TaskSLAHours:             settings.TaskSLAHours,
		StaleTasks:               []AgentStaleTask{},
	}
	for _, task := range tasks {
		if task.Status == AgentTaskStatusDone || task.CreatedAt.IsZero() {
			continue
		}
		if task.Status == AgentTaskStatusSnoozed && task.SnoozedUntil != nil && task.SnoozedUntil.After(now) {
			continue
		}
		ageHours := int(now.Sub(task.CreatedAt).Hours())
		if task.Status != AgentTaskStatusStale && task.Status != AgentTaskStatusEscalated && ageHours < settings.TaskSLAHours {
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

func BuildAgentAutomationDiagnostics(settings Settings, webhookConfigured bool, schedulerExpected bool, now time.Time) AgentAutomationDiagnostics {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	settings = normalizeSettings(settings)
	ready := schedulerExpected && webhookConfigured && settings.AutoDutyReportEnabled
	reason := "Automatic duty report is ready."
	if !schedulerExpected {
		reason = "Backend scheduler is disabled or not expected to run."
	} else if !webhookConfigured {
		reason = "Feishu webhook is not configured."
	} else if !settings.AutoDutyReportEnabled {
		reason = "Automatic duty report is disabled in Settings."
	}
	return AgentAutomationDiagnostics{
		GeneratedAt:             now.UTC(),
		SchedulerExpected:       schedulerExpected,
		WebhookConfigured:       webhookConfigured,
		DutyReportEnabled:       settings.AutoDutyReportEnabled,
		DutyReportTime:          settings.DutyReportTime,
		TimeZone:                settings.TimeZone,
		NextDutyReportAt:        nextDutyReportAt(settings, now),
		LastDutyReportSentAt:    settings.LastDutyReportSentAt,
		SourceDiscoveryEnabled:  settings.AutoSourceDiscoveryEnabled,
		NextSourceDiscoveryAt:   nextSourceDiscoveryDueAt(settings, now),
		LastSourceDiscoveryAt:   settings.LastSourceDiscoveryAt,
		ReadyForAutomaticReport: ready,
		Reason:                  reason,
	}
}

func ShouldSendDutyReport(settings Settings, now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	settings = normalizeSettings(settings)
	if !settings.AutoDutyReportEnabled {
		return false
	}
	now = now.In(settingsLocation(settings))
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

func ShouldRunSourceDiscovery(settings Settings, now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	settings = normalizeSettings(settings)
	if !settings.AutoSourceDiscoveryEnabled {
		return false
	}
	if settings.LastSourceDiscoveryAt == nil {
		return true
	}
	return !settings.LastSourceDiscoveryAt.Add(time.Duration(settings.SourceDiscoveryIntervalHours) * time.Hour).After(now)
}

func nextDutyReportAt(settings Settings, now time.Time) string {
	settings = normalizeSettings(settings)
	now = now.In(settingsLocation(settings))
	hour, minute := parseClock(settings.DutyReportTime, 18, 0)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Format(time.RFC3339)
}

func settingsLocation(settings Settings) *time.Location {
	name := strings.TrimSpace(settings.TimeZone)
	if name == "" {
		name = DefaultSettings().TimeZone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

func nextSourceDiscoveryDueAt(settings Settings, now time.Time) string {
	settings = normalizeSettings(settings)
	if settings.LastSourceDiscoveryAt == nil {
		return now.Format(time.RFC3339)
	}
	return settings.LastSourceDiscoveryAt.Add(time.Duration(settings.SourceDiscoveryIntervalHours) * time.Hour).Format(time.RFC3339)
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
