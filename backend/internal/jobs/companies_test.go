package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositorySeedsCompaniesFromRecommendedSources(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	result, err := repo.SeedRecommendedSources(ctx)
	if err != nil {
		t.Fatalf("seed recommended sources: %v", err)
	}
	companies, err := repo.ListCompanies(ctx)
	if err != nil {
		t.Fatalf("list companies: %v", err)
	}
	if len(companies) != result.Total {
		t.Fatalf("expected one company per recommended source, got %d vs %d", len(companies), result.Total)
	}
	for _, company := range companies {
		if company.SourceCount == 0 {
			t.Fatalf("expected company %q to have a source", company.Name)
		}
	}
}

func TestRepositoryUpdateCompanyEnabledTogglesSources(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	source, err := repo.CreateSource(ctx, SourceInput{
		Name:     "Tencent Careers",
		URL:      "https://careers.tencent.com/",
		Enabled:  true,
		Category: "internet",
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if source.CompanyID == 0 {
		t.Fatalf("expected source to be linked to company: %#v", source)
	}
	if err := repo.UpdateCompanyEnabled(ctx, source.CompanyID, false); err != nil {
		t.Fatalf("disable company: %v", err)
	}
	updated, err := repo.GetSource(ctx, source.ID)
	if err != nil {
		t.Fatalf("get source: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected source to be disabled with company: %#v", updated)
	}
}

func TestRepositorySeedRecommendedSourcesPreservesCompanyEnabledChoice(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	source, err := repo.CreateSource(ctx, SourceInput{
		Name:       "OPPO Careers",
		URL:        "https://careers.oppo.com/",
		Enabled:    true,
		Category:   "hardware",
		ParserType: "generic",
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := repo.UpdateCompanyEnabled(ctx, source.CompanyID, false); err != nil {
		t.Fatalf("disable company: %v", err)
	}
	if _, err := repo.SeedRecommendedSources(ctx); err != nil {
		t.Fatalf("seed recommended: %v", err)
	}
	company, err := repo.GetCompany(ctx, source.CompanyID)
	if err != nil {
		t.Fatalf("get company: %v", err)
	}
	if company.Enabled {
		t.Fatalf("expected seed to preserve disabled company: %#v", company)
	}
}
