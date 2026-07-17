package crawl

import (
	"context"
	"net/http"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/importer"
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

type PublicURLCollector struct {
	urls   []string
	client *http.Client
}

func NewPublicURLCollector(urls []string, client *http.Client) PublicURLCollector {
	return PublicURLCollector{urls: urls, client: client}
}

func (PublicURLCollector) Name() string {
	return "public_urls"
}

func (c PublicURLCollector) Collect(ctx context.Context) ([]domain.Job, error) {
	jobs := []domain.Job{}
	for _, sourceURL := range c.urls {
		job, err := importer.ImportURL(ctx, sourceURL, c.client)
		if err != nil {
			jobs = append(jobs, domain.Job{
				Title:          sourceURL,
				SourceName:     "public_urls",
				SourceURL:      sourceURL,
				ApplyURL:       sourceURL,
				Status:         domain.StatusManualCheck,
				Description:    "Import failed: " + err.Error(),
				PenaltyReasons: []string{"Invalid source URL"},
			})
			continue
		}
		if job.SourceName == "" {
			job.SourceName = "public_urls"
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}
