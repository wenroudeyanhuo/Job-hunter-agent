package jobs

import (
	"strings"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type ScoreResult struct {
	Job              domain.Job
	HardFiltered     bool
	HardFilterReason string
}

func ScoreJob(input domain.Job) ScoreResult {
	job := input
	text := normalizedSearchText(job.Company, job.Title, job.City, job.Description)

	if filtered, reason := IsHardFiltered(job); filtered {
		return ScoreResult{Job: job, HardFiltered: true, HardFilterReason: reason}
	}

	score := 0
	reasons := []string{}
	penalties := []string{}

	if hasAny(text, "shenzhen", "深圳") {
		score += 25
		reasons = append(reasons, "Shenzhen role")
	} else if strings.TrimSpace(job.City) == "" {
		score -= 10
		penalties = append(penalties, "Unclear city")
	}

	tags := detectDirectionTags(text)
	if len(tags) > 0 {
		score += 20
		reasons = append(reasons, "Matches target direction")
	}
	if containsString(tags, "algorithm") || containsString(tags, "ai_application") {
		score += 10
		reasons = append(reasons, "High-priority algorithm or AI application role")
	}

	if hasAny(text, "tencent", "bytedance", "huawei", "alibaba", "baidu", "meituan", "kuaishou", "oppo", "vivo", "honor", "dji", "ai", "cloud", "fintech", "腾讯", "字节", "华为", "阿里", "百度", "美团", "快手", "大疆", "金融科技") {
		score += 15
		reasons = append(reasons, "Preferred company category")
	}

	if hasAny(text, "campus", "graduate", "2027", "校招", "校园招聘", "应届", "秋招") {
		score += 15
		reasons = append(reasons, "Clear campus or graduate signal")
	}

	if strings.TrimSpace(job.ApplyURL) != "" {
		score += 10
		reasons = append(reasons, "Clear application URL")
	}

	if len([]rune(strings.TrimSpace(job.Description))) >= 40 {
		score += 5
		reasons = append(reasons, "Detailed job description")
	} else if strings.TrimSpace(job.Description) == "" {
		score -= 10
		penalties = append(penalties, "Missing job description")
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	job.MatchScore = score
	job.DirectionTags = mergeStrings(job.DirectionTags, tags)
	job.RecommendReasons = mergeStrings(job.RecommendReasons, reasons)
	job.PenaltyReasons = mergeStrings(job.PenaltyReasons, penalties)
	if job.Status == "" {
		job.Status = domain.StatusNew
	}

	return ScoreResult{Job: job}
}

func IsHardFiltered(job domain.Job) (bool, string) {
	text := normalizedSearchText(job.Company, job.Title, job.City, job.Description)
	if hasAny(text, "outsourcing", "外包") {
		return true, "Suspected outsourcing"
	}
	if hasAny(text, "training", "course", "bootcamp", "培训", "课程") {
		return true, "Suspected training or course-sales content"
	}
	if hasAny(text, "intern", "实习") && hasAny(text, "转正不明", "转正未知", "unclear conversion") {
		return true, "Internship conversion is unclear"
	}
	if hasAny(text, "sales", "销售") && !hasAny(text, "software", "engineer", "developer", "算法", "开发", "工程师") {
		return true, "Non-technical sales role"
	}
	return false, ""
}

func detectDirectionTags(text string) []string {
	tags := []string{}
	if hasAny(text, "frontend", "front-end", "web", "react", "vue", "typescript", "前端") {
		tags = append(tags, "frontend")
	}
	if hasAny(text, "backend", "back-end", "server-side", "server side", "service", "后端", "服务端", "后台开发", "软件开发工程师") {
		tags = append(tags, "backend")
	}
	if hasAny(text, "java", "spring", "spring boot") {
		tags = append(tags, "java")
	}
	if hasAny(text, "golang", " go ", "go/", "go,", "云原生", "微服务") {
		tags = append(tags, "go")
	}
	if hasAny(text, "algorithm", "machine learning", "deep learning", "recommendation", "search", "nlp", "cv", "算法", "机器学习", "深度学习", "推荐", "搜索") {
		tags = append(tags, "algorithm")
	}
	if hasAny(text, "ai application", "llm", "agent", "rag", "aigc", "ai应用", "ai 应用", "大模型") {
		tags = append(tags, "ai_application")
	}
	return tags
}

func normalizedSearchText(values ...string) string {
	return " " + strings.ToLower(strings.Join(values, " ")) + " "
}

func hasAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func mergeStrings(existing []string, additions []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range append(existing, additions...) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
