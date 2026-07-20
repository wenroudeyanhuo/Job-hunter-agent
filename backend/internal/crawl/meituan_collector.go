package crawl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

const defaultMeituanCareerEndpoint = "https://campus.meituan.com"

var meituanCareerKeywords = []string{
	"go",
	"java",
	"\u540e\u7aef",
	"\u524d\u7aef",
	"\u7b97\u6cd5",
	"AI",
	"\u5927\u6a21\u578b",
	"\u8f6f\u4ef6",
	"\u670d\u52a1\u7aef",
}

type MeituanCareerCollector struct {
	endpoint string
	client   *http.Client
}

func NewMeituanCareerCollector(client *http.Client) MeituanCareerCollector {
	return newMeituanCareerCollector(defaultMeituanCareerEndpoint, client)
}

func newMeituanCareerCollector(endpoint string, client *http.Client) MeituanCareerCollector {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return MeituanCareerCollector{endpoint: normalizeEndpointRoot(endpoint), client: client}
}

func (MeituanCareerCollector) Name() string {
	return "meituan_careers_api"
}

func (c MeituanCareerCollector) Collect(ctx context.Context) ([]domain.Job, error) {
	out := []domain.Job{}
	seen := map[string]struct{}{}

	for _, keyword := range meituanCareerKeywords {
		jobs, err := c.collectKeyword(ctx, keyword)
		if err != nil {
			return nil, err
		}
		for _, job := range jobs {
			key := job.ApplyURL
			if key == "" {
				key = job.Title + "\x00" + job.City
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, job)
		}
	}
	return out, nil
}

func (c MeituanCareerCollector) collectKeyword(ctx context.Context, keyword string) ([]domain.Job, error) {
	body, err := json.Marshal(map[string]any{
		"page": map[string]any{
			"pageNo":   1,
			"pageSize": 10,
		},
		"keywords": keyword,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/official/job/getJobList", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create Meituan careers request: %w", err)
	}
	req.Header.Set("User-Agent", defaultTencentCareerUserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", c.endpoint)
	req.Header.Set("Referer", c.endpoint+"/web/position")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Meituan careers: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch Meituan careers returned HTTP %d", resp.StatusCode)
	}

	var payload meituanCareerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Meituan careers response: %w", err)
	}

	jobs := []domain.Job{}
	for _, post := range payload.Data.List {
		title := strings.TrimSpace(post.Name)
		if title == "" {
			continue
		}
		jobs = append(jobs, domain.Job{
			Company:      "Meituan",
			Title:        title,
			City:         normalizeMeituanCity(joinMeituanNames(post.CityList)),
			Description:  buildMeituanDescription(post),
			SourceName:   "Meituan Campus",
			SourceURL:    c.endpoint + "/web/position",
			ApplyURL:     normalizeMeituanPostURL(c.endpoint, post.JobUnionID),
			PublishedAt:  parseMeituanMillis(post.RefreshTime),
			DiscoveredAt: time.Now().UTC(),
		})
	}
	return jobs, nil
}

type meituanCareerResponse struct {
	Data struct {
		List []meituanCareerPost `json:"list"`
	} `json:"data"`
}

type meituanCareerPost struct {
	JobUnionID     string              `json:"jobUnionId"`
	Name           string              `json:"name"`
	JobFamily      string              `json:"jobFamily"`
	JobFamilyGroup string              `json:"jobFamilyGroup"`
	CityList       []meituanNamedValue `json:"cityList"`
	Department     []meituanNamedValue `json:"department"`
	JobDuty        string              `json:"jobDuty"`
	JobRequirement string              `json:"jobRequirement"`
	HighLight      string              `json:"highLight"`
	RefreshTime    int64               `json:"refreshTime"`
}

type meituanNamedValue struct {
	Name string `json:"name"`
}

func buildMeituanDescription(post meituanCareerPost) string {
	parts := []string{}
	if value := strings.TrimSpace(joinMeituanNames(post.Department)); value != "" {
		parts = append(parts, "Department: "+value)
	}
	if value := strings.TrimSpace(joinMeituanStrings([]string{post.JobFamily, post.JobFamilyGroup}, " / ")); value != "" {
		parts = append(parts, "Category: "+value)
	}
	for _, value := range []string{post.JobDuty, post.JobRequirement, post.HighLight} {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func joinMeituanNames(values []meituanNamedValue) string {
	names := []string{}
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func joinMeituanStrings(values []string, sep string) string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return strings.Join(out, sep)
}

func normalizeMeituanCity(value string) string {
	if strings.Contains(strings.ToLower(value), "shenzhen") || strings.Contains(value, "\u6df1\u5733") {
		return "Shenzhen"
	}
	return strings.TrimSpace(value)
}

func normalizeMeituanPostURL(endpoint string, id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return endpoint + "/web/position"
	}
	return endpoint + "/web/position/detail?jobUnionId=" + url.QueryEscape(id)
}

func parseMeituanMillis(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	parsed := time.UnixMilli(value).UTC()
	return &parsed
}

func isMeituanCareerSource(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.Contains(host, "meituan.com")
}
