package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
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
	scored := jobs.ScoreJob(imported)
	if scored.HardFiltered {
		scored.Job.Status = domain.StatusManualCheck
		scored.Job.PenaltyReasons = append(scored.Job.PenaltyReasons, scored.HardFilterReason)
	}
	created, duplicate, err := h.Repo.UpsertJob(c.Request.Context(), scored.Job)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"job":         created,
		"duplicate":   duplicate,
		"manual_only": manualOnly,
	})
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
	c.JSON(http.StatusOK, summary)
}

func (h *Handlers) ListRuns(c *gin.Context) {
	runs, err := h.Repo.ListRuns(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, runs)
}

func (h *Handlers) GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"target_cities":     []string{"Shenzhen"},
		"target_directions": []string{"frontend", "backend", "java", "go", "algorithm", "ai_application"},
		"crawl_schedule":    []string{"09:00", "12:00", "18:00"},
		"feishu_configured": h.FeishuWebhookURL != "",
		"updated_at":        time.Now().UTC(),
	})
}

func (h *Handlers) UpdateSettings(c *gin.Context) {
	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settings payload"})
		return
	}
	c.JSON(http.StatusOK, raw)
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
