package crawl

import (
	"context"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type Collector interface {
	Name() string
	Collect(ctx context.Context) ([]domain.Job, error)
}

type SeedCollector struct{}

func (SeedCollector) Name() string {
	return "seed"
}

func (SeedCollector) Collect(context.Context) ([]domain.Job, error) {
	return []domain.Job{}, nil
}
