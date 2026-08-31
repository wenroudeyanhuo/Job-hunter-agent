package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
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

const (
	SourceCandidateDiscoveredByRules     = "rules"
	SourceCandidateDiscoveredByWebSearch = "web_search"
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
	EnableWebSearch  bool     `json:"enable_web_search"`
	SearchLimit      int      `json:"search_limit"`
	SearchEndpoint   string   `json:"-"`
}

type SourceCandidateFilter struct {
	Status string
}

type SourceDiscoveryResult struct {
	Total               int `json:"total"`
	Created             int `json:"created"`
	Duplicated          int `json:"duplicated"`
	WebSearchCandidates int `json:"web_search_candidates"`
}

type sourceCandidateInput struct {
	Name         string
	URL          string
	Category     string
	ParserType   string
	Reason       string
	Confidence   int
	DiscoveredBy string
}

func (r *Repository) DiscoverSourceCandidates(ctx context.Context, input SourceDiscoveryInput) (SourceDiscoveryResult, error) {
	candidates := BuildSourceDiscoveryCandidates(input)
	if input.EnableWebSearch && activeSourceWebSearchEnabled(input.SearchEndpoint) {
		discovered, err := DiscoverActiveSourceCandidates(ctx, input, &http.Client{Timeout: 8 * time.Second})
		if err != nil {
			discovered = nil
		}
		candidates = append(candidates, discovered...)
		candidates = dedupeSourceCandidateInputs(candidates)
	}
	result := SourceDiscoveryResult{Total: len(candidates)}
	for _, candidate := range candidates {
		if candidate.DiscoveredBy == SourceCandidateDiscoveredByWebSearch {
			result.WebSearchCandidates++
		}
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

func activeSourceWebSearchEnabled(searchEndpoint string) bool {
	if strings.TrimSpace(searchEndpoint) != "" {
		return true
	}
	return strings.TrimSpace(os.Getenv("ACTIVE_SOURCE_WEB_SEARCH")) != "0"
}

func BuildActiveSourceSearchQueries(input SourceDiscoveryInput) []string {
	cities := cleanStringList(input.TargetCities)
	if len(cities) == 0 {
		cities = []string{"Shenzhen"}
	}
	directions := cleanStringList(input.TargetDirections)
	if len(directions) == 0 {
		directions = []string{"go", "backend", "ai_application"}
	}
	limit := input.SearchLimit
	if limit <= 0 || limit > 8 {
		limit = 6
	}
	queries := []string{}
	for _, city := range cities {
		for _, direction := range directions {
			label := directionLabel(direction)
			queries = append(queries,
				city+" "+label+" 校招 招聘 官网",
				city+" "+label+" 实习生 campus careers",
			)
			if len(queries) >= limit {
				return queries
			}
		}
	}
	return queries
}

func DiscoverActiveSourceCandidates(ctx context.Context, input SourceDiscoveryInput, client *http.Client) ([]sourceCandidateInput, error) {
	if client == nil {
		client = http.DefaultClient
	}
	queries := BuildActiveSourceSearchQueries(input)
	out := []sourceCandidateInput{}
	for _, query := range queries {
		results, err := fetchPublicSearchResults(ctx, client, input.SearchEndpoint, query)
		if err != nil {
			continue
		}
		for _, result := range results {
			candidate, ok := sourceCandidateFromSearchResult(result, query)
			if !ok {
				continue
			}
			out = append(out, candidate)
		}
	}
	return dedupeSourceCandidateInputs(out), nil
}

type publicSearchResult struct {
	Title string
	URL   string
}

func fetchPublicSearchResults(ctx context.Context, client *http.Client, endpoint string, query string) ([]publicSearchResult, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = "https://www.bing.com/search"
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	values := parsed.Query()
	values.Set("q", query)
	parsed.RawQuery = values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", sourceCandidateUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSourceCandidateValidationBytes))
	if err != nil {
		return nil, err
	}
	return parsePublicSearchResults(string(body)), nil
}

var searchAnchorPattern = regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
var htmlTagPattern = regexp.MustCompile(`(?is)<[^>]+>`)

func parsePublicSearchResults(html string) []publicSearchResult {
	matches := searchAnchorPattern.FindAllStringSubmatch(html, -1)
	out := []publicSearchResult{}
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		link := normalizeSearchResultURL(match[1])
		if link == "" {
			continue
		}
		title := strings.TrimSpace(htmlTagPattern.ReplaceAllString(match[2], " "))
		title = strings.Join(strings.Fields(title), " ")
		out = append(out, publicSearchResult{Title: title, URL: link})
	}
	return out
}

func normalizeSearchResultURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return ""
	}
	if strings.HasPrefix(raw, "/url?") || strings.HasPrefix(raw, "/search?") {
		if parsed, err := url.Parse(raw); err == nil {
			for _, key := range []string{"q", "url", "u"} {
				if value := parsed.Query().Get(key); value != "" {
					raw = value
					break
				}
			}
		}
	}
	raw = strings.TrimSpace(raw)
	if decoded, err := url.QueryUnescape(raw); err == nil {
		raw = decoded
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if isSearchEngineHost(parsed.Host) {
		return ""
	}
	parsed.Fragment = ""
	return parsed.String()
}

