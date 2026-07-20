package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	SourceHealthUnknown = "unknown"
	SourceHealthHealthy = "healthy"
	SourceHealthWarning = "warning"
	SourceHealthBroken  = "broken"
)

type Source struct {
	ID                  int64      `json:"id"`
	Name                string     `json:"name"`
	Type                string     `json:"type"`
	URL                 string     `json:"url"`
	Enabled             bool       `json:"enabled"`
	ParserType          string     `json:"parser_type"`
	LastRunAt           *time.Time `json:"last_run_at,omitempty"`
	HealthStatus        string     `json:"health_status"`
	HealthReason        string     `json:"health_reason"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`
	LastFailureAt       *time.Time `json:"last_failure_at,omitempty"`
	LastFoundCount      int        `json:"last_found_count"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type SourceInput struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	Enabled    bool   `json:"enabled"`
	ParserType string `json:"parser_type"`
}

type SeedSourcesResult struct {
	Total      int `json:"total"`
	Created    int `json:"created"`
	Duplicated int `json:"duplicated"`
}

type SourceHealthInput struct {
	Status     string
	Reason     string
	FoundCount int
	Success    bool
}

func RecommendedSources() []SourceInput {
	return []SourceInput{
		{Name: "Tencent Careers", URL: "https://careers.tencent.com/", Enabled: true, ParserType: "tencent_api"},
		{Name: "Huawei Careers", URL: "https://career.huawei.com/reccampportal/portal5/index.html", Enabled: true},
		{Name: "ByteDance Jobs", URL: "https://jobs.bytedance.com/campus/", Enabled: true, ParserType: "bytedance_api"},
		{Name: "Alibaba Campus", URL: "https://talent.alibaba.com/campus/home", Enabled: true},
		{Name: "Meituan Campus", URL: "https://campus.meituan.com/", Enabled: true, ParserType: "meituan_api"},
		{Name: "DJI Careers", URL: "https://we.dji.com/zh-CN/campus", Enabled: true},
		{Name: "Kuaishou Campus", URL: "https://campus.kuaishou.cn/", Enabled: true},
		{Name: "Baidu Talent", URL: "https://talent.baidu.com/jobs/list", Enabled: true},
		{Name: "OPPO Careers", URL: "https://careers.oppo.com/", Enabled: true, ParserType: "oppo_api"},
		{Name: "vivo Careers", URL: "https://hr.vivo.com/", Enabled: true},
		{Name: "Honor Careers", URL: "https://career.hihonor.com/", Enabled: true},
	}
}

func (r *Repository) CreateSource(ctx context.Context, input SourceInput) (Source, error) {
	input, err := normalizeSourceInput(input)
	if err != nil {
		return Source{}, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO job_sources (name, type, url, enabled, parser_type)
		VALUES (?, ?, ?, ?, ?)
	`, input.Name, input.Type, input.URL, boolToInt(input.Enabled), input.ParserType)
	if err != nil {
		return Source{}, fmt.Errorf("insert source: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Source{}, fmt.Errorf("read source id: %w", err)
	}
	return r.GetSource(ctx, id)
}

func (r *Repository) SeedRecommendedSources(ctx context.Context) (SeedSourcesResult, error) {
	result := SeedSourcesResult{Total: len(RecommendedSources())}
	for _, source := range RecommendedSources() {
		if source.Type == "" {
			source.Type = "public_url"
		}
		if source.ParserType == "" {
			source.ParserType = "generic"
		}
		created, err := r.createSourceIfMissing(ctx, source)
		if err != nil {
			return SeedSourcesResult{}, err
		}
		if created {
			result.Created++
		} else {
			result.Duplicated++
		}
	}
	return result, nil
}

func (r *Repository) GetSource(ctx context.Context, id int64) (Source, error) {
	row := r.db.QueryRowContext(ctx, selectSourceSQL()+` WHERE id = ?`, id)
	source, err := scanSource(row)
	if err != nil {
		return Source{}, fmt.Errorf("get source %d: %w", id, err)
	}
	return source, nil
}

