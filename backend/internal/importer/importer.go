package importer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
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
	"\u62db\u8058",
	"\u6821\u62db",
	"\u79cb\u62db",
	"\u6625\u62db",
	"\u5e94\u5c4a",
	"\u5b9e\u4e60",
	"\u5c97\u4f4d",
	"\u804c\u4f4d",
	"\u6295\u9012",
	"\u524d\u7aef",
	"\u540e\u7aef",
	"\u7b97\u6cd5",
	"\u5927\u6a21\u578b",
}

var recruitmentURLPattern = regexp.MustCompile("https?://[^\\s\"'\\\\<>]+|/[^\\s\"'\\\\<>]*(?:job|jobs|position|positions|recruit|campus|intern|\u5c97\u4f4d|\u804c\u4f4d|\u6821\u62db|\u793e\u62db)[^\\s\"'\\\\<>]*")

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
	if containsAny(job.Title+" "+job.Description, "shenzhen", "\u6df1\u5733") {
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImportBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read discovery page: %w", err)
	}
	doc, err := html.Parse(bytes.NewReader(body))
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
	for _, link := range discoverLinksInText(string(body), parsed, seen, limit-len(links)) {
		links = append(links, link)
	}
	return links, nil
}

func LooksLikeConcreteJobPosting(job domain.Job) bool {
	text := cleanText(strings.ToLower(strings.Join([]string{job.Title, job.Description, job.ApplyURL, job.SourceURL}, " ")))
	landingSignal := containsAny(text,
		"\u62db\u8058\u5b98\u7f51", "\u6821\u56ed\u62db\u8058\u5b98\u7f51", "\u6821\u56ed\u62db\u8058", "\u793e\u4f1a\u62db\u8058", "\u62db\u8058\u5e73\u53f0", "\u6700\u65b0\u62db\u8058\u4fe1\u606f", "\u6821\u62db\u8d44\u8baf",
		"jobs list", "job list", "/jobs/list", "/static/index.html", "careers home",
	)
	roleSignal := containsAny(text,
		"engineer", "developer", "frontend", "backend", "java engineer", "golang", "go backend", "algorithm", "ai application", "llm",
		"\u5de5\u7a0b\u5e08", "\u5f00\u53d1", "\u524d\u7aef", "\u540e\u7aef", "\u7b97\u6cd5", "\u5927\u6a21\u578b", "ai\u5e94\u7528", "\u5b9e\u4e60\u751f",
	)
	detailSignal := containsAny(text,
		"job description", "responsibilities", "requirements", "apply now", "apply online",
		"\u5c97\u4f4d\u804c\u8d23", "\u4efb\u804c\u8981\u6c42", "\u804c\u4f4d\u63cf\u8ff0", "\u5de5\u4f5c\u804c\u8d23", "\u6295\u9012", "\u7acb\u5373\u7533\u8bf7",
	)
	pathSignal := containsAny(text, "/job/", "/jobs/", "/position/", "/positions/", "requisition", "campus/position")

	if landingSignal && !detailSignal {
		return false
	}
	if roleSignal && (detailSignal || pathSignal || !landingSignal) {
		return true
	}
	return pathSignal && detailSignal && !landingSignal
}

func discoverLinksInText(raw string, base *url.URL, seen map[string]struct{}, limit int) []string {
	if limit <= 0 {
		return []string{}
	}
	raw = strings.ReplaceAll(raw, `\/`, `/`)
	links := []string{}
	for _, match := range recruitmentURLPattern.FindAllString(raw, -1) {
		candidate := strings.Trim(match, `"' ,;)]}`)
		resolved, err := base.Parse(candidate)
		if err != nil || (resolved.Scheme != "http" && resolved.Scheme != "https") || resolved.Host == "" {
			continue
		}
		resolved.Fragment = ""
		link := resolved.String()
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}
		links = append(links, link)
		if len(links) >= limit {
			return links
		}
	}
	return links
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
		if n.Type == html.ElementNode && n.Data == "meta" {
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
			if title == "" && name == "og:title" {
				title = cleanText(content)
			}
			if description == "" && (name == "description" || name == "og:description" || name == "keywords") {
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
