package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/importer"
)

const (
	SourceCandidateStatusPending  = "pending"
	SourceCandidateStatusAccepted = "accepted"
	SourceCandidateStatusRejected = "rejected"

	SourceCandidateValidationUnchecked    = "unchecked"
	SourceCandidateValidationURLCandidate = "reachable_candidate"
	SourceCandidateValidationGood         = "verified_good"
	SourceCandidateValidationWeak         = "weak_signal"
	SourceCandidateValidationUnreachable  = "unreachable"
	SourceCandidateValidationInvalid      = "invalid"
)

const maxSourceCandidateValidationBytes = 512 << 10
const sourceCandidateUserAgent = "JobHunterAgent/0.1 (+https://github.com/wenroudeyanhuo/Job-hunter-agent)"

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
			query := city + " " + directionLabel(direction) + " \u6821\u62db \u5b9e\u4e60 \u62db\u8058"
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

func (r *Repository) ValidateSourceCandidate(ctx context.Context, id int64, client *http.Client) (SourceCandidate, error) {
	candidate, err := r.GetSourceCandidate(ctx, id)
	if err != nil {
		return SourceCandidate{}, err
	}
	status, reason, confidenceDelta := validateSourceCandidatePage(ctx, candidate.URL, client)
	confidence := clampConfidence(candidate.Confidence + confidenceDelta)
	checkedAt := time.Now().UTC()
	_, err = r.db.ExecContext(ctx, `
		UPDATE source_candidates
		SET validation_status = ?, validation_reason = ?, confidence = ?, last_checked_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, reason, confidence, checkedAt, id)
	if err != nil {
		return SourceCandidate{}, fmt.Errorf("validate source candidate: %w", err)
	}
	return r.GetSourceCandidate(ctx, id)
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
		return SourceCandidateValidationInvalid
	}
	return SourceCandidateValidationURLCandidate
}

func validationReason(rawURL string) string {
	if validateCandidateURL(rawURL) == SourceCandidateValidationInvalid {
		return "Candidate URL is invalid."
	}
	return "URL shape is valid. Accept it to include in the next crawl."
}

func validateSourceCandidatePage(ctx context.Context, rawURL string, client *http.Client) (string, string, int) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return SourceCandidateValidationInvalid, "Candidate URL is invalid.", -40
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return SourceCandidateValidationInvalid, "Could not build validation request.", -40
	}
	req.Header.Set("User-Agent", sourceCandidateUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return SourceCandidateValidationUnreachable, "Fetch failed: " + err.Error(), -25
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SourceCandidateValidationUnreachable, fmt.Sprintf("Fetch returned HTTP %d.", resp.StatusCode), -20
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSourceCandidateValidationBytes))
	if err != nil {
		return SourceCandidateValidationWeak, "Could not read response body.", -5
	}
	text := strings.ToLower(string(body))
	signalCount := countContainsAny(text,
		"career", "careers", "campus", "graduate", "intern", "recruit", "recruitment",
		"job description", "requirements", "apply", "position", "frontend", "backend", "golang", "algorithm", "llm",
		"\u62db\u8058", "\u6821\u62db", "\u79cb\u62db", "\u5b9e\u4e60", "\u5c97\u4f4d", "\u804c\u4f4d", "\u6295\u9012", "\u6df1\u5733",
	)
	links, _ := importer.DiscoverLinks(ctx, parsed.String(), client, 8)
	jobCards, _ := importer.DiscoverJobCards(ctx, parsed.String(), client, 6)
	if len(jobCards) > 0 {
		return SourceCandidateValidationGood, fmt.Sprintf("Verified %d recruitment signals, %d candidate links, and %d structured job cards.", signalCount, len(links), len(jobCards)), 20
	}
	if signalCount >= 3 || len(links) >= 2 {
		return SourceCandidateValidationGood, fmt.Sprintf("Verified %d recruitment signals and %d candidate links.", signalCount, len(links)), 18
	}
	if signalCount > 0 || len(links) > 0 {
		return SourceCandidateValidationWeak, fmt.Sprintf("Found %d recruitment signals and %d candidate links; manual review recommended.", signalCount, len(links)), 6
	}
	return SourceCandidateValidationWeak, "Fetched successfully, but found no clear recruitment signals.", -10
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
		return "AI application"
	case "backend":
		return "backend"
	case "frontend":
		return "frontend"
	default:
		return strings.TrimSpace(direction)
	}
}
func countContainsAny(value string, needles ...string) int {
	count := 0
	value = strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(value, strings.ToLower(needle)) {
			count++
		}
	}
	return count
}

func clampConfidence(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
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