func (r *Repository) createSourceIfMissing(ctx context.Context, input SourceInput) (bool, error) {
	input, err := normalizeSourceInput(input)
	if err != nil {
		return false, err
	}
	var existingID int64
	err = r.db.QueryRowContext(ctx, `SELECT id FROM job_sources WHERE name = ?`, input.Name).Scan(&existingID)
	if err == nil {
		_, err = r.db.ExecContext(ctx, `
			UPDATE job_sources
			SET type = ?, url = ?, parser_type = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, input.Type, input.URL, input.ParserType, existingID)
		if err != nil {
			return false, fmt.Errorf("refresh source %q: %w", input.Name, err)
		}
		return false, nil
	}
	if err != sql.ErrNoRows {
		return false, fmt.Errorf("find source %q: %w", input.Name, err)
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO job_sources (name, type, url, enabled, parser_type)
		VALUES (?, ?, ?, ?, ?)
	`, input.Name, input.Type, input.URL, boolToInt(input.Enabled), input.ParserType)
	if err != nil {
		return false, fmt.Errorf("insert source: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read rows affected: %w", err)
	}
	return affected > 0, nil
}

func (r *Repository) ListSources(ctx context.Context, enabledOnly bool) ([]Source, error) {
	query := selectSourceSQL()
	if enabledOnly {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY enabled DESC, updated_at DESC, id DESC"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()

	out := []Source{}
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sources: %w", err)
	}
	return out, nil
}

func (r *Repository) UpdateSourceEnabled(ctx context.Context, id int64, enabled bool) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE job_sources
		SET enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, boolToInt(enabled), id)
	if err != nil {
		return fmt.Errorf("update source enabled: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) UpdateSourceHealthByURL(ctx context.Context, rawURL string, input SourceHealthInput) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	status := normalizeSourceHealthStatus(input.Status)
	now := time.Now().UTC()
	var result sql.Result
	var err error
	if input.Success {
		result, err = r.db.ExecContext(ctx, `
			UPDATE job_sources
			SET last_run_at = ?, health_status = ?, health_reason = ?, consecutive_failures = 0,
				last_success_at = ?, last_found_count = ?, updated_at = CURRENT_TIMESTAMP
			WHERE url = ?
		`, now, status, strings.TrimSpace(input.Reason), now, input.FoundCount, rawURL)
	} else {
		result, err = r.db.ExecContext(ctx, `
			UPDATE job_sources
			SET last_run_at = ?, health_status = ?, health_reason = ?,
				consecutive_failures = consecutive_failures + 1, last_failure_at = ?,
				last_found_count = ?, updated_at = CURRENT_TIMESTAMP
			WHERE url = ?
		`, now, status, strings.TrimSpace(input.Reason), now, input.FoundCount, rawURL)
	}
	if err != nil {
		return fmt.Errorf("update source health: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read source health rows affected: %w", err)
	}
	if affected == 0 {
		return nil
	}
	return nil
}

func (r *Repository) SeedPublicURLSources(ctx context.Context, urls []string) error {
	seen := map[string]bool{}
	for _, rawURL := range urls {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" || seen[rawURL] {
			continue
		}
		seen[rawURL] = true
		parsed, err := url.ParseRequestURI(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			continue
		}
		name := sourceNameFromURL(parsed)
		_, err = r.db.ExecContext(ctx, `
			INSERT INTO job_sources (name, type, url, enabled, parser_type)
			VALUES (?, 'public_url', ?, 1, 'generic')
			ON CONFLICT(name) DO NOTHING
		`, name, parsed.String())
		if err != nil {
			return fmt.Errorf("seed source %q: %w", rawURL, err)
		}
	}
	return nil
}

func normalizeSourceInput(input SourceInput) (SourceInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Type = strings.TrimSpace(input.Type)
	input.URL = strings.TrimSpace(input.URL)
	input.ParserType = strings.TrimSpace(input.ParserType)
	if input.Type == "" {
		input.Type = "public_url"
	}
	if input.ParserType == "" {
		input.ParserType = "generic"
	}
	parsed, err := url.ParseRequestURI(input.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return SourceInput{}, fmt.Errorf("source URL must be a valid http or https URL")
	}
	input.URL = parsed.String()
	if input.Name == "" {
		input.Name = sourceNameFromURL(parsed)
	}
	return input, nil
}

func sourceNameFromURL(parsed *url.URL) string {
	host := strings.TrimPrefix(parsed.Hostname(), "www.")
	if host == "" {
		host = "source"
	}
	return host + " " + strings.Trim(strings.ReplaceAll(parsed.Path, "/", " "), " ")
}

func selectSourceSQL() string {
	return `
		SELECT id, name, type, url, enabled, parser_type, last_run_at,
			health_status, health_reason, consecutive_failures, last_success_at,
			last_failure_at, last_found_count, created_at, updated_at
		FROM job_sources`
}

type sourceScanner interface {
	Scan(dest ...any) error
}

func scanSource(scanner sourceScanner) (Source, error) {
	var source Source
	var enabled int
	if err := scanner.Scan(
		&source.ID,
		&source.Name,
		&source.Type,
		&source.URL,
		&enabled,
		&source.ParserType,
		&source.LastRunAt,
		&source.HealthStatus,
		&source.HealthReason,
		&source.ConsecutiveFailures,
		&source.LastSuccessAt,
		&source.LastFailureAt,
		&source.LastFoundCount,
		&source.CreatedAt,
		&source.UpdatedAt,
	); err != nil {
		return Source{}, fmt.Errorf("scan source: %w", err)
	}
	source.Enabled = enabled == 1
	source.HealthStatus = normalizeSourceHealthStatus(source.HealthStatus)
	return source, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func normalizeSourceHealthStatus(status string) string {
	switch strings.TrimSpace(status) {
	case SourceHealthHealthy, SourceHealthWarning, SourceHealthBroken:
		return strings.TrimSpace(status)
	default:
		return SourceHealthUnknown
	}
}
