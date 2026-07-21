package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func BuildFeishuSummary(summary crawl.RunSummary, jobs []domain.Job) string {
	jobs = append([]domain.Job(nil), jobs...)
	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].MatchScore > jobs[j].MatchScore
	})
	strongMatches := 0
	for _, job := range jobs {
		if job.MatchScore >= 70 {
			strongMatches++
		}
	}

	var b strings.Builder
	b.WriteString("Job Hunter Agent update\n\n")
	b.WriteString(fmt.Sprintf("Jobs created: %d\n", summary.JobsCreated))
	b.WriteString(fmt.Sprintf("Strong matches: %d\n", strongMatches))
	b.WriteString(fmt.Sprintf("Manual check: %d\n", summary.ManualCheckCount))
	b.WriteString(fmt.Sprintf("Failed sources: %d\n", summary.SourcesFailed))

	limit := len(jobs)
	if limit > 10 {
		limit = 10
	}
	if limit == 0 {
		b.WriteString("\nNo recommended jobs in this run.")
		return b.String()
	}

	b.WriteString("\nTop recommendations:\n")
	for i := 0; i < limit; i++ {
		job := jobs[i]
		b.WriteString(fmt.Sprintf("%d. %s - %s - %s - %d\n", i+1, job.Company, job.Title, job.City, job.MatchScore))
		if len(job.RecommendReasons) > 0 {
			b.WriteString("   Reasons: ")
			b.WriteString(strings.Join(job.RecommendReasons, ", "))
			b.WriteString("\n")
		}
		link := job.ApplyURL
		if link == "" {
			link = job.SourceURL
		}
		if link != "" {
			b.WriteString("   Link: ")
			b.WriteString(link)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func BuildFeishuDutyReport(report jobs.AgentDutyReport) string {
	var b strings.Builder
	b.WriteString("Job Hunter Agent duty report\n\n")
	b.WriteString(report.Headline)
	b.WriteString("\n\nSummary:\n")
	b.WriteString(fmt.Sprintf("- New jobs: %d\n", report.Summary.NewJobs))
	b.WriteString(fmt.Sprintf("- Strong matches: %d\n", report.Summary.StrongMatches))
	b.WriteString(fmt.Sprintf("- Manual check: %d\n", report.Summary.ManualCheck))
	b.WriteString(fmt.Sprintf("- Source issues: %d\n", report.Summary.SourceIssues))
	b.WriteString(fmt.Sprintf("- Open tasks: %d\n", report.Summary.OpenTasks))
	b.WriteString(fmt.Sprintf("- Done tasks: %d\n", report.Summary.DoneTasks))
	b.WriteString(fmt.Sprintf("- Stale tasks: %d\n", report.Summary.StaleTasks))
	b.WriteString(fmt.Sprintf("- Escalated tasks: %d\n", report.Summary.EscalatedTasks))
	if strings.TrimSpace(report.TrendSummary) != "" {
		b.WriteString("\nTrend:\n")
		b.WriteString(report.TrendSummary)
		b.WriteString("\n")
	}
	if len(report.Tasks) > 0 {
		b.WriteString("\nDaily tasks:\n")
		written := 0
		for _, task := range report.Tasks {
			if task.Status == jobs.AgentTaskStatusDone {
				continue
			}
			b.WriteString(fmt.Sprintf("- [%s] %s: %s\n", task.Status, task.Title, task.Detail))
			written++
			if written >= 5 {
				break
			}
		}
	}
	if len(report.TodaysWork) > 0 {
		b.WriteString("\nToday's work:\n")
		for _, item := range report.TodaysWork {
			b.WriteString(fmt.Sprintf("- %s (%d): %s\n", item.Title, item.Count, item.Detail))
		}
	}
	if len(report.NeedsDecision) > 0 {
		b.WriteString("\nNeeds your decision:\n")
		limit := len(report.NeedsDecision)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			item := report.NeedsDecision[i]
			b.WriteString(fmt.Sprintf("- %s - %s - %s - score %d\n", item.Company, item.JobTitle, item.City, item.Score))
		}
	}
	if len(report.SourceIssues) > 0 {
		b.WriteString("\nSource issues:\n")
		limit := len(report.SourceIssues)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			issue := report.SourceIssues[i]
			b.WriteString(fmt.Sprintf("- %s: %s, %s\n", issue.Name, issue.Status, issue.Reason))
		}
	}
	b.WriteString("\nNext best action: ")
	b.WriteString(report.NextBestAction.Label)
	b.WriteString(" - ")
	b.WriteString(report.NextBestAction.Reason)
	return b.String()
}

func SendFeishuWebhook(ctx context.Context, webhookURL string, text string) error {
	if strings.TrimSpace(webhookURL) == "" {
		return fmt.Errorf("feishu webhook URL is empty")
	}
	payload := map[string]any{
		"msg_type": "text",
		"content": map[string]string{
			"text": text,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal feishu payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create feishu request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send feishu request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook returned status %d", resp.StatusCode)
	}
	return nil
}