func sourceCandidateFromSearchResult(result publicSearchResult, query string) (sourceCandidateInput, bool) {
	link := normalizeSearchResultURL(result.URL)
	if link == "" {
		return sourceCandidateInput{}, false
	}
	text := strings.ToLower(result.Title + " " + link)
	if !looksLikeRecruitingSearchResult(text) {
		return sourceCandidateInput{}, false
	}
	parsed, _ := url.Parse(link)
	name := strings.TrimSpace(result.Title)
	if name == "" {
		name = parsed.Host
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return sourceCandidateInput{
		Name:         name + " discovery",
		URL:          link,
		Category:     "web_search",
		ParserType:   "generic",
		Reason:       "Active Source Scout found this public recruiting result from query: " + query,
		Confidence:   activeSearchConfidence(text),
		DiscoveredBy: SourceCandidateDiscoveredByWebSearch,
	}, true
}

func looksLikeRecruitingSearchResult(text string) bool {
	if countContainsAny(text, "校招", "招聘", "实习", "职位") > 0 &&
		!strings.Contains(text, "培训") &&
		!strings.Contains(text, "课程") {
		return true
	}
	return countContainsAny(text, "career", "careers", "campus", "job", "jobs", "hiring", "recruit", "recruitment", "校招", "招聘", "实习", "职位") > 0 &&
		!strings.Contains(text, "培训") &&
		!strings.Contains(text, "课程")
}

func activeSearchConfidence(text string) int {
	score := 45 + countContainsAny(text, "career", "careers", "campus", "job", "jobs", "hiring", "recruit", "校招", "招聘", "职位")*5
	if strings.Contains(text, "official") || strings.Contains(text, "官网") {
		score += 8
	}
	if score > 82 {
		score = 82
	}
	return score
}

func isSearchEngineHost(host string) bool {
	host = strings.ToLower(host)
	return strings.Contains(host, "bing.com") || strings.Contains(host, "google.com") || strings.Contains(host, "baidu.com") || strings.Contains(host, "sogou.com")
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
			Name:         source.Name + " discovery",
			URL:          source.URL,
			Category:     source.Category,
			ParserType:   source.ParserType,
			Reason:       "Official career source adjacent to the configured company pool.",
			Confidence:   72,
			DiscoveredBy: SourceCandidateDiscoveredByRules,
		})
	}
	for _, source := range broaderCompanyCareerCandidates() {
		out = append(out, sourceCandidateInput{
			Name:         source.Name,
			URL:          source.URL,
			Category:     source.Category,
			ParserType:   source.ParserType,
			Reason:       "Broader official career entrance for personal source expansion beyond top-tier companies.",
			Confidence:   source.Confidence,
			DiscoveredBy: SourceCandidateDiscoveredByRules,
		})
	}
	for _, city := range cities {
		for _, direction := range directions {
			query := city + " " + directionLabel(direction) + " \u6821\u62db \u5b9e\u4e60 \u62db\u8058"
			out = append(out,
				sourceCandidateInput{
					Name:         "Nowcoder search - " + city + " " + directionLabel(direction),
					URL:          "https://www.nowcoder.com/search?query=" + url.QueryEscape(query),
					Category:     "community",
					ParserType:   "generic",
					Reason:       "Community search can surface fresh campus openings beyond fixed official sources.",
					Confidence:   62,
					DiscoveredBy: SourceCandidateDiscoveredByRules,
				},
				sourceCandidateInput{
					Name:         "Boss search - " + city + " " + directionLabel(direction),
					URL:          "https://www.zhipin.com/web/geek/job?query=" + url.QueryEscape(directionLabel(direction)) + "&city=101280600",
					Category:     "job_platform",
					ParserType:   "generic",
					Reason:       "Job-platform query candidate derived from current city and direction preferences.",
					Confidence:   55,
					DiscoveredBy: SourceCandidateDiscoveredByRules,
				},
				sourceCandidateInput{
					Name:         "Lagou search - " + city + " " + directionLabel(direction),
					URL:          "https://www.lagou.com/wn/jobs?kd=" + url.QueryEscape(directionLabel(direction)),
					Category:     "job_platform",
					ParserType:   "generic",
					Reason:       "Platform search candidate for broadening non-official source coverage.",
					Confidence:   52,
					DiscoveredBy: SourceCandidateDiscoveredByRules,
				},
				sourceCandidateInput{
					Name:         "Liepin search - " + city + " " + directionLabel(direction),
					URL:          "https://www.liepin.com/zhaopin/?key=" + url.QueryEscape(city+" "+directionLabel(direction)+" 校招"),
					Category:     "job_platform",
					ParserType:   "generic",
					Reason:       "Liepin search can broaden mid-size company coverage beyond campus-only official sites.",
					Confidence:   50,
					DiscoveredBy: SourceCandidateDiscoveredByRules,
				},
				sourceCandidateInput{
					Name:         "Maimai search - " + city + " " + directionLabel(direction),
					URL:          "https://maimai.cn/web/search_center?type=feed&query=" + url.QueryEscape(city+" "+directionLabel(direction)+" 招聘 校招"),
					Category:     "community",
					ParserType:   "generic",
					Reason:       "Professional community search can surface team-level recruiting posts and referral links.",
					Confidence:   48,
					DiscoveredBy: SourceCandidateDiscoveredByRules,
				},
				sourceCandidateInput{
					Name:         "GitHub topic search - " + city + " " + directionLabel(direction),
					URL:          "https://github.com/search?q=" + url.QueryEscape(city+" "+directionLabel(direction)+" campus hiring jobs"),
					Category:     "community",
					ParserType:   "generic",
					Reason:       "GitHub search can discover open-source or startup hiring pages relevant to technical roles.",
					Confidence:   45,
					DiscoveredBy: SourceCandidateDiscoveredByRules,
				},
			)
		}
	}
	return dedupeSourceCandidateInputs(out)
}

