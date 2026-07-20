package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

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

func (h *Handlers) ListAgentEvents(c *gin.Context) {
	events, err := h.Repo.ListAgentEvents(c.Request.Context(), 20)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, events)
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
	c.JSON(http.StatusOK, settingsResponse(settings, h.FeishuWebhookURL != ""))
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
	c.JSON(http.StatusOK, settingsResponse(settings, h.FeishuWebhookURL != ""))
}

func settingsResponse(settings jobs.Settings, feishuConfigured bool) gin.H {
	return gin.H{
		"target_cities":     settings.TargetCities,
		"target_directions": settings.TargetDirections,
		"excluded_keywords": settings.ExcludedKeywords,
		"crawl_schedule":    settings.CrawlSchedule,
		"feishu_configured": feishuConfigured,
		"updated_at":        settings.UpdatedAt,
	}
}

func (h *Handlers) ListSources(c *gin.Context) {
	sources, err := h.Repo.ListSources(c.Request.Context(), false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, sources)
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
	if h.FeishuWebhookURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "FEISHU_WEBHOOK_URL is not configured"})
		return
	}
	err := notify.SendFeishuWebhook(c.Request.Context(), h.FeishuWebhookURL, "Job Hunter Agent test notification")
	if err != nil {
		respondError(c, http.StatusBadGateway, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "sent"})
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
