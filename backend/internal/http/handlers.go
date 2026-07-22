package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/importer"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/notify"
)

type CrawlRunner interface {
	Run(ctx context.Context, trigger string) (crawl.RunSummary, error)
}

type Handlers struct {
	Repo             *jobs.Repository
	Runner           CrawlRunner
	FeishuWebhookURL string
	LLM              jobs.LLMConfig
}

func (h *Handlers) GetAgentBriefing(c *gin.Context) {
	jobList, err := h.Repo.ListJobs(c.Request.Context(), jobs.ListFilter{})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	sources, err := h.Repo.ListSources(c.Request.Context(), false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	runs, err := h.Repo.ListRuns(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, jobs.BuildAgentBriefing(jobList, sources, runs))
}

func (h *Handlers) GetAgentState(c *gin.Context) {
	state, err := h.buildAgentState(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, state)
}

func (h *Handlers) RunAgentCommand(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid command payload"})
		return
	}
	settings, err := h.Repo.GetSettings(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	plan := jobs.PlanAgentCommand(req.Text, settings)
	if len(plan.Result.Actions) == 0 {
		c.JSON(http.StatusOK, plan.Result)
		return
	}
	if !sameStringSlice(plan.TargetCities, settings.TargetCities) || !sameStringSlice(plan.TargetDirections, settings.TargetDirections) {
		settings.TargetCities = plan.TargetCities
		settings.TargetDirections = plan.TargetDirections
		if _, err := h.Repo.SaveSettings(c.Request.Context(), settings); err != nil {
			respondError(c, http.StatusInternalServerError, err)
			return
		}
	}
	if plan.RunCrawl {
		if _, err := h.Runner.Run(c.Request.Context(), "command"); err != nil {
			respondError(c, http.StatusConflict, err)
			return
		}
	}
	if plan.RefreshTasks || plan.RunCrawl {
		if _, err := h.Repo.SyncAgentTasks(c.Request.Context(), time.Now().UTC()); err != nil {
			respondError(c, http.StatusInternalServerError, err)
			return
		}
	}
	if plan.SyncApplicationPlans {
		if _, err := h.Repo.SyncApplicationPlans(c.Request.Context(), time.Now().UTC()); err != nil {
			respondError(c, http.StatusInternalServerError, err)
			return
		}
	}
	if plan.SendFeishuReport {
		webhookURL, err := h.effectiveFeishuWebhookURL(c.Request.Context())
		if err != nil {
			respondError(c, http.StatusInternalServerError, err)
			return
		}
		if webhookURL == "" {
			plan.Result.Needs = append(plan.Result.Needs, "Configure Feishu webhook before sending duty reports.")
		} else {
			report, err := h.buildDutyReport(c.Request.Context())
			if err != nil {
				respondError(c, http.StatusInternalServerError, err)
				return
			}
			if err := notify.SendFeishuWebhook(c.Request.Context(), webhookURL, notify.BuildFeishuDutyReport(report)); err != nil {
				respondError(c, http.StatusBadGateway, err)
				return
			}
		}
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "agent_command_executed",
		Title:   "Executed command",
		Summary: plan.Result.Summary,
		Level:   "success",
	})
	c.JSON(http.StatusOK, plan.Result)
}

func (h *Handlers) GetAgentChatStatus(c *gin.Context) {
	c.JSON(http.StatusOK, jobs.BuildAgentChatStatus(h.LLM))
}

func (h *Handlers) GetAutomationStatus(c *gin.Context) {
	settings, err := h.Repo.GetSettings(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	webhookConfigured := strings.TrimSpace(settings.FeishuWebhookURL) != "" || strings.TrimSpace(h.FeishuWebhookURL) != ""
	c.JSON(http.StatusOK, jobs.BuildAgentAutomationDiagnostics(settings, webhookConfigured, true, time.Now().UTC()))
}

func (h *Handlers) ListAgentChatMessages(c *gin.Context) {
	messages, err := h.Repo.ListAgentChatMessages(c.Request.Context(), 30)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, messages)
}

