package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	SourceCandidateStatusPending  = "pending"
	SourceCandidateStatusAccepted = "accepted"
	SourceCandidateStatusRejected = "rejected"
)

type SourceCandidate struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	URL              string     `json:"url"`
	Category         string     `json:"category"`
	ParserType       string     `json:"parser_type"`
	DiscoveredBy     string     `json:"discovered_by"`
	Reason           string     `json:"reason"`
	Confidence       int        `json:"confidence"`
	Status           string     `json:"status"`
	ValidationStatus string     `json:"validation_status"`
	ValidationReason string     `json:"validation_reason"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	SourceID         int64      `json:"source_id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type SourceDiscoveryInput struct {
	TargetCities     []string `json:"target_cities"`
	TargetDirections []string `json:"target_directions"`
}

type SourceCandidateFilter struct {
	Status string
}

type SourceDiscoveryResult struct {
	Total      int `json:"total"`
	Created    int `json:"created"`
	Duplicated int `json:"duplicated"`
}

type sourceCandidateInput struct {
	Name       string
	URL        string
	Category   string
	ParserType string
	Reason     string
	Confidence int
}

func (r *Repository) DiscoverSourceCandidates(ctx context.Context, input SourceDiscoveryInput) (SourceDiscoveryResult, error) {
	candidates := BuildSourceDiscoveryCandidates(input)
	result := SourceDiscoveryResult{Total: len(candidates)}
	for _, candidate := range candidates {
		created, err := r.createSourceCandidateIfMissing(ctx, candidate)
		if err != nil {
			return SourceDiscoveryResult{}, err
		}
		if created {
			result.Created++
		} else {
			result.Duplicated++
		}
	}
	return result, nil
}

func BuildSourceDiscoveryCandidates(input SourceDiscoveryInput) []sourceCandidateInput {
	cities := cleanStringList(input.TargetCities)
	if len(cities) == 0 {
		cities = []string{"Shenzhen"}
	}
	directions := cleanStringList(input.TargetDirections)
	if len(directions) == 0 {
		directions = []string{"go", "backend", "ai_application"}
	}

	out := []sourceCandidateInput{}
	for _, source := range expandableRecommendedSources() {
		out = append(out, sourceCandidateInput{
			Name:       source.Name + " discovery",
			URL:        source.URL,
			Category:   source.Category,
			ParserType: source.ParserType,
			Reason:     "Official career source adjacent to the configured company pool.",
			Confidence: 72,
		})
	}
	for _, city := range cities {
		for _, direction := range directions {
			query := city + " " + directionLabel(direction) + " 校招 实习 招聘"
			out = append(out,
				sourceCandidateInput{
					Name:       "Nowcoder search - " + city + " " + directionLabel(direction),
					URL:        "https://www.nowcoder.com/search?query=" + url.QueryEscape(query),
					Category:   "community",
					ParserType: "generic",
					Reason:     "Community search can surface fresh campus openings beyond fixed official sources.",
					Confidence: 62,
				},
				sourceCandidateInput{
					Name:       "Boss search - " + city + " " + directionLabel(direction),
					URL:        "https://www.zhipin.com/web/geek/job?query=" + url.QueryEscape(directionLabel(direction)) + "&city=101280600",
					Category:   "job_platform",
					ParserType: "generic",
					Reason:     "Job-platform query candidate derived from current city and direction preferences.",
					Confidence: 55,
				},
				sourceCandidateInput{
					Name:       "Lagou search - " + city + " " + directionLabel(direction),
					URL:        "https://www.lagou.com/wn/jobs?kd=" + url.QueryEscape(directionLabel(direction)),
					Category:   "job_platform",
					ParserType: "generic",
					Reason:     "Platform search candidate for broadening non-official source coverage.",
					Confidence: 52,
				},
			)
		}
	}
	return dedupeSourceCandidateInputs(out)
}

func (r *Repository) ListSourceCandidates(ctx context.Context, filter SourceCandidateFilter) ([]SourceCandidate, error) {
	query := selectSourceCandidateSQL()
	args := []any{}
	if strings.TrimSpace(filter.Status) != "" {
		query += " WHERE status = ?"
		args = append(args, normalizeSourceCandidateStatus(filter.Status))
	}
	query += " ORDER BY status ASC, confidence DESC, updated_at DESC, id DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list source candidates: %w", err)
	}
	defer rows.Close()

	out := []SourceCandidate{}
	for rows.Next() {
		candidate, err := scanSourceCandidate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source candidates: %w", err)
	}
	return out, nil
}

func (r *Repository) GetSourceCandidate(ctx context.Context, id int64) (SourceCandidate, error) {
	row := r.db.QueryRowContext(ctx, selectSourceCandidateSQL()+` WHERE id = ?`, id)
	candidate, err := scanSourceCandidate(row)
	if err != nil {
		return SourceCandidate{}, fmt.Errorf("get source candidate %d: %w", id, err)
	}
	return candidate, nil
}

func (r *Repository) AcceptSourceCandidate(ctx context.Context, id int64) (SourceCandidate, Source, error) {
	candidate, err := r.GetSourceCandidate(ctx, id)
	if err != nil {
		return SourceCandidate{}, Source{}, err
	}
	source, err := r.CreateSource(ctx, SourceInput{
		Name:       strings.TrimSuffix(candidate.Name, " discovery"),
		Type:       "public_url",
		URL:        candidate.URL,
		Enabled:    true,
		Category:   candidate.Category,
		ParserType: candidate.ParserType,
	})
	if err != nil {
		return SourceCandidate{}, Source{}, err
	}
	updated, err := r.setSourceCandidateStatus(ctx, id, SourceCandidateStatusAccepted, source.ID)
	if err != nil {
		return SourceCandidate{}, Source{}, err
	}
	return updated, source, nil
}

