package app

import (
	"context"
	"fmt"
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
	ran := false
	if jobs.ShouldRunSourceDiscovery(settings, now) {
		result, err := r.repo.DiscoverSourceCandidates(ctx, jobs.SourceDiscoveryInput{
			TargetCities:     settings.TargetCities,
			TargetDirections: settings.TargetDirections,
		})
		if err != nil {
			return false, err
		}
		checkedAt := now.UTC()
		settings.LastSourceDiscoveryAt = &checkedAt
		if _, err := r.repo.SaveSettings(ctx, settings); err != nil {
			return false, err
		}
		ran = true
		_, _ = r.repo.CreateAgentEvent(ctx, jobs.AgentEventInput{
			Type:    "auto_source_discovery",
			Title:   "Ran automatic source discovery",
			Summary: "I proposed " + itoa(result.Created) + " new source candidates and skipped " + itoa(result.Duplicated) + " duplicates.",
			Level:   "success",
		})
	}
	if !jobs.ShouldSendDutyReport(settings, now) {
		return ran, nil
	}
	planned, err := r.ensureDailyWorkPlan(ctx, now)
	if err != nil {
		return false, err
	}
	if planned {
		ran = true
	}
	webhookURL := strings.TrimSpace(settings.FeishuWebhookURL)
	if webhookURL == "" {
		webhookURL = r.fallbackWebhookURL
	}
	if webhookURL == "" {
		return ran, nil
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
	if review, err := r.buildAgentReview(ctx); err == nil {
		_, _ = r.repo.CreateAgentReviewSnapshot(ctx, review, "automation_tick")
	}
	return true, nil
}

func (r *automationRunner) ensureDailyWorkPlan(ctx context.Context, now time.Time) (bool, error) {
	review, err := r.buildAgentReview(ctx)
	if err != nil {
		return false, err
	}
	input := jobs.BuildAgentPlanInputFromReview(review)
	input.Source = "automation"
	if len(input.Steps) == 0 {
		return false, nil
	}
	day := now.UTC().Format("2006-01-02")
	exists, err := r.repo.HasAgentPlanForDay(ctx, input.Source, input.Goal, day)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	plan, err := r.repo.CreateAgentPlan(ctx, input)
	if err != nil {
		return false, err
	}
	actions := make([]jobs.AgentCommandAction, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		actions = append(actions, jobs.AgentCommandAction{
			Type:   step.ActionType,
			Target: step.Target,
			Detail: step.Detail,
		})
	}
	if err := r.repo.RecordAgentActionRequestsForPlan(ctx, plan.ID, plan.Source, actions); err != nil {
		return false, err
	}
	_, _ = r.repo.CreateAgentEvent(ctx, jobs.AgentEventInput{
		Type:    "auto_plan_created",
		Title:   "Created automatic daily work plan",
		Summary: plan.Summary,
		Level:   "info",
	})
	return true, nil
}

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
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
	tasks, err := r.repo.ListAgentTasks(ctx, r.today(ctx))
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	report := jobs.BuildAgentDutyReport(jobList, sources, runs)
	report = jobs.AddTasksToDutyReport(report, tasks)
	snapshots, err := r.repo.ListAgentReviewSnapshots(ctx, 2)
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	report.TrendSummary = jobs.BuildAgentReviewHistory(snapshots).Summary
	return report, nil
}

func (r *automationRunner) buildAgentReview(ctx context.Context) (jobs.AgentReview, error) {
	jobList, err := r.repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		return jobs.AgentReview{}, err
	}
	sources, err := r.repo.ListSources(ctx, false)
	if err != nil {
		return jobs.AgentReview{}, err
	}
	runs, err := r.repo.ListRuns(ctx)
	if err != nil {
		return jobs.AgentReview{}, err
	}
	tasks, err := r.repo.ListAgentTasks(ctx, r.today(ctx))
	if err != nil {
		return jobs.AgentReview{}, err
	}
	return jobs.BuildAgentReview(jobList, sources, runs, tasks), nil
}

func (r *automationRunner) today(ctx context.Context) string {
	settings, err := r.repo.GetSettings(ctx)
	if err != nil {
		return time.Now().UTC().Format("2006-01-02")
	}
	loc, err := time.LoadLocation(settings.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}
