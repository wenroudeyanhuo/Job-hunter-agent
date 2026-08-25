package jobs

import (
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestBuildAgentToolObserverCycleRecordsExecutionResult(t *testing.T) {
	cycle := BuildAgentToolObserverCycle(AgentActionRequest{
		ActionType: "run_crawl",
		Target:     "sources",
	}, AgentActionExecutionSucceeded, "Created 3 jobs.", time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC))

	if len(cycle.Trace) != 2 {
		t.Fatalf("expected observer and planner trace, got %#v", cycle.Trace)
	}
	if cycle.Trace[0].AgentKey != MultiAgentObserver || cycle.Trace[1].AgentKey != MultiAgentPlanner {
		t.Fatalf("expected observer -> planner trace, got %#v", cycle.Trace)
	}
	if cycle.Team.Orchestrator.Provider != "tool_observer" {
		t.Fatalf("expected tool observer provider, got %#v", cycle.Team.Orchestrator)
	}
	if cycle.ReadinessScore <= 0 || cycle.Summary == "" {
		t.Fatalf("expected complete observer cycle, got %#v", cycle)
	}
}

func TestSourceOperationsSummaryIncludesQualityScore(t *testing.T) {
	repo := newProductGapTestRepository(t)
	ctx := t.Context()
	healthy, err := repo.CreateSource(ctx, SourceInput{Name: "Healthy", Type: "public_url", URL: "https://example.com/jobs", Enabled: true})
	if err != nil {
		t.Fatalf("create healthy source: %v", err)
	}
	broken, err := repo.CreateSource(ctx, SourceInput{Name: "Broken", Type: "public_url", URL: "https://broken.example.com/jobs", Enabled: true})
	if err != nil {
		t.Fatalf("create broken source: %v", err)
	}
	if err := repo.UpdateSourceHealthByURL(ctx, healthy.URL, SourceHealthInput{Success: true, Reason: "Collected 5 jobs", FoundCount: 5}); err != nil {
		t.Fatalf("mark healthy: %v", err)
	}
	if err := repo.UpdateSourceHealthByURL(ctx, broken.URL, SourceHealthInput{Success: false, Reason: "HTTP 500", FoundCount: 0}); err != nil {
		t.Fatalf("mark broken: %v", err)
	}

	summary, err := repo.BuildSourceOperationsSummary(ctx)
	if err != nil {
		t.Fatalf("build summary: %v", err)
	}
	if summary.SourceQualityScore <= 0 || summary.SourceQualityScore >= 100 {
		t.Fatalf("expected mixed source quality score, got %#v", summary)
	}
}

func TestParserGapManualJobsBecomeTasks(t *testing.T) {
	repo := newProductGapTestRepository(t)
	ctx := t.Context()
	if _, err := repo.CreateJob(ctx, domain.Job{
		Company:        "Unknown",
		Title:          "Recruiting landing page",
		Status:         domain.StatusManualCheck,
		PenaltyReasons: []string{"Parser gap: no concrete job links found"},
		SourceURL:      "https://example.com/campus",
		ApplyURL:       "https://example.com/campus",
	}); err != nil {
		t.Fatalf("save job: %v", err)
	}

	tasks, err := repo.SyncAgentTasks(ctx, time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("sync tasks: %v", err)
	}
	if !hasAgentTaskKind(tasks, AgentTaskKindFixParserGap) {
		t.Fatalf("expected parser gap task, got %#v", tasks)
	}
}

func TestSemanticMemoryStoresProfileAndDecisionKinds(t *testing.T) {
	repo := newProductGapTestRepository(t)
	ctx := t.Context()
	if _, err := repo.SaveCandidateProfile(ctx, CandidateProfile{
		TargetCities:       []string{"Shenzhen"},
		TargetDirections:   []string{"go", "ai_application"},
		Skills:             []string{"Go", "LLM"},
		PreferredCompanies: []string{"Tencent"},
		BlockedKeywords:    []string{"外包"},
		Notes:              "Prefer platform engineering roles.",
	}); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	job, err := repo.CreateJob(ctx, domain.Job{Company: "Tencent", Title: "Go Backend Engineer", Status: domain.StatusNew})
	if err != nil {
		t.Fatalf("save job: %v", err)
	}
	if _, err := repo.RecordJobDecision(ctx, JobDecisionInput{
		JobID:    job.ID,
		Action:   "interested",
		Reason:   "Good Go backend match.",
		ToStatus: string(domain.StatusInterested),
	}); err != nil {
		t.Fatalf("record decision: %v", err)
	}

	profileMatches, err := repo.SearchSemanticMemory(ctx, SemanticMemoryQuery{Query: "Go LLM Tencent preference", Kind: SemanticMemoryKindProfile, Limit: 3})
	if err != nil {
		t.Fatalf("search profile memory: %v", err)
	}
	decisionMatches, err := repo.SearchSemanticMemory(ctx, SemanticMemoryQuery{Query: "why interested Go backend", Kind: SemanticMemoryKindDecision, Limit: 3})
	if err != nil {
		t.Fatalf("search decision memory: %v", err)
	}
	if len(profileMatches) == 0 || len(decisionMatches) == 0 {
		t.Fatalf("expected profile and decision memories, got profile=%#v decision=%#v", profileMatches, decisionMatches)
	}
}

func newProductGapTestRepository(t *testing.T) *Repository {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return NewRepository(conn)
}

func hasAgentTaskKind(tasks []AgentTask, kind string) bool {
	for _, task := range tasks {
		if task.Kind == kind {
			return true
		}
	}
	return false
}