func (h *Handlers) RunAgentChat(c *gin.Context) {
	var req struct {
		Message    string `json:"message"`
		ActiveView string `json:"active_view"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}
	if _, err := h.Repo.RecordAgentChatMessage(c.Request.Context(), jobs.AgentChatMessageInput{
		Role:    jobs.AgentChatRoleUser,
		Content: req.Message,
		Source:  "user",
	}); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	context, err := h.buildAgentChatContext(c.Request.Context(), req.ActiveView)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	reply := jobs.BuildLocalAgentChatReply(req.Message, context)
	if jobs.BuildAgentChatStatus(h.LLM).Configured {
		if modelReply, err := h.runModelChat(c.Request.Context(), req.Message, context); err == nil && strings.TrimSpace(modelReply) != "" {
			reply.Content = modelReply
			reply.Source = "model"
		} else if err != nil {
			reply.Content += "\n\n模型调用暂时失败，我先用本地规则继续工作：" + err.Error()
			reply.Source = "local_fallback"
		}
	}
	message, err := h.Repo.RecordAgentChatMessage(c.Request.Context(), jobs.AgentChatMessageInput{
		Role:    jobs.AgentChatRoleAssistant,
		Content: reply.Content,
		Source:  reply.Source,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"reply":   reply,
	})
}

func (h *Handlers) buildAgentChatContext(ctx context.Context, activeView string) (jobs.AgentChatContext, error) {
	jobList, err := h.Repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		return jobs.AgentChatContext{}, err
	}
	sources, err := h.Repo.ListSources(ctx, false)
	if err != nil {
		return jobs.AgentChatContext{}, err
	}
	tasks, err := h.Repo.ListAgentTasks(ctx, h.today(ctx))
	if err != nil {
		return jobs.AgentChatContext{}, err
	}
	context := jobs.AgentChatContext{
		ActiveView:   activeView,
		ModelEnabled: jobs.BuildAgentChatStatus(h.LLM).Configured,
	}
	for _, job := range jobList {
		if job.MatchScore >= 70 {
			context.StrongMatches++
		}
		if job.Status == domain.StatusManualCheck {
			context.ManualDecisions++
		}
	}
	for _, task := range tasks {
		if task.Status != jobs.AgentTaskStatusDone {
			context.OpenTasks++
		}
	}
	for _, source := range sources {
		if source.HealthStatus == jobs.SourceHealthWarning || source.HealthStatus == jobs.SourceHealthBroken {
			context.SourceIssues++
		}
	}
	return context, nil
}

func (h *Handlers) runModelChat(ctx context.Context, userMessage string, chatContext jobs.AgentChatContext) (string, error) {
	config := jobs.NormalizeLLMConfig(h.LLM)
	if config.APIKey == "" || config.Model == "" {
		return "", fmt.Errorf("model is not configured")
	}
	payload := map[string]any{
		"model": config.Model,
		"messages": []map[string]string{
			{
				"role": "system",
				"content": fmt.Sprintf("You are Job Hunter Agent, a Chinese-speaking digital employee for autumn recruitment. Be concise, practical, and use the current local context. Current view: %s. Open tasks: %d. Strong matches: %d. Manual decisions: %d. Source issues: %d.",
					chatContext.ActiveView, chatContext.OpenTasks, chatContext.StrongMatches, chatContext.ManualDecisions, chatContext.SourceIssues),
			},
			{"role": "user", "content": userMessage},
		},
		"temperature": 0.4,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode model request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create model request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call model: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("model returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", fmt.Errorf("decode model response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("model returned no choices")
	}
	return strings.TrimSpace(decoded.Choices[0].Message.Content), nil
}

func (h *Handlers) RunAutomationDutyReport(c *gin.Context) {
	settings, err := h.Repo.GetSettings(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if !settings.AutoDutyReportEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auto duty report is not enabled"})
		return
	}
	webhookURL, err := h.effectiveFeishuWebhookURL(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if webhookURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feishu webhook URL is not configured"})
		return
	}
	report, err := h.buildDutyReport(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if err := notify.SendFeishuWebhook(c.Request.Context(), webhookURL, notify.BuildFeishuDutyReport(report)); err != nil {
		respondError(c, http.StatusBadGateway, err)
		return
	}
	now := time.Now().UTC()
	settings.LastDutyReportSentAt = &now
	if _, err := h.Repo.SaveSettings(c.Request.Context(), settings); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "auto_duty_report_sent",
		Title:   "Sent automatic duty report",
		Summary: "I sent the scheduled duty report and updated the last sent time.",
		Level:   "success",
	})
	h.snapshotAgentReview(c, "auto_duty_report_sent")
	c.JSON(http.StatusOK, gin.H{"status": "sent", "sent_at": now})
}

func (h *Handlers) GetAgentDutyReport(c *gin.Context) {
	report, err := h.buildDutyReport(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handlers) GetAgentReview(c *gin.Context) {
	review, err := h.buildAgentReview(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, review)
}

func (h *Handlers) CreateAgentReviewSnapshot(c *gin.Context) {
	var req struct {
		TriggerType string `json:"trigger_type"`
	}
	_ = c.ShouldBindJSON(&req)
	review, err := h.buildAgentReview(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	snapshot, err := h.Repo.CreateAgentReviewSnapshot(c.Request.Context(), review, req.TriggerType)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "agent_review_snapshotted",
		Title:   "Saved review snapshot",
		Summary: "I saved the current review so future trend reports can compare progress.",
		Level:   "info",
	})
	c.JSON(http.StatusCreated, snapshot)
}

func (h *Handlers) ListAgentReviewHistory(c *gin.Context) {
	snapshots, err := h.Repo.ListAgentReviewSnapshots(c.Request.Context(), 14)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, jobs.BuildAgentReviewHistory(snapshots))
}

func (h *Handlers) buildAgentReview(ctx context.Context) (jobs.AgentReview, error) {
	jobList, err := h.Repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		return jobs.AgentReview{}, err
	}
	sources, err := h.Repo.ListSources(ctx, false)
	if err != nil {
		return jobs.AgentReview{}, err
	}
	runs, err := h.Repo.ListRuns(ctx)
	if err != nil {
		return jobs.AgentReview{}, err
	}
	tasks, err := h.Repo.ListAgentTasks(ctx, h.today(ctx))
	if err != nil {
		return jobs.AgentReview{}, err
	}
	return jobs.BuildAgentReview(jobList, sources, runs, tasks), nil
}

func (h *Handlers) buildAgentState(ctx context.Context) (jobs.AgentState, error) {
	jobList, err := h.Repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		return jobs.AgentState{}, err
	}
	sources, err := h.Repo.ListSources(ctx, false)
	if err != nil {
		return jobs.AgentState{}, err
	}
	runs, err := h.Repo.ListRuns(ctx)
	if err != nil {
		return jobs.AgentState{}, err
	}
	tasks, err := h.Repo.ListAgentTasks(ctx, h.today(ctx))
	if err != nil {
		return jobs.AgentState{}, err
	}
	settings, err := h.Repo.GetSettings(ctx)
	if err != nil {
		return jobs.AgentState{}, err
	}
	if strings.TrimSpace(settings.FeishuWebhookURL) == "" {
		settings.FeishuWebhookURL = strings.TrimSpace(h.FeishuWebhookURL)
	}
	return jobs.BuildAgentState(jobList, sources, runs, tasks, settings), nil
}

func (h *Handlers) ListAgentEvents(c *gin.Context) {
	events, err := h.Repo.ListAgentEvents(c.Request.Context(), 20)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *Handlers) ListAgentTasks(c *gin.Context) {
	tasks, err := h.Repo.ListAgentTasks(c.Request.Context(), h.today(c.Request.Context()))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *Handlers) RefreshAgentTasks(c *gin.Context) {
	tasks, err := h.Repo.SyncAgentTasks(c.Request.Context(), time.Now().UTC())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	settings, err := h.Repo.GetSettings(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if _, err := h.Repo.EscalateAgentTasks(c.Request.Context(), time.Now().UTC(), settings); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	tasks, err = h.Repo.ListAgentTasks(c.Request.Context(), h.today(c.Request.Context()))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "agent_tasks_refreshed",
		Title:   "Refreshed daily tasks",
		Summary: "I rebuilt today's recruiting work queue from jobs, sources, and crawl history.",
		Level:   "info",
	})
	h.snapshotAgentReview(c, "tasks_refreshed")
	c.JSON(http.StatusOK, tasks)
}

func (h *Handlers) UpdateAgentTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Status           string     `json:"status"`
		CompletionReason string     `json:"completion_reason"`
		SnoozedUntil     *time.Time `json:"snoozed_until"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Status) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	if err := h.Repo.UpdateAgentTask(c.Request.Context(), id, jobs.AgentTaskUpdate{
		Status:           req.Status,
		CompletionReason: req.CompletionReason,
		SnoozedUntil:     req.SnoozedUntil,
	}); err != nil {
		respondRepoError(c, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "agent_task_updated",
		Title:   "Updated daily task",
		Summary: "You marked task #" + strconv.FormatInt(id, 10) + " as " + req.Status + ".",
		Level:   "info",
	})
	c.Status(http.StatusNoContent)
}

