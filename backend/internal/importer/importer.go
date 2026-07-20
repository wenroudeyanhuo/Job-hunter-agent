package importer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"golang.org/x/net/html"
)

const maxImportBodyBytes = 1 << 20
const defaultUserAgent = "JobHunterAgent/0.1 (+https://github.com/wenroudeyanhuo/Job-hunter-agent)"

var recruitmentLinkKeywords = []string{
	"job",
	"jobs",
	"career",
	"careers",
	"campus",
	"graduate",
	"intern",
	"recruit",
	"recruitment",
	"apply",
	"frontend",
	"backend",
	"java",
	"golang",
	"algorithm",
	"ai",
	"llm",
	"招聘",
	"校招",
	"秋招",
	"春招",
	"应届",
	"实习",
	"岗位",
	"职位",
	"投递",
	"前端",
	"后端",
	"算法",
	"大模型",
}

func ImportURL(ctx context.Context, rawURL string, client *http.Client) (domain.Job, error) {
	parsed, err := parseHTTPURL(rawURL)
	if err != nil {
		return domain.Job{}, err
	}
	if client == nil {
		client = http.DefaultClient
	}

	job := domain.Job{
		Company:      companyFromHost(parsed.Hostname()),
		Title:        fallbackTitle(parsed),
		SourceName:   parsed.Hostname(),
		SourceURL:    parsed.String(),
		ApplyURL:     parsed.String(),
		DiscoveredAt: time.Now().UTC(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return domain.Job{}, fmt.Errorf("create import request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		job.Status = domain.StatusManualCheck
		job.Description = "Fetch failed: " + err.Error()
		return job, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		job.Status = domain.StatusManualCheck
		job.Description = fmt.Sprintf("Fetch returned HTTP %d", resp.StatusCode)
		return job, nil
	}

	doc, err := html.Parse(io.LimitReader(resp.Body, maxImportBodyBytes))
	if err != nil {
		job.Status = domain.StatusManualCheck
		job.Description = "HTML parse failed: " + err.Error()
		return job, nil
	}

	title, description := extractTitleAndDescription(doc)
	if title != "" {
		job.Title = title
	}
	if description != "" {
		job.Description = description
	}
	if containsAny(job.Title+" "+job.Description, "shenzhen", "深圳") {
		job.City = "Shenzhen"
	}
	return job, nil
}

func DiscoverLinks(ctx context.Context, rawURL string, client *http.Client, limit int) ([]string, error) {
	parsed, err := parseHTTPURL(rawURL)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		return []string{}, nil
	}
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create discovery request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch discovery page returned HTTP %d", resp.StatusCode)
	}

	doc, err := html.Parse(io.LimitReader(resp.Body, maxImportBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("parse discovery page: %w", err)
	}

	links := []string{}
	seen := map[string]struct{}{
		parsed.String(): {},
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(links) >= limit {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attrValue(n, "href")
			text := cleanText(nodeText(n))
			if href != "" && containsAny(href+" "+text, recruitmentLinkKeywords...) {
				resolved, err := parsed.Parse(href)
				if err == nil && (resolved.Scheme == "http" || resolved.Scheme == "https") && resolved.Host != "" {
					resolved.Fragment = ""
					link := resolved.String()
					if _, ok := seen[link]; !ok {
						seen[link] = struct{}{}
						links = append(links, link)
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
			if len(links) >= limit {
				return
			}
		}
	}
	walk(doc)
	return links, nil
}

func parseHTTPURL(rawURL string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("URL host is required")
	}
	return parsed, nil
}

func extractTitleAndDescription(node *html.Node) (string, string) {
	var title string
	var description string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil && title == "" {
			title = cleanText(n.FirstChild.Data)
		}
		if n.Type == html.ElementNode && n.Data == "meta" && description == "" {
			var name string
			var content string
			for _, attr := range n.Attr {
				switch strings.ToLower(attr.Key) {
				case "name", "property":
					name = strings.ToLower(attr.Val)
				case "content":
					content = attr.Val
				}
			}
			if name == "description" || name == "og:description" {
				description = cleanText(content)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return title, description
}

func companyFromHost(host string) string {
	parts := strings.Split(strings.ToLower(host), ".")
	for _, part := range parts {
		if part == "" || part == "www" || part == "jobs" || part == "careers" || part == "campus" || part == "apply" {
			continue
		}
		return strings.ToUpper(part[:1]) + part[1:]
	}
	return host
}

func fallbackTitle(parsed *url.URL) string {
	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return "Manual import from " + parsed.Hostname()
	}
	path = strings.ReplaceAll(path, "-", " ")
	path = strings.ReplaceAll(path, "_", " ")
	return cleanText(path)
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func attrValue(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func nodeText(node *html.Node) string {
	var values []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			values = append(values, n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(values, " ")
}

func containsAny(value string, needles ...string) bool {
	value = strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(value, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
