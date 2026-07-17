package jobs

import (
	"regexp"
	"strings"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

var whitespace = regexp.MustCompile(`\s+`)

func DedupeKey(job domain.Job) string {
	if strings.TrimSpace(job.ApplyURL) != "" {
		return "url:" + normalizeKeyPart(job.ApplyURL)
	}
	parts := []string{
		normalizeKeyPart(job.Company),
		normalizeKeyPart(job.Title),
		normalizeKeyPart(job.City),
	}
	return "job:" + strings.Join(parts, "|")
}

func normalizeKeyPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimRight(value, "/")
	value = whitespace.ReplaceAllString(value, " ")
	return value
}
