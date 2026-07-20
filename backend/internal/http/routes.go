package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(handlers *Handlers) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	api.GET("/agent/briefing", handlers.GetAgentBriefing)
	api.GET("/agent/report", handlers.GetAgentDutyReport)
	api.GET("/agent/events", handlers.ListAgentEvents)
	api.GET("/jobs", handlers.ListJobs)
	api.POST("/jobs/cleanup-landing-pages", handlers.CleanupLandingPages)
	api.POST("/jobs/import-url", handlers.ImportURL)
	api.GET("/jobs/:id", handlers.GetJob)
	api.PATCH("/jobs/:id/status", handlers.UpdateJobStatus)
	api.PATCH("/jobs/:id/notes", handlers.UpdateJobNotes)
	api.POST("/crawl/run", handlers.RunCrawl)
	api.GET("/crawl/runs", handlers.ListRuns)
	api.GET("/crawl/runs/:id/sources", handlers.ListRunSources)
	api.GET("/settings", handlers.GetSettings)
	api.PATCH("/settings", handlers.UpdateSettings)
	api.GET("/companies", handlers.ListCompanies)
	api.PATCH("/companies/:id", handlers.UpdateCompany)
	api.GET("/sources", handlers.ListSources)
	api.POST("/sources/recommended", handlers.SeedRecommendedSources)
	api.POST("/sources", handlers.CreateSource)
	api.PATCH("/sources/:id", handlers.UpdateSource)
	api.POST("/crawl/recommended", handlers.RunRecommendedCrawl)
	api.POST("/notifications/feishu/test", handlers.SendFeishuTest)
	api.POST("/notifications/feishu/report", handlers.SendFeishuReport)

	return router
}
