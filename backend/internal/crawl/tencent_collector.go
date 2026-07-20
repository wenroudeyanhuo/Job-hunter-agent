package crawl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

const defaultTencentCareerEndpoint = "https://careers.tencent.com"
const defaultTencentCareerUserAgent = "JobHunterAgent/0.1 (+https://github.com/wenroudeyanhuo/Job-hunter-agent)"

var tencentCareerKeywords = []string{"go", "java", "后端", "前端", "算法", "AI", "大模型"}

type TencentCareerCollector struct {
	endpoint string
	client   *http.Client
}

func NewTencentCareerCollector(client *http.Client) TencentCareerCollector {
	return newTencentCareerCollector(defaultTencentCareerEndpoint, client)
}

func newTencentCareerCollector(endpoint string, client *http.Client) TencentCareerCollector {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return TencentCareerCollector{endpoint: strings.TrimRight(endpoint, "/"), client: client}
}

func (TencentCareerCollector) Name() string {
	return "tencent_careers_api"
}

func (c TencentCareerCollector) Collect(ctx context.Context) ([]domain.Job, error) {
	out := []domain.Job{}
	seen := map[string]struct{}{}
	for _, keyword := range tencentCareerKeywords {
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

func (c TencentCareerCollector) collectKeyword(ctx context.Context, keyword string) ([]domain.Job, error) {
	queryURL, err := url.Parse(c.endpoint + "/tencentcareer/api/post/Query")
	if err != nil {
		return nil, err
	}
	q := queryURL.Query()
	q.Set("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	q.Set("countryId", "")
	q.Set("cityId", "")
	q.Set("bgIds", "")
	q.Set("productId", "")
	q.Set("categoryId", "")
	q.Set("parentCategoryId", "")
	q.Set("attrId", "1")
	q.Set("keyword", keyword)
	q.Set("pageIndex", "1")
	q.Set("pageSize", "10")
	q.Set("language", "zh-cn")
	q.Set("area", "cn")
	queryURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create Tencent careers request: %w", err)
	}
	req.Header.Set("User-Agent", defaultTencentCareerUserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Tencent careers: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch Tencent careers returned HTTP %d", resp.StatusCode)
	}

	var payload tencentCareerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Tencent careers response: %w", err)
	}
	if payload.Code != 200 {
		return nil, fmt.Errorf("Tencent careers returned code %d", payload.Code)
	}

	jobs := []domain.Job{}
	for _, post := range payload.Data.Posts {
		title := strings.TrimSpace(post.RecruitPostName)
		if title == "" {
			continue
		}
		jobs = append(jobs, domain.Job{
			Company:      "Tencent",
			Title:        title,
			City:         normalizeTencentCity(post.LocationName),
			Description:  strings.TrimSpace(post.Responsibility),
			SourceName:   "Tencent Careers",
			SourceURL:    c.endpoint + "/",
			ApplyURL:     normalizeTencentPostURL(post.PostURL, post.PostID),
			DiscoveredAt: time.Now().UTC(),
		})
	}
	return jobs, nil
}

type tencentCareerResponse struct {
	Code int `json:"Code"`
	Data struct {
		Posts []tencentCareerPost `json:"Posts"`
	} `json:"Data"`
}

type tencentCareerPost struct {
	PostID          string `json:"PostId"`
	RecruitPostName string `json:"RecruitPostName"`
	LocationName    string `json:"LocationName"`
	Responsibility  string `json:"Responsibility"`
	PostURL         string `json:"PostURL"`
}

func normalizeTencentCity(value string) string {
	if strings.Contains(strings.ToLower(value), "shenzhen") || strings.Contains(value, "深圳") {
		return "Shenzhen"
	}
	return strings.TrimSpace(value)
}

func normalizeTencentPostURL(raw string, postID string) string {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		raw = strings.Replace(raw, "http://", "https://", 1)
		return raw
	}
	if postID == "" {
		return defaultTencentCareerEndpoint
	}
	return defaultTencentCareerEndpoint + "/jobdesc.html?postId=" + url.QueryEscape(postID)
}
