package jobs

import (
	"context"
	"fmt"
)

type SourceOperationsSummary struct {
	TotalSources        int                  `json:"total_sources"`
	EnabledSources      int                  `json:"enabled_sources"`
	HealthySources      int                  `json:"healthy_sources"`
	WarningSources      int                  `json:"warning_sources"`
	BrokenSources       int                  `json:"broken_sources"`
	UnknownSources      int                  `json:"unknown_sources"`
	PendingCandidates   int                  `json:"pending_candidates"`
	VerifiedCandidates  int                  `json:"verified_candidates"`
	RejectedCandidates  int                  `json:"rejected_candidates"`
	NeedsAttention      []SourceAttention    `json:"needs_attention"`
	RecommendedPromotes []SourceCandidate    `json:"recommended_promotes"`
	Actions             []AgentCommandAction `json:"actions"`
}

type SourceAttention struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (r *Repository) BuildSourceOperationsSummary(ctx context.Context) (SourceOperationsSummary, error) {
	sources, err := r.ListSources(ctx, false)
	if err != nil {
		return SourceOperationsSummary{}, err
	}
	candidates, err := r.ListSourceCandidates(ctx, SourceCandidateFilter{})
	if err != nil {
		return SourceOperationsSummary{}, err
	}
	summary := SourceOperationsSummary{}
	for _, source := range sources {
		summary.TotalSources++
		if source.Enabled {
			summary.EnabledSources++
		}
		switch source.HealthStatus {
		case SourceHealthHealthy:
			summary.HealthySources++
		case SourceHealthWarning:
			summary.WarningSources++
			summary.NeedsAttention = append(summary.NeedsAttention, sourceAttention(source))
		case SourceHealthBroken:
			summary.BrokenSources++
			summary.NeedsAttention = append(summary.NeedsAttention, sourceAttention(source))
		default:
			summary.UnknownSources++
		}
	}
	for _, candidate := range candidates {
		switch candidate.Status {
		case SourceCandidateStatusRejected:
			summary.RejectedCandidates++
		case SourceCandidateStatusPending:
			summary.PendingCandidates++
			if candidate.ValidationStatus == SourceCandidateValidationGood || candidate.Confidence >= 70 {
				summary.RecommendedPromotes = append(summary.RecommendedPromotes, candidate)
			}
		}
		if candidate.ValidationStatus == SourceCandidateValidationGood {
			summary.VerifiedCandidates++
		}
	}
	if summary.BrokenSources > 0 || summary.WarningSources > 0 {
		summary.Actions = append(summary.Actions, AgentCommandAction{
			Type:   "inspect_failed_sources",
			Target: "sources",
			Detail: fmt.Sprintf("Inspect %d unhealthy sources.", summary.BrokenSources+summary.WarningSources),
		})
	}
	if summary.PendingCandidates > 0 {
		summary.Actions = append(summary.Actions, AgentCommandAction{
			Type:   "discover_sources",
			Target: "sources",
			Detail: fmt.Sprintf("Review %d pending source candidates.", summary.PendingCandidates),
		})
	}
	if summary.TotalSources == 0 {
		summary.Actions = append(summary.Actions, AgentCommandAction{
			Type:   "add_recommended_and_crawl",
			Target: "sources",
			Detail: "Seed recommended sources and run the first crawl.",
		})
	}
	return summary, nil
}

func sourceAttention(source Source) SourceAttention {
	return SourceAttention{
		ID:     source.ID,
		Name:   source.Name,
		URL:    source.URL,
		Status: source.HealthStatus,
		Reason: source.HealthReason,
	}
}
