package jobs

import (
	"strings"
	"testing"
	"time"
)

func TestBuildObserverReplanActionsSuggestsSourceDiscoveryWhenCrawlCreatesNoJobs(t *testing.T) {
	request := AgentActionRequest{ActionType: "run_crawl", Target: "sources", Detail: "Run crawl"}

	actions := BuildObserverReplanActions(request, AgentActionExecutionSucceeded, "Crawl completed: found 0 jobs, created 0 jobs.", time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC))

	if len(actions) == 0 {
		t.Fatal("expected observer to suggest follow-up actions")
	}
	if !containsAgentAction(actions, "discover_sources") || !containsAgentAction(actions, "validate_source_candidates") {
		t.Fatalf("expected source discovery and validation after empty crawl, got %#v", actions)
	}
}

func TestBuildAgentToolObserverCycleIncludesReplanActions(t *testing.T) {
	cycle := BuildAgentToolObserverCycle(
		AgentActionRequest{ActionType: "run_crawl", Target: "sources", Detail: "Run crawl"},
		AgentActionExecutionSucceeded,
		"Crawl completed: found 18 jobs, created 12 jobs.",
		time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC),
	)

	if !cycle.AutonomyPlan.ReplanAfterExecution {
		t.Fatalf("expected replan after execution, got %#v", cycle.AutonomyPlan)
	}
	if !containsAgentAction(cycle.Actions, "review_strong_matches") && !strings.Contains(cycle.AutonomyPlan.Summary, "re-plan") {
		t.Fatalf("expected observer re-plan signal in cycle, got %#v", cycle)
	}
}

func containsAgentAction(actions []AgentCommandAction, actionType string) bool {
	for _, action := range actions {
		if action.Type == actionType {
			return true
		}
	}
	return false
}
