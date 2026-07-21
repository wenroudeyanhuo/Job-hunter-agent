package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const candidateProfileID int64 = 1

type CandidateProfile struct {
	ID                   int64     `json:"id"`
	TargetCities         []string  `json:"target_cities"`
	TargetDirections     []string  `json:"target_directions"`
	Skills               []string  `json:"skills"`
	Education            string    `json:"education"`
	GraduationYear       string    `json:"graduation_year"`
	InternshipPreference string    `json:"internship_preference"`
	PreferredCompanies   []string  `json:"preferred_companies"`
	BlockedKeywords      []string  `json:"blocked_keywords"`
	Notes                string    `json:"notes"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func DefaultCandidateProfile() CandidateProfile {
	return CandidateProfile{
		ID:                   candidateProfileID,
		TargetCities:         []string{"Shenzhen"},
		TargetDirections:     []string{"frontend", "backend", "java", "go", "algorithm", "ai_application"},
		Skills:               []string{"Go", "Java", "React", "TypeScript", "Algorithm", "LLM"},
		InternshipPreference: "accept_conversion_clear",
		BlockedKeywords:      []string{"outsourcing", "training", "bootcamp", "外包", "培训"},
		UpdatedAt:            time.Now().UTC(),
	}
}

func (r *Repository) GetCandidateProfile(ctx context.Context) (CandidateProfile, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, target_cities, target_directions, skills, education, graduation_year,
			internship_preference, preferred_companies, blocked_keywords, notes, updated_at
		FROM candidate_profiles
		WHERE id = ?
	`, candidateProfileID)
	profile, err := scanCandidateProfile(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return DefaultCandidateProfile(), nil
		}
		return CandidateProfile{}, fmt.Errorf("get candidate profile: %w", err)
	}
	return normalizeCandidateProfile(profile), nil
}

func (r *Repository) SaveCandidateProfile(ctx context.Context, profile CandidateProfile) (CandidateProfile, error) {
	profile = normalizeCandidateProfile(profile)
	now := time.Now().UTC()
	targetCities, err := marshalStrings(profile.TargetCities)
	if err != nil {
		return CandidateProfile{}, err
	}
	targetDirections, err := marshalStrings(profile.TargetDirections)
	if err != nil {
		return CandidateProfile{}, err
	}
	skills, err := marshalStrings(profile.Skills)
	if err != nil {
		return CandidateProfile{}, err
	}
	preferredCompanies, err := marshalStrings(profile.PreferredCompanies)
	if err != nil {
		return CandidateProfile{}, err
	}
	blockedKeywords, err := marshalStrings(profile.BlockedKeywords)
	if err != nil {
		return CandidateProfile{}, err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO candidate_profiles (
			id, target_cities, target_directions, skills, education, graduation_year,
			internship_preference, preferred_companies, blocked_keywords, notes, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			target_cities = excluded.target_cities,
			target_directions = excluded.target_directions,
			skills = excluded.skills,
			education = excluded.education,
			graduation_year = excluded.graduation_year,
			internship_preference = excluded.internship_preference,
			preferred_companies = excluded.preferred_companies,
			blocked_keywords = excluded.blocked_keywords,
			notes = excluded.notes,
			updated_at = excluded.updated_at
	`, candidateProfileID, targetCities, targetDirections, skills, profile.Education,
		profile.GraduationYear, profile.InternshipPreference, preferredCompanies, blockedKeywords,
		profile.Notes, now)
	if err != nil {
		return CandidateProfile{}, fmt.Errorf("save candidate profile: %w", err)
	}
	return r.GetCandidateProfile(ctx)
}

func normalizeCandidateProfile(profile CandidateProfile) CandidateProfile {
	defaults := DefaultCandidateProfile()
	profile.ID = candidateProfileID
	profile.TargetCities = cleanStringList(profile.TargetCities)
	if len(profile.TargetCities) == 0 {
		profile.TargetCities = defaults.TargetCities
	}
	profile.TargetDirections = cleanStringList(profile.TargetDirections)
	if len(profile.TargetDirections) == 0 {
		profile.TargetDirections = defaults.TargetDirections
	}
	profile.Skills = cleanStringList(profile.Skills)
	profile.PreferredCompanies = cleanStringList(profile.PreferredCompanies)
	profile.BlockedKeywords = cleanStringList(profile.BlockedKeywords)
	if len(profile.BlockedKeywords) == 0 {
		profile.BlockedKeywords = defaults.BlockedKeywords
	}
	profile.Education = strings.TrimSpace(profile.Education)
	profile.GraduationYear = strings.TrimSpace(profile.GraduationYear)
	profile.InternshipPreference = strings.TrimSpace(profile.InternshipPreference)
	if profile.InternshipPreference == "" {
		profile.InternshipPreference = defaults.InternshipPreference
	}
	profile.Notes = strings.TrimSpace(profile.Notes)
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = time.Now().UTC()
	}
	return profile
}

func scanCandidateProfile(scanner jobScanner) (CandidateProfile, error) {
	var profile CandidateProfile
	var targetCities string
	var targetDirections string
	var skills string
	var preferredCompanies string
	var blockedKeywords string
	if err := scanner.Scan(
		&profile.ID,
		&targetCities,
		&targetDirections,
		&skills,
		&profile.Education,
		&profile.GraduationYear,
		&profile.InternshipPreference,
		&preferredCompanies,
		&blockedKeywords,
		&profile.Notes,
		&profile.UpdatedAt,
	); err != nil {
		return CandidateProfile{}, err
	}
	profile.TargetCities = unmarshalStrings(targetCities)
	profile.TargetDirections = unmarshalStrings(targetDirections)
	profile.Skills = unmarshalStrings(skills)
	profile.PreferredCompanies = unmarshalStrings(preferredCompanies)
	profile.BlockedKeywords = unmarshalStrings(blockedKeywords)
	return profile, nil
}
