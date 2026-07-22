package jobs

import (
	"testing"
	"time"
)

func TestBuildAgentAutomationStateFindsStaleTasks(t *testing.T) {
	now := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.AutoDutyReportEnabled = true
	settings.DutyReportTime = "18:30"
	settings.TaskSLAHours = 4

	state := BuildAgentAutomationState(settings, []AgentTask{
		{
			ID:        1,
			Title:     "Review Tencent",
			Detail:    "Go Backend Engineer / Shenzhen",
			Status:    AgentTaskStatusOpen,
			CreatedAt: now.Add(-5 * time.Hour),
		},
		{
			ID:        2,
			Title:     "Done task",
			Status:    AgentTaskStatusDone,
			CreatedAt: now.Add(-8 * time.Hour),
		},
	}, now)

	if !state.DutyReportEnabled || state.DutyReportTime != "18:30" {
		t.Fatalf("expected duty report settings, got %#v", state)
	}
	if state.StaleTaskCount != 1 || len(state.StaleTasks) != 1 {
		t.Fatalf("expected one stale open task, got %#v", state)
	}
	if state.NextDutyReportAt == "" {
		t.Fatalf("expected next report time")
	}
}

func TestShouldSendDutyReportOncePerDayAfterConfiguredTime(t *testing.T) {
	now := time.Date(2026, 7, 21, 18, 5, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.AutoDutyReportEnabled = true
	settings.DutyReportTime = "18:00"
	settings.TimeZone = "UTC"

	if !ShouldSendDutyReport(settings, now) {
		t.Fatal("expected duty report to be due")
	}

	sent := time.Date(2026, 7, 21, 18, 1, 0, 0, time.UTC)
	settings.LastDutyReportSentAt = &sent
	if ShouldSendDutyReport(settings, now) {
		t.Fatal("expected duty report not to send twice on the same day")
	}
}

func TestShouldSendDutyReportWaitsUntilConfiguredTime(t *testing.T) {
	now := time.Date(2026, 7, 21, 17, 59, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.AutoDutyReportEnabled = true
	settings.DutyReportTime = "18:00"
	settings.TimeZone = "UTC"

	if ShouldSendDutyReport(settings, now) {
		t.Fatal("expected duty report to wait until configured time")
	}
}

func TestShouldSendDutyReportUsesConfiguredTimezone(t *testing.T) {
	now := time.Date(2026, 7, 22, 10, 5, 0, 0, time.UTC) // 18:05 in Asia/Shanghai.
	settings := DefaultSettings()
	settings.AutoDutyReportEnabled = true
	settings.DutyReportTime = "18:00"
	settings.TimeZone = "Asia/Shanghai"

	if !ShouldSendDutyReport(settings, now) {
		t.Fatal("expected duty report to be due at 18:00 Asia/Shanghai")
	}

	sent := time.Date(2026, 7, 22, 9, 59, 0, 0, time.UTC) // Same Shanghai day.
	settings.LastDutyReportSentAt = &sent
	if ShouldSendDutyReport(settings, now) {
		t.Fatal("expected duty report not to send twice on the same configured timezone day")
	}
}

func TestShouldRunSourceDiscoveryAfterConfiguredInterval(t *testing.T) {
	now := time.Date(2026, 7, 21, 9, 5, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.AutoSourceDiscoveryEnabled = true
	settings.SourceDiscoveryIntervalHours = 6

	if !ShouldRunSourceDiscovery(settings, now) {
		t.Fatal("expected source discovery to run when it has never run")
	}

	last := now.Add(-5 * time.Hour)
	settings.LastSourceDiscoveryAt = &last
	if ShouldRunSourceDiscovery(settings, now) {
		t.Fatal("expected source discovery to wait for interval")
	}

	last = now.Add(-6 * time.Hour)
	settings.LastSourceDiscoveryAt = &last
	if !ShouldRunSourceDiscovery(settings, now) {
		t.Fatal("expected source discovery to run after interval")
	}
}
