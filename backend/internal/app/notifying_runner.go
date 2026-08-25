package app

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/notify"
)

type notifyingRunner struct {
	base               crawl.Runnable
	repo               *jobs.Repository
	fallbackWebhookURL string
	orchestrator       jobs.RecruitingOrchestrator
}

func newNotifyingRunner(base crawl.Runnable, repo *jobs.Repository, fallbackWebhookURL string, orchestrator jobs.RecruitingOrchestrator) crawl.Runnable {
	return &notifyingRunner{base: base, repo: repo, fallbackWebhookURL: strings.TrimSpace(fallbackWebhookURL), orchestrator: orchestrator}
}

func (r *notifyingRunner) Run(ctx context.Context, trigger string) (crawl.RunSummary, error) {
	summary, err := r.base.Run(ctx, trigger)
	if err != nil {
		return summary, err
	}
	if r.repo != nil && strings.HasPrefix(strings.TrimSpace(trigger), "scheduled") {
		if _, err := runAndRecordAgentCycle(ctx, r.repo, trigger, timeNowUTC(), r.orchestrator); err != nil {
			log.Printf("record scheduled agent cycle: %v", err)
		}
	}
	webhookURL := r.effectiveFeishuWebhookURL(ctx)
	if webhookURL == "" || !shouldSendSummary(summary) {
		return summary, nil
	}
	text := notify.BuildFeishuSummary(summary, summary.RecommendedJobs)
	if err := notify.SendFeishuWebhook(ctx, webhookURL, text); err != nil {
		log.Printf("send Feishu crawl summary: %v", err)
	}
	return summary, nil
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}

func (r *notifyingRunner) effectiveFeishuWebhookURL(ctx context.Context) string {
	if r.repo != nil {
		settings, err := r.repo.GetSettings(ctx)
		if err == nil && strings.TrimSpace(settings.FeishuWebhookURL) != "" {
			return strings.TrimSpace(settings.FeishuWebhookURL)
		}
	}
	return r.fallbackWebhookURL
}

func shouldSendSummary(summary crawl.RunSummary) bool {
	return summary.JobsCreated > 0 || summary.ManualCheckCount > 0 || summary.SourcesFailed > 0
}
