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

const defaultByteDanceCareerEndpoint = "https://jobs.bytedance.com"

var byteDanceCareerKeywords = []string{"go", "java", "后端", "前端", "算法", "AI", "大模型"}

type ByteDanceCareerCollector struct {
	endpoint string
	client   *http.Client
}

func NewByteDanceCareerCollector(client *http.Client) ByteDanceCareerCollector {
	return newByteDanceCareerCollector(defaultByteDanceCareerEndpoint, client)
}

func newByteDanceCareerCollector(endpoint string, client *http.Client) ByteDanceCareerCollector {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return ByteDanceCareerCollector{endpoint: normalizeEndpointRoot(endpoint), client: client}
}

func (ByteDanceCareerCollector) Name() string {
	return "bytedance_careers_api"
}

func (c ByteDanceCareerCollector) Collect(ctx context.Context) ([]domain.Job, error) {
	out := []domain.Job{}
	seen := map[string]struct{}{}
	for _, keyword := range byteDanceCareerKeywords {
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

func (c ByteDanceCareerCollector) collectKeyword(ctx context.Context, keyword string) ([]domain.Job, error) {
	body, err := json.Marshal(map[string]any{
		"keyword":                   keyword,
		"limit":                     10,
		"offset":                    0,
		"job_category_id_list":      []string{},
		"location_code_list":        []string{},
		"subject_id_list":           []string{},
		"recruitment_id_list":       []string{},
		"portal_type":               6,
		"portal_entrance":           1,
		"job_hot_flag":              "",
		"function_category_id_list": []string{},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/v1/search/job/posts", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create ByteDance careers request: %w", err)
	}
	req.Header.Set("User-Agent", defaultTencentCareerUserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("portal-channel", "campus")
	req.Header.Set("website-path", "campus")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch ByteDance careers: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch ByteDance careers returned HTTP %d", resp.StatusCode)
	}

	var payload byteDanceCareerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode ByteDance careers response: %w", err)
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("ByteDance careers returned code %d", payload.Code)
	}

	jobs := []domain.Job{}
	for _, post := range payload.Data.JobPostList {
		title := strings.TrimSpace(post.Title)
		if title == "" {
			continue
		}
		description := strings.TrimSpace(strings.Join([]string{post.Description, post.Requirement}, "\n\n"))
		jobs = append(jobs, domain.Job{
			Company:      "ByteDance",
			Title:        title,
			City:         normalizeByteDanceCity(post.CityInfo.Name),
			Description:  description,
			SourceName:   "ByteDance Jobs",
			SourceURL:    c.endpoint + "/campus/",
			ApplyURL:     normalizeByteDancePostURL(c.endpoint, post.ID),
			DiscoveredAt: time.Now().UTC(),
		})
	}
	return jobs, nil
}

type byteDanceCareerResponse struct {
	Code int `json:"code"`
	Data struct {
		Count       int                   `json:"count"`
		JobPostList []byteDanceCareerPost `json:"job_post_list"`
	} `json:"data"`
}

type byteDanceCareerPost struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Requirement string `json:"requirement"`
	CityInfo    struct {
		Name string `json:"name"`
	} `json:"city_info"`
}

func normalizeByteDanceCity(value string) string {
	if strings.Contains(strings.ToLower(value), "shenzhen") || strings.Contains(value, "深圳") {
		return "Shenzhen"
	}
	return strings.TrimSpace(value)
}

func normalizeByteDancePostURL(endpoint string, id string) string {
	if strings.TrimSpace(id) == "" {
		return endpoint + "/campus/position"
	}
	return endpoint + "/campus/position/" + url.PathEscape(id)
}
