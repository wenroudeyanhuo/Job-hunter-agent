package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryGetsDefaultCandidateProfile(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	profile, err := repo.GetCandidateProfile(ctx)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if len(profile.TargetCities) == 0 || profile.TargetCities[0] != "Shenzhen" {
		t.Fatalf("expected Shenzhen default city, got %#v", profile.TargetCities)
	}
	if len(profile.TargetDirections) == 0 {
		t.Fatalf("expected default directions, got %#v", profile.TargetDirections)
	}
	if profile.UpdatedAt.IsZero() {
		t.Fatal("expected default profile timestamp")
	}
}

func TestRepositorySavesCandidateProfile(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	saved, err := repo.SaveCandidateProfile(ctx, CandidateProfile{
		TargetCities:         []string{" Shenzhen ", "Guangzhou", "Shenzhen"},
		TargetDirections:     []string{"go", "backend", "go"},
		Skills:               []string{"Go", "React", "LLM"},
		Education:            "本科",
		GraduationYear:       "2027",
		InternshipPreference: "accept_conversion_clear",
		PreferredCompanies:   []string{"Tencent", "ByteDance"},
		BlockedKeywords:      []string{"外包", "培训"},
		Notes:                "Prefer product engineering roles.",
	})
	if err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if len(saved.TargetCities) != 2 || saved.TargetCities[0] != "Shenzhen" || saved.TargetCities[1] != "Guangzhou" {
		t.Fatalf("target cities were not normalized: %#v", saved.TargetCities)
	}
	if saved.Education != "本科" || saved.GraduationYear != "2027" {
		t.Fatalf("profile fields did not round trip: %#v", saved)
	}

	loaded, err := repo.GetCandidateProfile(ctx)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if loaded.Notes != "Prefer product engineering roles." {
		t.Fatalf("expected saved notes, got %q", loaded.Notes)
	}
	if len(loaded.PreferredCompanies) != 2 {
		t.Fatalf("preferred companies did not round trip: %#v", loaded.PreferredCompanies)
	}
}
