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
	api.GET("/agent/state", handlers.GetAgentState)
	api.POST("/agent/commands", handlers.RunAgentCommand)
	api.GET("/agent/plans", handlers.ListAgentPlans)
	api.GET("/agent/actions", handlers.ListAgentActionRequests)
	api.PATCH("/agent/actions/:id", handlers.UpdateAgentActionRequest)
	api.GET("/agent/chat/status", handlers.GetAgentChatStatus)
	api.POST("/agent/chat/healthcheck", handlers.CheckAgentChatModel)
	api.GET("/agent/chat/messages", handlers.ListAgentChatMessages)
	api.POST("/agent/chat", handlers.RunAgentChat)
	api.POST("/agent/automation/duty-report", handlers.RunAutomationDutyReport)
	api.GET("/agent/automation/status", handlers.GetAutomationStatus)
	api.GET("/agent/report", handlers.GetAgentDutyReport)
	api.GET("/agent/review", handlers.GetAgentReview)
	api.POST("/agent/review/snapshot", handlers.CreateAgentReviewSnapshot)
	api.GET("/agent/review/history", handlers.ListAgentReviewHistory)
	api.GET("/agent/events", handlers.ListAgentEvents)
	api.GET("/agent/tasks", handlers.ListAgentTasks)
	api.POST("/agent/tasks/refresh", handlers.RefreshAgentTasks)
	api.PATCH("/agent/tasks/:id", handlers.UpdateAgentTask)
	api.GET("/applications", handlers.ListApplicationPlans)
	api.POST("/applications/sync", handlers.SyncApplicationPlans)
	api.PATCH("/applications/:id", handlers.UpdateApplicationPlan)
	api.GET("/profile", handlers.GetCandidateProfile)
	api.PATCH("/profile", handlers.UpdateCandidateProfile)
	api.GET("/jobs", handlers.ListJobs)
	api.POST("/jobs/cleanup-landing-pages", handlers.CleanupLandingPages)
	api.POST("/jobs/import-url", handlers.ImportURL)
	api.GET("/jobs/:id/detail", handlers.GetJobDetail)
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
	api.GET("/sources/operations", handlers.GetSourceOperations)
	api.POST("/sources/discovery/run", handlers.RunSourceDiscovery)
	api.GET("/sources/candidates", handlers.ListSourceCandidates)
	api.POST("/sources/candidates/:id/validate", handlers.ValidateSourceCandidate)
	api.POST("/sources/candidates/:id/accept", handlers.AcceptSourceCandidate)
	api.POST("/sources/candidates/:id/reject", handlers.RejectSourceCandidate)
	api.POST("/sources/recommended", handlers.SeedRecommendedSources)
	api.POST("/sources", handlers.CreateSource)
	api.PATCH("/sources/:id", handlers.UpdateSource)
	api.POST("/crawl/recommended", handlers.RunRecommendedCrawl)
	api.POST("/notifications/feishu/test", handlers.SendFeishuTest)
	api.POST("/notifications/feishu/report", handlers.SendFeishuReport)

	return router
}
