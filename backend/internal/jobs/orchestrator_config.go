//go:build !eino

package jobs

import (
	"context"
	"strings"
)

func NewConfiguredRecruitingOrchestrator(_ context.Context, mode string) RecruitingOrchestrator {
	switch strings.TrimSpace(mode) {
	default:
		return DefaultRecruitingOrchestrator()
	}
}