func (h *Handlers) ListApplicationPlans(c *gin.Context) {
	plans, err := h.Repo.ListApplicationPlans(c.Request.Context(), c.Query("status"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, plans)
}

func (h *Handlers) SyncApplicationPlans(c *gin.Context) {
	plans, err := h.Repo.SyncApplicationPlans(c.Request.Context(), time.Now().UTC())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "application_plans_synced",
		Title:   "Synced application plans",
		Summary: "I prepared " + strconv.Itoa(len(plans)) + " application plans for interested strong matches.",
		Level:   "success",
	})
	c.JSON(http.StatusOK, plans)
}

func (h *Handlers) UpdateApplicationPlan(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req jobs.ApplicationPlanUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application plan payload"})
		return
	}
	plan, err := h.Repo.UpdateApplicationPlan(c.Request.Context(), id, req)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "application_plan_updated",
		Title:   "Updated application plan",
		Summary: "Application plan #" + strconv.FormatInt(plan.ID, 10) + " is now " + plan.Status + ".",
		Level:   "info",
	})
	c.JSON(http.StatusOK, plan)
}

func (h *Handlers) ListJobs(c *gin.Context) {
	filter := jobs.ListFilter{Status: domain.JobStatus(c.Query("status"))}
	list, err := h.Repo.ListJobs(c.Request.Context(), filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) GetCandidateProfile(c *gin.Context) {
	profile, err := h.Repo.GetCandidateProfile(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handlers) UpdateCandidateProfile(c *gin.Context) {
	var req jobs.CandidateProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile payload"})
		return
	}
	profile, err := h.Repo.SaveCandidateProfile(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "candidate_profile_updated",
		Title:   "Updated candidate profile",
		Summary: "I updated your target cities, directions, skills, and preference signals.",
		Level:   "success",
	})
	c.JSON(http.StatusOK, profile)
}

