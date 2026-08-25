package app

import (
	"context"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func buildMultiAgentCycleInput(ctx context.Context, repo *jobs.Repository, now time.Time) (jobs.MultiAgentCycleInput, error) {
	jobList, err := repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		return jobs.MultiAgentCycleInput{}, err
	}
	sources, err := repo.ListSources(ctx, false)
	if err != nil {
		return jobs.MultiAgentCycleInput{}, err
	}
	settings, err := repo.GetSettings(ctx)
	if err != nil {
		return jobs.MultiAgentCycleInput{}, err
	}
	taskDate := todayFromSettings(settings, now)
	tasks, err := repo.ListAgentTasks(ctx, taskDate)
	if err != nil {
		return jobs.MultiAgentCycleInput{}, err
	}
	plans, err := repo.ListAgentPlans(ctx, "", 20)
	if err != nil {
		return jobs.MultiAgentCycleInput{}, err
	}
	memory, err := repo.GetSemanticMemoryStats(ctx)
	if err != nil {
		return jobs.MultiAgentCycleInput{}, err
	}
	return jobs.MultiAgentCycleInput{
		Jobs:    jobList,
		Sources: sources,
		Tasks:   tasks,
		Plans:   plans,
		Memory:  memory,
		Now:     now.UTC(),
	}, nil
}

func runAndRecordAgentCycle(ctx context.Context, repo *jobs.Repository, source string, now time.Time, orchestrator jobs.RecruitingOrchestrator) (jobs.MultiAgentCycleResult, error) {
	input, err := buildMultiAgentCycleInput(ctx, repo, now)
	if err != nil {
		return jobs.MultiAgentCycleResult{}, err
	}
	result, err := jobs.NewAgentRuntime(repo).RunMultiAgentCycle(ctx, jobs.MultiAgentCycleRequest{
		Input:                input,
		Source:               source,
		RecordActionRequests: true,
		Orchestrator:         orchestrator,
	})
	if err != nil {
		return jobs.MultiAgentCycleResult{}, err
	}
	_, _ = repo.CreateAgentEvent(ctx, jobs.AgentEventInput{
		Type:    "multi_agent_cycle_completed",
		Title:   "Completed multi-agent cycle",
		Summary: result.Cycle.Summary,
		Level:   "success",
	})
	return result, nil
}

func latestAgentCycle(ctx context.Context, repo *jobs.Repository) *jobs.AgentCycleRecord {
	cycles, err := repo.ListAgentCycles(ctx, 1)
	if err != nil || len(cycles) == 0 {
		return nil
	}
	return &cycles[0]
}

func todayFromSettings(settings jobs.Settings, now time.Time) string {
	loc, err := time.LoadLocation(settings.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	return now.In(loc).Format("2006-01-02")
}
