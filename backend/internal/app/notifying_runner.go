package app

import (
	"context"
	"log"
	"strings"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/notify"
)

type notifyingRunner struct {
	base             crawl.Runnable
	feishuWebhookURL string
}

func newNotifyingRunner(base crawl.Runnable, feishuWebhookURL string) crawl.Runnable {
	return &notifyingRunner{base: base, feishuWebhookURL: strings.TrimSpace(feishuWebhookURL)}
}

func (r *notifyingRunner) Run(ctx context.Context, trigger string) (crawl.RunSummary, error) {
	summary, err := r.base.Run(ctx, trigger)
	if err != nil {
		return summary, err
	}
	if r.feishuWebhookURL == "" || !shouldSendSummary(summary) {
		return summary, nil
	}
	text := notify.BuildFeishuSummary(summary, summary.RecommendedJobs)
	if err := notify.SendFeishuWebhook(ctx, r.feishuWebhookURL, text); err != nil {
		log.Printf("send Feishu crawl summary: %v", err)
	}
	return summary, nil
}

func shouldSendSummary(summary crawl.RunSummary) bool {
	return summary.JobsCreated > 0 || summary.ManualCheckCount > 0 || summary.SourcesFailed > 0
}