func (h *Handlers) GetJob(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	job, err := h.Repo.GetJob(c.Request.Context(), id)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *Handlers) GetJobDetail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	job, err := h.Repo.GetJob(c.Request.Context(), id)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	profile, err := h.Repo.GetCandidateProfile(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	decisions, err := h.Repo.ListJobDecisions(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	detail := jobs.BuildJobDetail(job, profile, decisions)
	if plan, found, err := h.Repo.GetApplicationPlanByJobID(c.Request.Context(), id); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	} else if found {
		detail.ApplicationPlan = &plan
	}
	c.JSON(http.StatusOK, detail)
}

func (h *Handlers) ImportURL(c *gin.Context) {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}
	imported, err := importer.ImportURL(c.Request.Context(), req.URL, http.DefaultClient)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	manualOnly := imported.Status == domain.StatusManualCheck
	settings, err := h.Repo.GetSettings(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	scored := jobs.ScoreJobWithSettings(imported, settings)
	if scored.HardFiltered {
		scored.Job.Status = domain.StatusManualCheck
		scored.Job.PenaltyReasons = append(scored.Job.PenaltyReasons, scored.HardFilterReason)
	}
	created, duplicate, err := h.Repo.UpsertJob(c.Request.Context(), scored.Job)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if duplicate {
		h.recordAgentEvent(c, jobs.AgentEventInput{
			Type:    "job_import_duplicate",
			Title:   "Imported link was already tracked",
			Summary: "I checked the pasted recruitment link and found it already exists: " + created.Title,
			Level:   "info",
		})
	} else {
		h.recordAgentEvent(c, jobs.AgentEventInput{
			Type:    "job_imported",
			Title:   "Imported a recruitment link",
			Summary: "I saved and scored: " + created.Title,
			Level:   "success",
		})
	}
	c.JSON(http.StatusCreated, gin.H{
		"job":         created,
		"duplicate":   duplicate,
		"manual_only": manualOnly,
	})
}

func (h *Handlers) CleanupLandingPages(c *gin.Context) {
	result, err := h.cleanupLandingPages(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "landing_pages_cleaned",
		Title:   "Cleaned recruitment landing pages",
		Summary: "I moved " + strconv.Itoa(result.Ignored) + " non-job recruitment pages to ignored.",
		Level:   "info",
	})
	c.JSON(http.StatusOK, gin.H{"ignored": result.Ignored})
}

