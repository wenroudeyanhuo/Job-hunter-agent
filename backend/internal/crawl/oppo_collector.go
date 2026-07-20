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

const defaultOPPOCareerEndpoint = "https://careers.oppo.com"

var oppoCareerKeywords = []string{
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

type OPPOCareerCollector struct {
	endpoint string
	client   *http.Client
}

func NewOPPOCareerCollector(client *http.Client) OPPOCareerCollector {
	return newOPPOCareerCollector(defaultOPPOCareerEndpoint, client)
}

func newOPPOCareerCollector(endpoint string, client *http.Client) OPPOCareerCollector {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return OPPOCareerCollector{endpoint: normalizeEndpointRoot(endpoint), client: client}
}

func (OPPOCareerCollector) Name() string {
	return "oppo_careers_api"
}

func (c OPPOCareerCollector) Collect(ctx context.Context) ([]domain.Job, error) {
	out := []domain.Job{}
	seen := map[string]struct{}{}

	for _, keyword := range oppoCareerKeywords {
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

func (c OPPOCareerCollector) collectKeyword(ctx context.Context, keyword string) ([]domain.Job, error) {
	body, err := json.Marshal(map[string]any{
		"pageNum":          1,
		"pageSize":         10,
		"positionName":     keyword,
		"projectList":      []any{},
		"positionTypeList": []any{},
		"workCityCodeList": []any{},
		"shareId":          "",
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/openapi/position/pageNew", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create OPPO careers request: %w", err)
	}
	req.Header.Set("User-Agent", defaultTencentCareerUserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", c.endpoint)
	req.Header.Set("Referer", c.endpoint+"/university/oppo/campus/post")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch OPPO careers: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch OPPO careers returned HTTP %d", resp.StatusCode)
	}

	var payload oppoCareerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode OPPO careers response: %w", err)
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("OPPO careers returned code %d: %s", payload.Code, payload.Message)
	}

	jobs := []domain.Job{}
	for _, post := range payload.Data.Records {
		title := strings.TrimSpace(post.PositionName)
		if title == "" {
			continue
		}
		description := strings.TrimSpace(strings.Join([]string{post.PositionDesc, post.PositionRequire}, "\n\n"))
		jobs = append(jobs, domain.Job{
			Company:      "OPPO",
			Title:        title,
			City:         normalizeOPPOCity(post.WorkCityName),
			Description:  description,
			SourceName:   "OPPO Careers",
			SourceURL:    c.endpoint + "/university/oppo/campus/post",
			ApplyURL:     normalizeOPPOPostURL(c.endpoint, post.IDProjPosition),
			PublishedAt:  parseOPPOTime(post.ReleaseTime),
			DiscoveredAt: time.Now().UTC(),
		})
	}
	return jobs, nil
}

type oppoCareerResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    struct {
		Records []oppoCareerPost `json:"records"`
		Total   int              `json:"total"`
	} `json:"data"`
}

type oppoCareerPost struct {
	IDProjPosition      int64  `json:"idProjPosition"`
	PositionName        string `json:"positionName"`
	WorkCityName        string `json:"workCityName"`
	PositionDesc        string `json:"positionDesc"`
	PositionRequire     string `json:"positionRequire"`
	ReleaseTime         string `json:"releaseTime"`
	RecruitmentTypeName string `json:"recruitmentTypeName"`
	ProjectName         string `json:"projectName"`
}

func normalizeOPPOCity(value string) string {
	if strings.Contains(strings.ToLower(value), "shenzhen") || strings.Contains(value, "\u6df1\u5733") {
		return "Shenzhen"
	}
	return strings.TrimSpace(value)
}

func normalizeOPPOPostURL(endpoint string, id int64) string {
	if id == 0 {
		return endpoint + "/university/oppo/campus/post"
	}
	return fmt.Sprintf("%s/university/oppo/campus/post/%d", endpoint, id)
}

func parseOPPOTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			utc := parsed.UTC()
			return &utc
		}
	}
	return nil
}

func isOPPOCareerSource(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.Contains(host, "careers.oppo.com") || strings.Contains(host, "career.oppo.com")
}
