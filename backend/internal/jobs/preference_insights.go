package jobs

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type AgentPreferenceInsights struct {
	GeneratedAt          time.Time                  `json:"generated_at"`
	Summary              string                     `json:"summary"`
	TotalDecisions       int                        `json:"total_decisions"`
	InterestedCompanies  []PreferenceSignal         `json:"interested_companies"`
	IgnoredCompanies     []PreferenceSignal         `json:"ignored_companies"`
	InterestedDirections []PreferenceSignal         `json:"interested_directions"`
	IgnoredDirections    []PreferenceSignal         `json:"ignored_directions"`
	Recommendations      []JobRecommendationInsight `json:"recommended_jobs"`
}

type PreferenceSignal struct {
	Name   string `json:"label"`
	Count  int    `json:"count"`
	Weight int    `json:"weight"`
	Reason string `json:"evidence"`
}

type JobRecommendationInsight struct {
	JobID    int64    `json:"job_id"`
	Company  string   `json:"company"`
	Title    string   `json:"title"`
	City     string   `json:"city"`
	Score    int      `json:"score"`
	Status   string   `json:"status"`
	Reasons  []string `json:"reasons"`
	Warnings []string `json:"warnings"`
}

func (r *Repository) BuildAgentPreferenceInsights(ctx context.Context, now time.Time) (AgentPreferenceInsights, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT j.company, j.direction_tags, d.to_status
		FROM job_decisions d
		JOIN jobs j ON j.id = d.job_id
		WHERE d.to_status IN (?, ?, ?)
		ORDER BY d.created_at DESC, d.id DESC
		LIMIT 300
	`, string(domain.StatusInterested), string(domain.StatusApplied), string(domain.StatusIgnored))
	if err != nil {
		return AgentPreferenceInsights{}, fmt.Errorf("query preference decisions: %w", err)
	}
	defer rows.Close()

	positiveCompanies := map[string]int{}
	ignoredCompanies := map[string]int{}
	positiveDirections := map[string]int{}
	ignoredDirections := map[string]int{}
	total := 0
	for rows.Next() {
		var company string
		var tagsJSON string
		var toStatus string
		if err := rows.Scan(&company, &tagsJSON, &toStatus); err != nil {
			return AgentPreferenceInsights{}, fmt.Errorf("scan preference decision: %w", err)
		}
		total++
		tags := unmarshalStrings(tagsJSON)
		switch toStatus {
		case string(domain.StatusInterested), string(domain.StatusApplied):
			incrementSignal(positiveCompanies, company)
			for _, tag := range tags {
				incrementSignal(positiveDirections, tag)
			}
		case string(domain.StatusIgnored):
			incrementSignal(ignoredCompanies, company)
			for _, tag := range tags {
				incrementSignal(ignoredDirections, tag)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return AgentPreferenceInsights{}, fmt.Errorf("iterate preference decisions: %w", err)
	}

	insights := AgentPreferenceInsights{
		GeneratedAt:          now.UTC(),
		TotalDecisions:       total,
		InterestedCompanies:  preferenceSignals(positiveCompanies, 8, "You marked similar companies as worth attention."),
		IgnoredCompanies:     preferenceSignals(ignoredCompanies, -10, "You ignored similar companies before."),
		InterestedDirections: preferenceSignals(positiveDirections, 6, "You marked this direction as interesting before."),
		IgnoredDirections:    preferenceSignals(ignoredDirections, -8, "You ignored this direction before."),
	}
	jobs, err := r.ListJobs(ctx, ListFilter{})
	if err != nil {
		return AgentPreferenceInsights{}, err
	}
	insights.Recommendations = buildRecommendationInsights(jobs, insights)
	insights.Summary = buildPreferenceInsightSummary(insights)
	return insights, nil
}

func incrementSignal(values map[string]int, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	values[name]++
}

func preferenceSignals(values map[string]int, weight int, reason string) []PreferenceSignal {
	signals := make([]PreferenceSignal, 0, len(values))
	for name, count := range values {
		signals = append(signals, PreferenceSignal{Name: name, Count: count, Weight: weight, Reason: reason})
	}
	sort.SliceStable(signals, func(i, j int) bool {
		if signals[i].Count == signals[j].Count {
			return signals[i].Name < signals[j].Name
		}
		return signals[i].Count > signals[j].Count
	})
	if len(signals) > 8 {
		return signals[:8]
	}
	return signals
}

func buildRecommendationInsights(jobs []domain.Job, insights AgentPreferenceInsights) []JobRecommendationInsight {
	out := []JobRecommendationInsight{}
	for _, job := range jobs {
		if job.Status == domain.StatusIgnored {
			continue
		}
		reasons := mergeStrings(job.RecommendReasons, learnedJobReasons(job, insights))
		warnings := mergeStrings(job.PenaltyReasons, learnedJobWarnings(job, insights))
		out = append(out, JobRecommendationInsight{
			JobID:    job.ID,
			Company:  job.Company,
			Title:    job.Title,
			City:     job.City,
			Score:    job.MatchScore,
			Status:   string(job.Status),
			Reasons:  reasons,
			Warnings: warnings,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	if len(out) > 5 {
		return out[:5]
	}
	return out
}

func learnedJobReasons(job domain.Job, insights AgentPreferenceInsights) []string {
	reasons := []string{}
	if hasSignal(insights.InterestedCompanies, job.Company) {
		reasons = append(reasons, "Matches a company pattern you marked Interested or Applied.")
	}
	for _, tag := range job.DirectionTags {
		if hasSignal(insights.InterestedDirections, tag) {
			reasons = append(reasons, "Matches a direction you previously liked: "+tag)
		}
	}
	return reasons
}

func learnedJobWarnings(job domain.Job, insights AgentPreferenceInsights) []string {
	warnings := []string{}
	if hasSignal(insights.IgnoredCompanies, job.Company) {
		warnings = append(warnings, "Similar company pattern was ignored before.")
	}
	for _, tag := range job.DirectionTags {
		if hasSignal(insights.IgnoredDirections, tag) {
			warnings = append(warnings, "Similar direction was ignored before: "+tag)
		}
	}
	return warnings
}

func hasSignal(signals []PreferenceSignal, name string) bool {
	for _, signal := range signals {
		if strings.EqualFold(signal.Name, strings.TrimSpace(name)) {
			return true
		}
	}
	return false
}

func buildPreferenceInsightSummary(insights AgentPreferenceInsights) string {
	if insights.TotalDecisions == 0 {
		return "No job decisions yet. Mark jobs as Interested, Applied, or Ignore to teach the agent your preferences."
	}
	parts := []string{fmt.Sprintf("Learned from %d job decisions.", insights.TotalDecisions)}
	if len(insights.InterestedCompanies) > 0 {
		parts = append(parts, "Top positive company: "+insights.InterestedCompanies[0].Name)
	}
	if len(insights.InterestedDirections) > 0 {
		parts = append(parts, "Top positive direction: "+insights.InterestedDirections[0].Name)
	}
	if len(insights.IgnoredDirections) > 0 {
		parts = append(parts, "Watch ignored direction: "+insights.IgnoredDirections[0].Name)
	}
	return strings.Join(parts, " ")
}