type landingPageCleanupResult struct {
	Ignored int
}

func (h *Handlers) cleanupLandingPages(ctx context.Context) (landingPageCleanupResult, error) {
	jobList, err := h.Repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		return landingPageCleanupResult{}, err
	}

	result := landingPageCleanupResult{}
	for _, job := range jobList {
		if job.Status == domain.StatusIgnored || job.Status == domain.StatusApplied || job.Status == domain.StatusInterested {
			continue
		}
		if importer.LooksLikeConcreteJobPosting(job) {
			continue
		}
		if err := h.Repo.UpdateStatus(ctx, job.ID, domain.StatusIgnored); err != nil {
			return landingPageCleanupResult{}, err
		}
		result.Ignored++
	}
	return result, nil
}

func (h *Handlers) UpdateJobStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Status domain.JobStatus `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	if err := h.Repo.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		respondRepoError(c, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "job_status_updated",
		Title:   "Updated job status",
		Summary: "You marked job #" + strconv.FormatInt(id, 10) + " as " + string(req.Status) + ".",
		Level:   "info",
	})
	c.Status(http.StatusNoContent)
}

func (h *Handlers) UpdateJobNotes(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notes payload"})
		return
	}
	if err := h.Repo.UpdateNotes(c.Request.Context(), id, req.Notes); err != nil {
		respondRepoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) RunCrawl(c *gin.Context) {
	summary, err := h.Runner.Run(c.Request.Context(), "manual")
	if err != nil {
		respondError(c, http.StatusConflict, err)
		return
	}
	cleanup, err := h.cleanupLandingPages(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	summary.LandingPagesIgnored = cleanup.Ignored
	h.recordCrawlEvent(c, "crawl_completed", "Manual crawl completed", summary)
	h.refreshAgentTasksAfterCrawl(c)
	h.snapshotAgentReview(c, "crawl_completed")
	c.JSON(http.StatusOK, summary)
}

func (h *Handlers) RunRecommendedCrawl(c *gin.Context) {
	seeded, err := h.Repo.SeedRecommendedSources(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	summary, err := h.Runner.Run(c.Request.Context(), "recommended")
	if err != nil {
		respondError(c, http.StatusConflict, err)
		return
	}
	cleanup, err := h.cleanupLandingPages(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	summary.LandingPagesIgnored = cleanup.Ignored
	h.recordCrawlEvent(c, "recommended_crawl_completed", "Recommended crawl completed", summary)
	h.refreshAgentTasksAfterCrawl(c)
	h.snapshotAgentReview(c, "recommended_crawl_completed")
	c.JSON(http.StatusOK, gin.H{
		"seeded":  seeded.Created,
		"sources": seeded,
		"summary": summary,
	})
}

func (h *Handlers) ListRuns(c *gin.Context) {
	runs, err := h.Repo.ListRuns(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, runs)
}

func (h *Handlers) ListRunSources(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	results, err := h.Repo.ListRunSources(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, results)
}

func (h *Handlers) GetSettings(c *gin.Context) {
	settings, err := h.Repo.GetSettings(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, h.settingsResponse(settings))
}

func (h *Handlers) UpdateSettings(c *gin.Context) {
	var req jobs.Settings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settings payload"})
		return
	}
	settings, err := h.Repo.SaveSettings(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, h.settingsResponse(settings))
}

func (h *Handlers) settingsResponse(settings jobs.Settings) gin.H {
	webhookURL := strings.TrimSpace(settings.FeishuWebhookURL)
	return gin.H{
		"target_cities":                   settings.TargetCities,
		"target_directions":               settings.TargetDirections,
		"excluded_keywords":               settings.ExcludedKeywords,
		"crawl_schedule":                  settings.CrawlSchedule,
		"feishu_webhook_url":              webhookURL,
		"feishu_configured":               webhookURL != "" || strings.TrimSpace(h.FeishuWebhookURL) != "",
		"time_zone":                       settings.TimeZone,
		"auto_duty_report_enabled":        settings.AutoDutyReportEnabled,
		"auto_source_discovery_enabled":   settings.AutoSourceDiscoveryEnabled,
		"source_discovery_interval_hours": settings.SourceDiscoveryIntervalHours,
		"duty_report_time":                settings.DutyReportTime,
		"task_sla_hours":                  settings.TaskSLAHours,
		"last_duty_report_sent_at":        settings.LastDutyReportSentAt,
		"last_source_discovery_at":        settings.LastSourceDiscoveryAt,
		"updated_at":                      settings.UpdatedAt,
	}
}

func (h *Handlers) today(ctx context.Context) string {
	settings, err := h.Repo.GetSettings(ctx)
	if err != nil {
		return time.Now().UTC().Format("2006-01-02")
	}
	loc, err := time.LoadLocation(settings.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func (h *Handlers) ListSources(c *gin.Context) {
	sources, err := h.Repo.ListSources(c.Request.Context(), false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, sources)
}

func (h *Handlers) RunSourceDiscovery(c *gin.Context) {
	var req jobs.SourceDiscoveryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discovery payload"})
		return
	}
	if len(req.TargetCities) == 0 || len(req.TargetDirections) == 0 {
		settings, err := h.Repo.GetSettings(c.Request.Context())
		if err != nil {
			respondError(c, http.StatusInternalServerError, err)
			return
		}
		if len(req.TargetCities) == 0 {
			req.TargetCities = settings.TargetCities
		}
		if len(req.TargetDirections) == 0 {
			req.TargetDirections = settings.TargetDirections
		}
	}
	result, err := h.Repo.DiscoverSourceCandidates(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "source_candidates_discovered",
		Title:   "Discovered source candidates",
		Summary: "I proposed " + strconv.Itoa(result.Created) + " new source candidates and skipped " + strconv.Itoa(result.Duplicated) + " duplicates.",
		Level:   "success",
	})
	c.JSON(http.StatusCreated, result)
}

func (h *Handlers) ListSourceCandidates(c *gin.Context) {
	candidates, err := h.Repo.ListSourceCandidates(c.Request.Context(), jobs.SourceCandidateFilter{Status: c.Query("status")})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, candidates)
}

func (h *Handlers) AcceptSourceCandidate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	candidate, source, err := h.Repo.AcceptSourceCandidate(c.Request.Context(), id)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "source_candidate_accepted",
		Title:   "Accepted source candidate",
		Summary: "I promoted " + candidate.Name + " into active crawl sources.",
		Level:   "success",
	})
	c.JSON(http.StatusCreated, gin.H{"candidate": candidate, "source": source})
}

func (h *Handlers) ValidateSourceCandidate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	candidate, err := h.Repo.ValidateSourceCandidate(c.Request.Context(), id, nil)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	level := "info"
	if candidate.ValidationStatus == jobs.SourceCandidateValidationGood {
		level = "success"
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "source_candidate_validated",
		Title:   "Validated source candidate",
		Summary: candidate.Name + ": " + candidate.ValidationReason,
		Level:   level,
	})
	c.JSON(http.StatusOK, candidate)
}

func (h *Handlers) RejectSourceCandidate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	candidate, err := h.Repo.UpdateSourceCandidateStatus(c.Request.Context(), id, jobs.SourceCandidateStatusRejected)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "source_candidate_rejected",
		Title:   "Rejected source candidate",
		Summary: "I will stop recommending " + candidate.Name + " unless discovery logic changes later.",
		Level:   "info",
	})
	c.JSON(http.StatusOK, candidate)
}

func (h *Handlers) ListCompanies(c *gin.Context) {
	companies, err := h.Repo.ListCompanies(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, companies)
}

func (h *Handlers) UpdateCompany(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enabled is required"})
		return
	}
	if err := h.Repo.UpdateCompanyEnabled(c.Request.Context(), id, *req.Enabled); err != nil {
		respondRepoError(c, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "company_scope_updated",
		Title:   "Updated company scope",
		Summary: "You " + enabledVerb(*req.Enabled) + " company #" + strconv.FormatInt(id, 10) + " for future crawls.",
		Level:   "info",
	})
	c.Status(http.StatusNoContent)
}

func (h *Handlers) CreateSource(c *gin.Context) {
	var req jobs.SourceInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source payload"})
		return
	}
	source, err := h.Repo.CreateSource(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, source)
}

func enabledVerb(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func (h *Handlers) SeedRecommendedSources(c *gin.Context) {
	result, err := h.Repo.SeedRecommendedSources(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if result.Created > 0 {
		h.recordAgentEvent(c, jobs.AgentEventInput{
			Type:    "recommended_sources_added",
			Title:   "Added recommended sources",
			Summary: "I added " + strconv.Itoa(result.Created) + " official recruitment sources.",
			Level:   "success",
		})
	}
	status := http.StatusCreated
	if result.Created == 0 {
		status = http.StatusOK
	}
	c.JSON(status, result)
}

func (h *Handlers) UpdateSource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enabled is required"})
		return
	}
	if err := h.Repo.UpdateSourceEnabled(c.Request.Context(), id, *req.Enabled); err != nil {
		respondRepoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) SendFeishuTest(c *gin.Context) {
	webhookURL, err := h.effectiveFeishuWebhookURL(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if webhookURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feishu webhook URL is not configured"})
		return
	}
	err = notify.SendFeishuWebhook(c.Request.Context(), webhookURL, "Job Hunter Agent test notification")
	if err != nil {
		respondError(c, http.StatusBadGateway, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (h *Handlers) SendFeishuReport(c *gin.Context) {
	webhookURL, err := h.effectiveFeishuWebhookURL(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if webhookURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feishu webhook URL is not configured"})
		return
	}
	report, err := h.buildDutyReport(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if err := notify.SendFeishuWebhook(c.Request.Context(), webhookURL, notify.BuildFeishuDutyReport(report)); err != nil {
		respondError(c, http.StatusBadGateway, err)
		return
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    "feishu_report_sent",
		Title:   "Sent duty report to Feishu",
		Summary: "I sent the current work queue, decisions, and source issues to your Feishu bot.",
		Level:   "success",
	})
	h.snapshotAgentReview(c, "feishu_report_sent")
	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (h *Handlers) buildDutyReport(ctx context.Context) (jobs.AgentDutyReport, error) {
	jobList, err := h.Repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	sources, err := h.Repo.ListSources(ctx, false)
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	runs, err := h.Repo.ListRuns(ctx)
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	report := jobs.BuildAgentDutyReport(jobList, sources, runs)
	tasks, err := h.Repo.ListAgentTasks(ctx, h.today(ctx))
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	report = jobs.AddTasksToDutyReport(report, tasks)
	snapshots, err := h.Repo.ListAgentReviewSnapshots(ctx, 2)
	if err != nil {
		return jobs.AgentDutyReport{}, err
	}
	report.TrendSummary = jobs.BuildAgentReviewHistory(snapshots).Summary
	return report, nil
}

func (h *Handlers) effectiveFeishuWebhookURL(ctx context.Context) (string, error) {
	settings, err := h.Repo.GetSettings(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(settings.FeishuWebhookURL) != "" {
		return strings.TrimSpace(settings.FeishuWebhookURL), nil
	}
	return strings.TrimSpace(h.FeishuWebhookURL), nil
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

func respondRepoError(c *gin.Context, err error) {
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, err)
		return
	}
	respondError(c, http.StatusInternalServerError, err)
}

func respondError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"error": err.Error()})
}

func (h *Handlers) recordCrawlEvent(c *gin.Context, eventType string, title string, summary crawl.RunSummary) {
	level := "success"
	if summary.SourcesFailed > 0 {
		level = "warning"
	}
	cleaned := ""
	if summary.LandingPagesIgnored > 0 {
		cleaned = " I also moved " + strconv.Itoa(summary.LandingPagesIgnored) + " recruitment landing pages to ignored."
	}
	h.recordAgentEvent(c, jobs.AgentEventInput{
		Type:    eventType,
		Title:   title,
		Summary: "Created " + strconv.Itoa(summary.JobsCreated) + " jobs, found " + strconv.Itoa(summary.JobsDuplicated) + " duplicates, and flagged " + strconv.Itoa(summary.ManualCheckCount) + " for review." + cleaned,
		Level:   level,
	})
}

func (h *Handlers) recordAgentEvent(c *gin.Context, input jobs.AgentEventInput) {
	if _, err := h.Repo.CreateAgentEvent(c.Request.Context(), input); err != nil {
		_ = c.Error(err)
	}
}

func (h *Handlers) refreshAgentTasksAfterCrawl(c *gin.Context) {
	if _, err := h.Repo.SyncAgentTasks(c.Request.Context(), time.Now().UTC()); err != nil {
		_ = c.Error(err)
	}
}

func (h *Handlers) snapshotAgentReview(c *gin.Context, triggerType string) {
	review, err := h.buildAgentReview(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	if _, err := h.Repo.CreateAgentReviewSnapshot(c.Request.Context(), review, triggerType); err != nil {
		_ = c.Error(err)
	}
}

func sameStringSlice(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
