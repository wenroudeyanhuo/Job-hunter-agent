package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/notify"
)

type AgentToolExecutor struct {
	handlers *Handlers
	registry jobs.AgentToolRegistry
}

func NewAgentToolExecutor(handlers *Handlers) AgentToolExecutor {
	return AgentToolExecutor{
		handlers: handlers,
		registry: jobs.NewDefaultAgentToolRegistry(),
	}
}

func (e AgentToolExecutor) Execute(c *gin.Context, request jobs.AgentActionRequest) (string, error) {
	tool, ok := e.registry.Get(request.ActionType)
	if !ok {
		return "", fmt.Errorf("unsupported agent action: %s", request.ActionType)
	}
	ctx := c.Request.Context()
	h := e.handlers
	message := "Action completed."
	switch tool.Name {
	case "add_recommended_and_crawl":
		if h.Runner == nil {
			return "", fmt.Errorf("crawl runner is not configured")
		}
		seeded, err := h.Repo.SeedRecommendedSources(ctx)
		if err != nil {
			return "", err
		}
		summary, err := h.Runner.Run(ctx, "agent_action_recommended_crawl")
		if err != nil {
			return "", err
		}
		cleanup, err := h.cleanupLandingPages(ctx)
		if err != nil {
			return "", err
		}
		summary.LandingPagesIgnored = cleanup.Ignored
		h.recordCrawlEvent(c, "agent_action_recommended_crawl_completed", "Agent recommended source crawl completed", summary)
		h.refreshAgentTasksAfterCrawl(c)
		h.snapshotAgentReview(c, "agent_action_recommended_crawl_completed")
		message = "Seeded " + strconv.Itoa(seeded.Created) + " recommended sources, skipped " + strconv.Itoa(seeded.Duplicated) + " duplicates, created " + strconv.Itoa(summary.JobsCreated) + " jobs, and flagged " + strconv.Itoa(summary.ManualCheckCount) + " for review."
	case "run_crawl":
		if h.Runner == nil {
			return "", fmt.Errorf("crawl runner is not configured")
		}
		summary, err := h.Runner.Run(ctx, "agent_action")
		if err != nil {
			return "", err
		}
		cleanup, err := h.cleanupLandingPages(ctx)
		if err != nil {
			return "", err
		}
		summary.LandingPagesIgnored = cleanup.Ignored
		h.recordCrawlEvent(c, "agent_action_crawl_completed", "Agent action crawl completed", summary)
		h.refreshAgentTasksAfterCrawl(c)
		h.snapshotAgentReview(c, "agent_action_crawl_completed")
		message = "Created " + strconv.Itoa(summary.JobsCreated) + " jobs, found " + strconv.Itoa(summary.JobsDuplicated) + " duplicates, and flagged " + strconv.Itoa(summary.ManualCheckCount) + " for review."
	case "refresh_tasks":
		if _, err := h.Repo.SyncAgentTasks(ctx, time.Now().UTC()); err != nil {
			return "", err
		}
		message = "Refreshed today's task queue."
	case "sync_application_plans":
		plans, err := h.Repo.SyncApplicationPlans(ctx, time.Now().UTC())
		if err != nil {
			return "", err
		}
		message = "Synced " + strconv.Itoa(len(plans)) + " application plans."
	case "send_feishu_report":
		webhookURL, err := h.effectiveFeishuWebhookURL(ctx)
		if err != nil {
			return "", err
		}
		if webhookURL == "" {
			return "", fmt.Errorf("Feishu webhook URL is not configured")
		}
		report, err := h.buildDutyReport(ctx)
		if err != nil {
			return "", err
		}
		if err := notify.SendFeishuWebhook(ctx, webhookURL, notify.BuildFeishuDutyReportWithCycle(report, h.latestAgentCycle(ctx))); err != nil {
			return "", err
		}
		message = "Sent the current duty report to Feishu."
	case "discover_sources":
		settings, err := h.Repo.GetSettings(ctx)
		if err != nil {
			return "", err
		}
		result, err := h.Repo.DiscoverSourceCandidates(ctx, jobs.SourceDiscoveryInput{
			TargetCities:     settings.TargetCities,
			TargetDirections: settings.TargetDirections,
			EnableWebSearch:  true,
			SearchLimit:      6,
		})
		if err != nil {
			return "", err
		}
		message = "Discovered " + strconv.Itoa(result.Created) + " new source candidates, found " + strconv.Itoa(result.WebSearchCandidates) + " via active web search, and skipped " + strconv.Itoa(result.Duplicated) + " duplicates."
	case "generate_daily_plan":
		review, err := h.buildAgentReview(ctx)
		if err != nil {
			return "", err
		}
		result, err := jobs.NewAgentRuntime(h.Repo).CreateReviewPlan(ctx, jobs.AgentReviewPlanRequest{
			Review: review,
			Source: "agent_tool",
			Now:    time.Now().UTC(),
		})
		if err != nil {
			return "", err
		}
		if !result.Created {
			message = "No daily plan was created because there were no actionable steps."
		} else {
			message = "Generated daily plan " + strconv.FormatInt(result.Plan.ID, 10) + " with " + strconv.Itoa(len(result.Plan.Steps)) + " steps."
		}
	case "inspect_source_health":
		summary, err := h.Repo.BuildSourceOperationsSummary(ctx)
		if err != nil {
			return "", err
		}
		message = "Inspected " + strconv.Itoa(summary.TotalSources) + " sources: " + strconv.Itoa(summary.HealthySources) + " healthy, " + strconv.Itoa(summary.WarningSources+summary.BrokenSources) + " unhealthy."
	case "validate_source_candidates":
		candidates, err := h.Repo.ListSourceCandidates(ctx, jobs.SourceCandidateFilter{Status: jobs.SourceCandidateStatusPending})
		if err != nil {
			return "", err
		}
		limit := 5
		validated := 0
		for _, candidate := range candidates {
			if validated >= limit {
				break
			}
			if _, err := h.Repo.ValidateSourceCandidate(ctx, candidate.ID, http.DefaultClient); err != nil {
				return "", err
			}
			validated++
		}
		message = "Validated " + strconv.Itoa(validated) + " pending source candidates."
	case "rebuild_semantic_memory":
		result, err := h.Repo.RebuildSemanticMemory(ctx)
		if err != nil {
			return "", err
		}
		reflections, err := h.Repo.RefreshMemoryReflections(ctx)
		if err != nil {
			return "", err
		}
		message = "Rebuilt semantic memory with " + strconv.Itoa(result.Created) + " indexed items, refreshed " + strconv.Itoa(reflections.Created) + " preference reflections, and skipped " + strconv.Itoa(result.Skipped) + "."
	case "review_strong_matches", "review_manual_check", "review_parser_gaps":
		message = "Opened the requested review workflow."
	default:
		return "", fmt.Errorf("agent tool %s is registered but has no executor", tool.Name)
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "agent_action_executed",
		Title:   "Executed approved action",
		Summary: request.ActionType + ": " + message,
		Level:   "success",
	})
	return message, nil
}
