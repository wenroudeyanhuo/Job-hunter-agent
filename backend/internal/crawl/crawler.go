package crawl

import (
	"context"
	"net/http"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/importer"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

const discoveredLinksPerSource = 10

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
	seen := map[string]struct{}{}
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

		links, err := importer.DiscoverLinks(ctx, sourceURL, c.client, discoveredLinksPerSource)
		discoveredConcreteJobs := 0
		if err == nil {
			for _, link := range links {
				discoveredJob, err := importer.ImportURL(ctx, link, c.client)
				if err != nil {
					continue
				}
				if discoveredJob.SourceName == "" {
					discoveredJob.SourceName = job.SourceName
				}
				if !importer.LooksLikeConcreteJobPosting(discoveredJob) && discoveredJob.Status != domain.StatusManualCheck {
					continue
				}
				discoveredConcreteJobs++
				jobs = appendUniqueJob(jobs, seen, discoveredJob)
			}
		}

		if importer.LooksLikeConcreteJobPosting(job) || (job.Status == domain.StatusManualCheck && discoveredConcreteJobs == 0) {
			jobs = appendUniqueJob(jobs, seen, job)
		}
	}
	return jobs, nil
}

func appendUniqueJob(jobs []domain.Job, seen map[string]struct{}, job domain.Job) []domain.Job {
	key := job.ApplyURL
	if key == "" {
		key = job.SourceURL
	}
	if key != "" {
		if _, ok := seen[key]; ok {
			return jobs
		}
		seen[key] = struct{}{}
	}
	return append(jobs, job)
}

type SourceLister interface {
	ListSources(ctx context.Context, enabledOnly bool) ([]jobs.Source, error)
}

type DBSourceCollector struct {
	repo   SourceLister
	client *http.Client
}

func NewDBSourceCollector(repo SourceLister, client *http.Client) DBSourceCollector {
	return DBSourceCollector{repo: repo, client: client}
}

func (DBSourceCollector) Name() string {
	return "db_sources"
}

func (c DBSourceCollector) Collect(ctx context.Context) ([]domain.Job, error) {
	sources, err := c.repo.ListSources(ctx, true)
	if err != nil {
		return nil, err
	}
	urls := []string{}
	for _, source := range sources {
		if source.Type != "public_url" || source.URL == "" {
			continue
		}
		urls = append(urls, source.URL)
	}
	return NewPublicURLCollector(urls, c.client).Collect(ctx)
}
