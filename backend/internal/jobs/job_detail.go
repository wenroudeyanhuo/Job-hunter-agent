package jobs

import (
	"strings"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type JobDetail struct {
	Job             domain.Job        `json:"job"`
	Fit             JobFitSummary     `json:"fit"`
	Decisions       []JobDecision     `json:"decisions"`
	SuggestedAction AgentReportAction `json:"suggested_action"`
}

type JobFitSummary struct {
	Score          int      `json:"score"`
	Verdict        string   `json:"verdict"`
	Strengths      []string `json:"strengths"`
	Risks          []string `json:"risks"`
	ProfileSignals []string `json:"profile_signals"`
}

func BuildJobDetail(job domain.Job, profile CandidateProfile, decisions []JobDecision) JobDetail {
	profile = normalizeCandidateProfile(profile)
	fit := JobFitSummary{
		Score:     job.MatchScore,
		Strengths: append([]string{}, job.RecommendReasons...),
		Risks:     append([]string{}, job.PenaltyReasons...),
	}
	text := normalizedSearchText(job.Company, job.Title, job.City, job.Description, strings.Join(job.DirectionTags, " "))
	if city, ok := matchedSettingValue(text, profile.TargetCities); ok {
		fit.Score += 6
		fit.ProfileSignals = append(fit.ProfileSignals, "Matches profile city: "+city)
	}
	if company, ok := matchedSettingValue(text, profile.PreferredCompanies); ok {
		fit.Score += 6
		fit.ProfileSignals = append(fit.ProfileSignals, "Preferred company: "+company)
	}
	matchedDirections := intersectStrings(job.DirectionTags, profile.TargetDirections)
	if len(matchedDirections) > 0 {
		fit.Score += 6
		fit.ProfileSignals = append(fit.ProfileSignals, "Matches profile directions: "+strings.Join(matchedDirections, ", "))
	}
	if skill, ok := matchedSettingValue(text, profile.Skills); ok {
		fit.Score += 6
		fit.ProfileSignals = append(fit.ProfileSignals, "Skill signal: "+skill)
	}
	for _, keyword := range cleanStringList(profile.BlockedKeywords) {
		if hasAny(text, keyword) {
			fit.Score -= 20
			fit.Risks = append(fit.Risks, "Matches blocked keyword: "+keyword)
		}
	}
	if fit.Score > 100 {
		fit.Score = 100
	}
	if fit.Score < 0 {
		fit.Score = 0
	}
	if len(fit.Strengths) == 0 && len(fit.ProfileSignals) == 0 {
		fit.Strengths = append(fit.Strengths, "No strong fit signal yet")
	}
	fit.Verdict = jobFitVerdict(fit.Score)
	return JobDetail{
		Job:       job,
		Fit:       fit,
		Decisions: decisions,
		SuggestedAction: AgentReportAction{
			Action: jobDetailSuggestedAction(job),
			Label:  jobDetailSuggestedLabel(job),
			Reason: jobDetailSuggestedReason(job, fit),
		},
	}
}

func jobFitVerdict(score int) string {
	switch {
	case score >= 85:
		return "strong_fit"
	case score >= 65:
		return "worth_reviewing"
	case score >= 45:
		return "manual_check"
	default:
		return "low_priority"
	}
}

func jobDetailSuggestedAction(job domain.Job) string {
	switch job.Status {
	case domain.StatusNew, domain.StatusManualCheck:
		return "mark_interested"
	case domain.StatusInterested:
		return "prepare_application"
	case domain.StatusApplied:
		return "follow_up"
	default:
		return "review"
	}
}

func jobDetailSuggestedLabel(job domain.Job) string {
	switch job.Status {
	case domain.StatusNew, domain.StatusManualCheck:
		return "Mark interested"
	case domain.StatusInterested:
		return "Prepare application"
	case domain.StatusApplied:
		return "Follow up"
	default:
		return "Review"
	}
}

func jobDetailSuggestedReason(job domain.Job, fit JobFitSummary) string {
	if job.Status == domain.StatusApplied {
		return "This role has already been applied to; keep the follow-up warm."
	}
	if fit.Score >= 85 {
		return "Profile signals are strong enough to prioritize this role."
	}
	if fit.Score >= 65 {
		return "This role is worth a manual decision before it gets stale."
	}
	return "Fit is not strong yet; review only if the role has hidden value."
}
