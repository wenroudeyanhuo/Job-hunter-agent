//go:build eino

package jobs

import (
	"context"
	"strings"
)

func NewConfiguredRecruitingOrchestrator(ctx context.Context, mode string) RecruitingOrchestrator {
	switch strings.TrimSpace(mode) {
	case "eino_graph", "eino":
		orchestrator, err := NewEinoRecruitingOrchestrator(ctx)
		if err == nil {
			return orchestrator
		}
	}
	return DefaultRecruitingOrchestrator()
}