func broaderCompanyCareerCandidates() []struct {
	Name       string
	URL        string
	Category   string
	ParserType string
	Confidence int
} {
	return []struct {
		Name       string
		URL        string
		Category   string
		ParserType string
		Confidence int
	}{
		{Name: "Bilibili Careers discovery", URL: "https://jobs.bilibili.com/", Category: "internet", ParserType: "generic", Confidence: 66},
		{Name: "Xiaohongshu Careers discovery", URL: "https://job.xiaohongshu.com/", Category: "internet", ParserType: "generic", Confidence: 66},
		{Name: "Shopee Careers discovery", URL: "https://careers.shopee.sg/jobs", Category: "cross_border", ParserType: "generic", Confidence: 62},
		{Name: "MiHoYo Careers discovery", URL: "https://jobs.mihoyo.com/", Category: "game", ParserType: "generic", Confidence: 62},
		{Name: "NetEase Careers discovery", URL: "https://hr.163.com/", Category: "internet", ParserType: "generic", Confidence: 60},
		{Name: "Kingsoft Careers discovery", URL: "https://join.kingsoft.com/", Category: "software", ParserType: "generic", Confidence: 58},
		{Name: "SHEIN Careers discovery", URL: "https://careers.sheingroup.com/", Category: "cross_border", ParserType: "generic", Confidence: 58},
	}
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
			SET name = ?, category = ?, parser_type = ?, discovered_by = ?, reason = ?, confidence = ?,
				validation_status = ?, validation_reason = ?, last_checked_at = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND status = ?
		`, input.Name, input.Category, input.ParserType, input.DiscoveredBy, input.Reason, input.Confidence,
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, input.Name, input.URL, input.Category, input.ParserType, input.DiscoveredBy, input.Reason, input.Confidence,
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
	input.DiscoveredBy = strings.TrimSpace(input.DiscoveredBy)
	if input.DiscoveredBy == "" {
		input.DiscoveredBy = SourceCandidateDiscoveredByRules
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
