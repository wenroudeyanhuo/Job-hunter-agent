package app

import (
	"context"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/notify"
)

type automationRunner struct {
	repo               *jobs.Repository
	fallbackWebhookURL string
}

func newAutomationRunner(repo *jobs.Repository, fallbackWebhookURL string) *automationRunner {
	return &automationRunner{repo: repo, fallbackWebhookURL: strings.TrimSpace(fallbackWebhookURL)}
}

func (r *automationRunner) Tick(ctx context.Context, now time.Time) (bool, error) {
	if r == nil || r.repo == nil {
		return false, nil
	}
	settings, err := r.repo.GetSettings(ctx)
	if err != nil {
		return false, err
	}
	if !jobs.ShouldSendDutyReport(settings, now) {
		return false, nil
	}
	webhookURL := strings.TrimSpace(settings.FeishuWebhookURL)
	if webhookURL == "" {
		webhookURL = r.fallbackWebhookURL
	}
	if webhookURL == "" {
		return false, nil
	}
	if _, err := r.repo.EscalateAgentTasks(ctx, now, settings); err != nil {
		return false, err
	}
	report, err := r.buildDutyReport(ctx)
	if err != nil {
		return false, err
	}
	if err := notify.SendFeishuWebhook(ctx, webhookURL, notify.BuildFeishuDutyReport(report)); err != nil {
		return false, err
	}
	sentAt := now.UTC()
	settings.LastDutyReportSentAt = &sentAt
	if _, err := r.repo.SaveSettings(ctx, settings); err != nil {
		return false, err
	}
	_, _ = r.repo.CreateAgentEvent(ctx, jobs.AgentEventInput{
		Type:    "auto_duty_report_sent",
		Title:   "Sent automatic duty report",
		Summary: "I sent the scheduled duty report from the automation scheduler.",
		Level:   "success",
	})
	return true, nil
}

func (r *automationRunner) buildDutyReport(ctx context.Context) (jobs.AgentDutyReport, error) {
	jobList, err := r.repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	sources, err := r.repo.ListSources(ctx, false)
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	runs, err := r.repo.ListRuns(ctx)
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	tasks, err := r.repo.ListAgentTasks(ctx, time.Now().UTC().Format("2006-01-02"))
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	report := jobs.BuildAgentDutyReport(jobList, sources, runs)
	return jobs.AddTasksToDutyReport(report, tasks), nil
}
