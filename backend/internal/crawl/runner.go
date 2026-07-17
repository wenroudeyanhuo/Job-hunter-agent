package crawl

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

type RunSummary struct {
	SourcesTotal     int          `json:"sources_total"`
	SourcesSuccess   int          `json:"sources_success"`
	SourcesFailed    int          `json:"sources_failed"`
	JobsFound        int          `json:"jobs_found"`
	JobsCreated      int          `json:"jobs_created"`
	JobsDuplicated   int          `json:"jobs_duplicated"`
	ManualCheckCount int          `json:"manual_check_count"`
	ErrorSummary     string       `json:"error_summary"`
	RecommendedJobs  []domain.Job `json:"recommended_jobs,omitempty"`
}

type Runnable interface {
	Run(ctx context.Context, trigger string) (RunSummary, error)
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
			_, _ = r.repo.CreateRunSourceResult(ctx, jobs.RunSourceResultInput{
				JobRunID:     run.ID,
				SourceName:   collector.Name(),
				Status:       "failed",
				ErrorMessage: err.Error(),
			})
			continue
		}
		summary.SourcesSuccess++
		summary.JobsFound += len(collected)
		sourceStats := map[string]*jobs.RunSourceResultInput{}
		for _, rawJob := range collected {
			stat := sourceStat(sourceStats, run.ID, collector.Name(), rawJob.SourceName, rawJob.SourceURL)
			stat.JobsFound++
			scored := jobs.ScoreJob(rawJob)
			if scored.HardFiltered {
				stat.JobsFiltered++
				continue
			}
			if scored.Job.Status == "manual_check" {
				summary.ManualCheckCount++
				stat.ManualCheckCount++
			}
			persisted, duplicated, err := r.repo.UpsertJob(ctx, scored.Job)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s upsert: %v", collector.Name(), err))
				stat.Status = "partial_success"
				stat.ErrorMessage = appendErrorMessage(stat.ErrorMessage, err.Error())
				continue
			}
			if duplicated {
				summary.JobsDuplicated++
				stat.JobsDuplicated++
			} else {
				summary.JobsCreated++
				stat.JobsCreated++
				summary.RecommendedJobs = append(summary.RecommendedJobs, persisted)
			}
		}
		if len(sourceStats) == 0 {
			sourceStats[collector.Name()] = &jobs.RunSourceResultInput{
				JobRunID:   run.ID,
				SourceName: collector.Name(),
				Status:     "success",
			}
		}
		for _, stat := range sourceStats {
			if stat.Status == "" {
				stat.Status = "success"
			}
			if _, err := r.repo.CreateRunSourceResult(ctx, *stat); err != nil {
				errors = append(errors, fmt.Sprintf("%s source result: %v", collector.Name(), err))
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

	sort.SliceStable(summary.RecommendedJobs, func(i, j int) bool {
		return summary.RecommendedJobs[i].MatchScore > summary.RecommendedJobs[j].MatchScore
	})
	return summary, nil
}

func sourceStat(stats map[string]*jobs.RunSourceResultInput, runID int64, collectorName string, sourceName string, sourceURL string) *jobs.RunSourceResultInput {
	if sourceName == "" {
		sourceName = collectorName
	}
	key := sourceName + "\x00" + sourceURL
	stat, ok := stats[key]
	if !ok {
		stat = &jobs.RunSourceResultInput{
			JobRunID:   runID,
			SourceName: sourceName,
			SourceURL:  sourceURL,
			Status:     "success",
		}
		stats[key] = stat
	}
	return stat
}

func appendErrorMessage(existing string, next string) string {
	if strings.TrimSpace(existing) == "" {
		return next
	}
	return existing + "; " + next
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
