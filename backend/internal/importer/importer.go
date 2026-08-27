package importer

import (
	"bytes"
	"context"
	"encoding/json"
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
const defaultImportHTTPTimeout = 20 * time.Second

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
var deadlinePattern = regexp.MustCompile(`(?i)(deadline|截止|截止时间|申请截止|投递截止)[:：\s]*(\d{4}[-/.年]\d{1,2}[-/.月]\d{1,2}日?)`)

func ImportURL(ctx context.Context, rawURL string, client *http.Client) (domain.Job, error) {
	parsed, err := parseHTTPURL(rawURL)
	if err != nil {
		return domain.Job{}, err
	}
	if client == nil {
		client = &http.Client{Timeout: defaultImportHTTPTimeout}
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
	job.DirectionTags = directionTagsFromText(job.Title + " " + job.Description)
	job.DeadlineAt = deadlineFromText(job.Title + " " + job.Description)
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
		client = &http.Client{Timeout: defaultImportHTTPTimeout}
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

func DiscoverJobCards(ctx context.Context, rawURL string, client *http.Client, limit int) ([]domain.Job, error) {
	parsed, err := parseHTTPURL(rawURL)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		return []domain.Job{}, nil
	}
	if client == nil {
		client = &http.Client{Timeout: defaultImportHTTPTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create job-card discovery request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch job-card page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch job-card page returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImportBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read job-card page: %w", err)
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse job-card page: %w", err)
	}
	jobs := []domain.Job{}
	seen := map[string]struct{}{}
	for _, job := range jobsFromJSONLD(doc, parsed) {
		if _, exists := seen[job.ApplyURL]; exists {
			continue
		}
		seen[job.ApplyURL] = struct{}{}
		jobs = append(jobs, job)
		if len(jobs) >= limit {
			return jobs, nil
		}
	}
	for _, job := range jobsFromEmbeddedJSON(doc, parsed) {
		if _, exists := seen[job.ApplyURL]; exists {
			continue
		}
		seen[job.ApplyURL] = struct{}{}
		jobs = append(jobs, job)
		if len(jobs) >= limit {
			return jobs, nil
		}
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(jobs) >= limit {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			if job, ok := jobFromAnchorCard(n, parsed); ok {
				if _, exists := seen[job.ApplyURL]; !exists {
					seen[job.ApplyURL] = struct{}{}
					jobs = append(jobs, job)
				}
			}
		}
		if n.Type == html.ElementNode && n.Data != "a" {
			if job, ok := jobFromStructuredCard(n, parsed); ok {
				if _, exists := seen[job.ApplyURL]; !exists {
					seen[job.ApplyURL] = struct{}{}
					jobs = append(jobs, job)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
			if len(jobs) >= limit {
				return
			}
		}
	}
	walk(doc)
	return jobs, nil
}

func jobsFromEmbeddedJSON(doc *html.Node, base *url.URL) []domain.Job {
	out := []domain.Job{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" && n.FirstChild != nil {
			id := strings.ToLower(attrValue(n, "id"))
			scriptType := strings.ToLower(attrValue(n, "type"))
			if strings.Contains(scriptType, "json") || strings.Contains(id, "__next_data__") || strings.Contains(id, "initial") {
				var payload any
				if err := json.Unmarshal([]byte(strings.TrimSpace(n.FirstChild.Data)), &payload); err == nil {
					out = append(out, parseGenericJobNodes(payload, base)...)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return out
}

func parseGenericJobNodes(value any, base *url.URL) []domain.Job {
	switch typed := value.(type) {
	case []any:
		out := []domain.Job{}
		for _, item := range typed {
			out = append(out, parseGenericJobNodes(item, base)...)
		}
		return out
	case map[string]any:
		out := []domain.Job{}
		if job, ok := jobFromGenericJSONMap(typed, base); ok {
			out = append(out, job)
		}
		for _, item := range typed {
			out = append(out, parseGenericJobNodes(item, base)...)
		}
		return out
	default:
		return nil
	}
}

func jobFromGenericJSONMap(payload map[string]any, base *url.URL) (domain.Job, bool) {
	title := firstJSONString(payload, "title", "name", "jobName", "positionName", "position", "job_title")
	applyURL := firstJSONString(payload, "url", "applyUrl", "apply_url", "jobUrl", "job_url", "detailUrl", "detail_url", "link", "href")
	if title == "" || applyURL == "" {
		return domain.Job{}, false
	}
	resolved, err := base.Parse(applyURL)
	if err != nil || resolved.Host == "" || (resolved.Scheme != "http" && resolved.Scheme != "https") {
		return domain.Job{}, false
	}
	resolved.Fragment = ""
	description := strings.Join(nonEmptyStrings(
		firstJSONString(payload, "description", "jobDesc", "job_desc", "desc"),
		firstJSONString(payload, "responsibilities", "responsibility", "duty"),
		firstJSONString(payload, "requirements", "requirement", "qualification"),
	), " ")
	company := firstJSONString(payload, "company", "companyName", "company_name", "department")
	if company == "" {
		company = companyFromHost(base.Hostname())
	}
	city := cityFromText(firstJSONString(payload, "city", "location", "workCity", "work_city", "workLocation", "work_location") + " " + title + " " + description)
	job := domain.Job{
		Company:       company,
		Title:         title,
		City:          city,
		DirectionTags: directionTagsFromText(title + " " + description),
		Description:   cardDescription(description, title),
		SourceName:    base.Hostname(),
		SourceURL:     base.String(),
		ApplyURL:      resolved.String(),
		DeadlineAt:    deadlineFromJSONString(firstJSONString(payload, "deadline", "validThrough", "endDate", "end_time", "expireTime")),
		DiscoveredAt:  time.Now().UTC(),
	}
	if !LooksLikeConcreteJobPosting(job) {
		return domain.Job{}, false
	}
	return job, true
}

func jobsFromJSONLD(doc *html.Node, base *url.URL) []domain.Job {
	out := []domain.Job{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" && strings.Contains(strings.ToLower(attrValue(n, "type")), "ld+json") && n.FirstChild != nil {
			out = append(out, parseJSONLDJobs(n.FirstChild.Data, base)...)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return out
}

func parseJSONLDJobs(raw string, base *url.URL) []domain.Job {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	return parseJSONLDJobNodes(payload, base)
}

func parseJSONLDJobNodes(value any, base *url.URL) []domain.Job {
	switch typed := value.(type) {
	case []any:
		out := []domain.Job{}
		for _, item := range typed {
			out = append(out, parseJSONLDJobNodes(item, base)...)
		}
		return out
	case map[string]any:
		out := []domain.Job{}
		if graph, ok := typed["@graph"]; ok {
			out = append(out, parseJSONLDJobNodes(graph, base)...)
		}
		if jsonLDTypeMatches(typed["@type"], "JobPosting") {
			if job, ok := jobFromJSONLDMap(typed, base); ok {
				out = append(out, job)
			}
		}
		return out
	default:
		return nil
	}
}

func jsonLDTypeMatches(value any, want string) bool {
	switch typed := value.(type) {
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), want)
	case []any:
		for _, item := range typed {
			if jsonLDTypeMatches(item, want) {
				return true
			}
		}
	}
	return false
}

func jobFromJSONLDMap(payload map[string]any, base *url.URL) (domain.Job, bool) {
	title := cleanText(jsonString(payload["title"]))
	description := cleanText(jsonString(payload["description"]))
	if title == "" {
		return domain.Job{}, false
	}
	applyURL := jsonString(payload["url"])
	if applyURL == "" {
		applyURL = base.String()
	}
	resolved, err := base.Parse(applyURL)
	if err != nil || resolved.Host == "" || (resolved.Scheme != "http" && resolved.Scheme != "https") {
		return domain.Job{}, false
	}
	resolved.Fragment = ""
	company := jsonNestedString(payload["hiringOrganization"], "name")
	if company == "" {
		company = companyFromHost(base.Hostname())
	}
	city := cityFromText(jsonNestedString(payload["jobLocation"], "address", "addressLocality") + " " + jsonString(payload["jobLocation"]) + " " + title + " " + description)
	job := domain.Job{
		Company:       company,
		Title:         title,
		City:          city,
		DirectionTags: directionTagsFromText(title + " " + description),
		Description:   description,
		SourceName:    base.Hostname(),
		SourceURL:     base.String(),
		ApplyURL:      resolved.String(),
		DeadlineAt:    deadlineFromJSONString(jsonString(payload["validThrough"])),
		DiscoveredAt:  time.Now().UTC(),
	}
	if !LooksLikeConcreteJobPosting(job) {
		return domain.Job{}, false
	}
	return job, true
}

func jsonString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return strings.TrimSpace(jsonNestedString(typed, "name"))
	}
	return ""
}

func jsonNestedString(value any, path ...string) string {
	current := value
	for _, key := range path {
		mapping, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = mapping[key]
	}
	return jsonString(current)
}

func firstJSONString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value := jsonString(payload[key])
		if value != "" {
			return cleanText(value)
		}
	}
	return ""
}

func nonEmptyStrings(values ...string) []string {
	out := []string{}
	for _, value := range values {
		value = cleanText(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func deadlineFromJSONString(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return &parsed
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed
	}
	return deadlineFromText("deadline " + value)
}

func DiscoverPaginationLinks(ctx context.Context, rawURL string, client *http.Client, limit int) ([]string, error) {
	parsed, err := parseHTTPURL(rawURL)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		return []string{}, nil
	}
	if client == nil {
		client = &http.Client{Timeout: defaultImportHTTPTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create pagination discovery request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch pagination page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch pagination page returned HTTP %d", resp.StatusCode)
	}
	doc, err := html.Parse(io.LimitReader(resp.Body, maxImportBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("parse pagination page: %w", err)
	}
	links := []string{}
	seen := map[string]struct{}{parsed.String(): {}}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(links) >= limit {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attrValue(n, "href")
			text := strings.ToLower(cleanText(nodeText(n)) + " " + attrValue(n, "rel") + " " + attrValue(n, "aria-label") + " " + attrValue(n, "class"))
			if href != "" && looksLikePaginationLink(text, href) {
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

func jobFromAnchorCard(node *html.Node, base *url.URL) (domain.Job, bool) {
	href := attrValue(node, "href")
	if href == "" {
		return domain.Job{}, false
	}
	resolved, err := base.Parse(href)
	if err != nil || (resolved.Scheme != "http" && resolved.Scheme != "https") || resolved.Host == "" {
		return domain.Job{}, false
	}
	resolved.Fragment = ""
	text := cleanText(nodeText(node))
	title := firstHeadingText(node)
	if title == "" {
		title = firstUsefulLine(text)
	}
	if title == "" {
		return domain.Job{}, false
	}
	if !containsAny(href+" "+title+" "+text, recruitmentLinkKeywords...) {
		return domain.Job{}, false
	}
	job := domain.Job{
		Company:      companyFromHost(base.Hostname()),
		Title:        title,
		City:         cityFromText(text),
		Description:  cardDescription(text, title),
		SourceName:   base.Hostname(),
		SourceURL:    base.String(),
		ApplyURL:     resolved.String(),
		DiscoveredAt: time.Now().UTC(),
	}
	job.DirectionTags = directionTagsFromText(title + " " + text)
	job.DeadlineAt = deadlineFromText(text)
	if !LooksLikeConcreteJobPosting(job) {
		return domain.Job{}, false
	}
	return job, true
}

func jobFromStructuredCard(node *html.Node, base *url.URL) (domain.Job, bool) {
	href := firstAttrValue(node, "data-href", "data-url", "data-apply-url", "data-link")
	if href == "" {
		return domain.Job{}, false
	}
	resolved, err := base.Parse(href)
	if err != nil || (resolved.Scheme != "http" && resolved.Scheme != "https") || resolved.Host == "" {
		return domain.Job{}, false
	}
	resolved.Fragment = ""
	text := cleanText(nodeText(node))
	title := firstHeadingText(node)
	if title == "" {
		title = firstUsefulLine(text)
	}
	if title == "" || !containsAny(href+" "+title+" "+text, recruitmentLinkKeywords...) {
		return domain.Job{}, false
	}
	job := domain.Job{
		Company:       companyFromHost(base.Hostname()),
		Title:         title,
		City:          cityFromText(text),
		DirectionTags: directionTagsFromText(title + " " + text),
		Description:   cardDescription(text, title),
		SourceName:    base.Hostname(),
		SourceURL:     base.String(),
		ApplyURL:      resolved.String(),
		DeadlineAt:    deadlineFromText(text),
		DiscoveredAt:  time.Now().UTC(),
	}
	if !LooksLikeConcreteJobPosting(job) {
		return domain.Job{}, false
	}
	return job, true
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

func firstAttrValue(node *html.Node, keys ...string) string {
	for _, key := range keys {
		if value := attrValue(node, key); value != "" {
			return value
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

func firstHeadingText(node *html.Node) string {
	var out string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if out != "" {
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1", "h2", "h3", "h4", "strong", "b":
				out = cleanText(nodeText(n))
				return
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return out
}

func firstUsefulLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = cleanText(line)
		if len([]rune(line)) >= 4 {
			return line
		}
	}
	return cleanText(text)
}

func cityFromText(text string) string {
	if containsAny(text, "shenzhen", "深圳") {
		return "Shenzhen"
	}
	if containsAny(text, "guangzhou", "广州") {
		return "Guangzhou"
	}
	if containsAny(text, "shanghai", "上海") {
		return "Shanghai"
	}
	if containsAny(text, "beijing", "北京") {
		return "Beijing"
	}
	if containsAny(text, "hangzhou", "杭州") {
		return "Hangzhou"
	}
	return ""
}

func directionTagsFromText(text string) []string {
	lower := strings.ToLower(text)
	tags := []string{}
	if containsAny(lower, "frontend", "\u524d\u7aef") {
		tags = append(tags, "frontend")
	}
	if containsAny(lower, "backend", "\u540e\u7aef", "\u670d\u52a1\u7aef") {
		tags = append(tags, "backend")
	}
	if containsAny(lower, "java") {
		tags = append(tags, "java")
	}
	if containsAny(lower, "go", "golang") {
		tags = append(tags, "go")
	}
	if containsAny(lower, "algorithm", "\u7b97\u6cd5") {
		tags = append(tags, "algorithm")
	}
	if containsAny(lower, "llm", "ai", "\u5927\u6a21\u578b", "agent") {
		tags = append(tags, "ai_application")
	}
	return uniqueStrings(tags)
}

func deadlineFromText(text string) *time.Time {
	match := deadlinePattern.FindStringSubmatch(text)
	if len(match) < 3 {
		return nil
	}
	value := strings.NewReplacer("年", "-", "月", "-", "日", "", "/", "-", ".", "-").Replace(match[2])
	parsed, err := time.Parse("2006-1-2", value)
	if err != nil {
		return nil
	}
	return &parsed
}

func looksLikePaginationLink(text string, href string) bool {
	lower := strings.ToLower(href + " " + text)
	return strings.Contains(lower, "page=") ||
		strings.Contains(lower, "page/") ||
		strings.Contains(lower, "next") ||
		strings.Contains(lower, "more") ||
		strings.Contains(lower, "\u4e0b\u4e00\u9875") ||
		strings.Contains(lower, "\u66f4\u591a")
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func cardDescription(text string, title string) string {
	text = cleanText(strings.Replace(text, title, "", 1))
	if len([]rune(text)) > 320 {
		runes := []rune(text)
		return string(runes[:320])
	}
	return text
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
