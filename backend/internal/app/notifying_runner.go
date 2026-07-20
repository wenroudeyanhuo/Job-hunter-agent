package app

import (
	"context"
	"log"
	"strings"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/notify"
)

type notifyingRunner struct {
	base               crawl.Runnable
	repo               *jobs.Repository
	fallbackWebhookURL string
}

func newNotifyingRunner(base crawl.Runnable, repo *jobs.Repository, fallbackWebhookURL string) crawl.Runnable {
	return &notifyingRunner{base: base, repo: repo, fallbackWebhookURL: strings.TrimSpace(fallbackWebhookURL)}
}

func (r *notifyingRunner) Run(ctx context.Context, trigger string) (crawl.RunSummary, error) {
	summary, err := r.base.Run(ctx, trigger)
	if err != nil {
		return summary, err
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