func (r *Repository) UpdateSourceCandidateStatus(ctx context.Context, id int64, status string) (SourceCandidate, error) {
	return r.setSourceCandidateStatus(ctx, id, status, 0)
}

func (r *Repository) createSourceCandidateIfMissing(ctx context.Context, input sourceCandidateInput) (bool, error) {
	input = normalizeSourceCandidateInput(input)
	if input.URL == "" {
		return false, nil
	}
	var existingID int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM source_candidates WHERE url = ?`, input.URL).Scan(&existingID)
	if err == nil {
		_, err = r.db.ExecContext(ctx, `
			UPDATE source_candidates
			SET name = ?, category = ?, parser_type = ?, reason = ?, confidence = ?,
				validation_status = ?, validation_reason = ?, last_checked_at = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND status = ?
		`, input.Name, input.Category, input.ParserType, input.Reason, input.Confidence,
			validateCandidateURL(input.URL), validationReason(input.URL), time.Now().UTC(), existingID, SourceCandidateStatusPending)
		if err != nil {
			return false, fmt.Errorf("refresh source candidate: %w", err)
		}
		return false, nil
	}
	if err != sql.ErrNoRows {
		return false, fmt.Errorf("find source candidate: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO source_candidates (
			name, url, category, parser_type, discovered_by, reason, confidence, status,
			validation_status, validation_reason, last_checked_at
		) VALUES (?, ?, ?, ?, 'agent', ?, ?, ?, ?, ?, ?)
	`, input.Name, input.URL, input.Category, input.ParserType, input.Reason, input.Confidence,
		SourceCandidateStatusPending, validateCandidateURL(input.URL), validationReason(input.URL), time.Now().UTC())
	if err != nil {
		return false, fmt.Errorf("insert source candidate: %w", err)
	}
	return true, nil
}

func (r *Repository) setSourceCandidateStatus(ctx context.Context, id int64, status string, sourceID int64) (SourceCandidate, error) {
	status = normalizeSourceCandidateStatus(status)
	_, err := r.db.ExecContext(ctx, `
		UPDATE source_candidates
		SET status = ?, source_id = CASE WHEN ? > 0 THEN ? ELSE source_id END, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, sourceID, sourceID, id)
	if err != nil {
		return SourceCandidate{}, fmt.Errorf("update source candidate status: %w", err)
	}
	return r.GetSourceCandidate(ctx, id)
}

func normalizeSourceCandidateInput(input sourceCandidateInput) sourceCandidateInput {
	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	input.Category = strings.TrimSpace(input.Category)
	input.ParserType = strings.TrimSpace(input.ParserType)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Category == "" {
		input.Category = "discovery"
	}
	if input.ParserType == "" {
		input.ParserType = "generic"
	}
	if input.Confidence <= 0 {
		input.Confidence = 50
	}
	parsed, err := url.ParseRequestURI(input.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		input.URL = ""
		return input
	}
	input.URL = parsed.String()
	if input.Name == "" {
		input.Name = sourceNameFromURL(parsed)
	}
	return input
}

func normalizeSourceCandidateStatus(status string) string {
	switch strings.TrimSpace(status) {
	case SourceCandidateStatusAccepted:
		return SourceCandidateStatusAccepted
	case SourceCandidateStatusRejected:
		return SourceCandidateStatusRejected
	default:
		return SourceCandidateStatusPending
	}
}

func validateCandidateURL(rawURL string) string {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" {
		return "invalid"
	}
	return "reachable_candidate"
}

func validationReason(rawURL string) string {
	if validateCandidateURL(rawURL) == "invalid" {
		return "Candidate URL is invalid."
	}
	return "URL shape is valid. Accept it to include in the next crawl."
}

func expandableRecommendedSources() []SourceInput {
	sources := append([]SourceInput(nil), RecommendedSources()...)
	sort.SliceStable(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})
	limit := 10
	if len(sources) < limit {
		limit = len(sources)
	}
	return sources[:limit]
}

func dedupeSourceCandidateInputs(values []sourceCandidateInput) []sourceCandidateInput {
	seen := map[string]bool{}
	out := []sourceCandidateInput{}
	for _, value := range values {
		value = normalizeSourceCandidateInput(value)
		if value.URL == "" || seen[value.URL] {
			continue
		}
		seen[value.URL] = true
		out = append(out, value)
	}
	return out
}

func directionLabel(direction string) string {
	switch strings.TrimSpace(direction) {
	case "ai_application":
		return "AI 应用开发"
	case "backend":
		return "后端"
	case "frontend":
		return "前端"
	default:
		return strings.TrimSpace(direction)
	}
}

func selectSourceCandidateSQL() string {
	return `
		SELECT id, name, url, category, parser_type, discovered_by, reason, confidence,
			status, validation_status, validation_reason, last_checked_at, source_id,
			created_at, updated_at
		FROM source_candidates`
}

func scanSourceCandidate(scanner interface {
	Scan(dest ...any) error
}) (SourceCandidate, error) {
	var candidate SourceCandidate
	if err := scanner.Scan(
		&candidate.ID,
		&candidate.Name,
		&candidate.URL,
		&candidate.Category,
		&candidate.ParserType,
		&candidate.DiscoveredBy,
		&candidate.Reason,
		&candidate.Confidence,
		&candidate.Status,
		&candidate.ValidationStatus,
		&candidate.ValidationReason,
		&candidate.LastCheckedAt,
		&candidate.SourceID,
		&candidate.CreatedAt,
		&candidate.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return SourceCandidate{}, err
		}
		return SourceCandidate{}, fmt.Errorf("scan source candidate: %w", err)
	}
	candidate.Status = normalizeSourceCandidateStatus(candidate.Status)
	return candidate, nil
}
