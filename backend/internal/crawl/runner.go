package crawl

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

type RunSummary struct {
	SourcesTotal     int    `json:"sources_total"`
	SourcesSuccess   int    `json:"sources_success"`
	SourcesFailed    int    `json:"sources_failed"`
	JobsFound        int    `json:"jobs_found"`
	JobsCreated      int    `json:"jobs_created"`
	JobsDuplicated   int    `json:"jobs_duplicated"`
	ManualCheckCount int    `json:"manual_check_count"`
	ErrorSummary     string `json:"error_summary"`
}

type Runner struct {
	repo       *jobs.Repository
	collectors []Collector
	mu         sync.Mutex
	running    bool
}

func NewRunner(repo *jobs.Repository, collectors []Collector) *Runner {
	return &Runner{repo: repo, collectors: collectors}
}

func (r *Runner) Run(ctx context.Context, trigger string) (RunSummary, error) {
	if !r.tryLock() {
		return RunSummary{}, fmt.Errorf("crawl run already active")
	}
	defer r.unlock()

	run, err := r.repo.CreateRun(ctx, trigger, time.Now().UTC())
	if err != nil {
		return RunSummary{}, err
	}

	summary := RunSummary{SourcesTotal: len(r.collectors)}
	errors := []string{}

	for _, collector := range r.collectors {
		collected, err := collector.Collect(ctx)
		if err != nil {
			summary.SourcesFailed++
			errors = append(errors, fmt.Sprintf("%s: %v", collector.Name(), err))
			continue
		}
		summary.SourcesSuccess++
		summary.JobsFound += len(collected)
		for _, rawJob := range collected {
			scored := jobs.ScoreJob(rawJob)
			if scored.HardFiltered {
				continue
			}
			if scored.Job.Status == "manual_check" {
				summary.ManualCheckCount++
			}
			_, duplicated, err := r.repo.UpsertJob(ctx, scored.Job)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s upsert: %v", collector.Name(), err))
				continue
			}
			if duplicated {
				summary.JobsDuplicated++
			} else {
				summary.JobsCreated++
			}
		}
	}

	summary.ErrorSummary = strings.Join(errors, "; ")
	status := "success"
	if summary.SourcesFailed > 0 || len(errors) > 0 {
		status = "partial_success"
	}
	if err := r.repo.FinishRun(ctx, run.ID, jobs.RunUpdate{
		Status:           status,
		SourcesTotal:     summary.SourcesTotal,
		SourcesSuccess:   summary.SourcesSuccess,
		SourcesFailed:    summary.SourcesFailed,
		JobsFound:        summary.JobsFound,
		JobsCreated:      summary.JobsCreated,
		JobsDuplicated:   summary.JobsDuplicated,
		ManualCheckCount: summary.ManualCheckCount,
		ErrorSummary:     summary.ErrorSummary,
	}); err != nil {
		return summary, err
	}

	return summary, nil
}

func (r *Runner) tryLock() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return false
	}
	r.running = true
	return true
}

func (r *Runner) unlock() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running = false
}
