package jobs

import (
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestDedupeKeyNormalizesCompanyTitleAndCity(t *testing.T) {
	a := DedupeKey(domain.Job{
		Company: " Tencent ",
		Title:   "Backend   Engineer",
		City:    " Shenzhen ",
	})
	b := DedupeKey(domain.Job{
		Company: "tencent",
		Title:   "backend engineer",
		City:    "shenzhen",
	})

	if a != b {
		t.Fatalf("expected keys to match, got %q and %q", a, b)
	}
}

func TestDedupeKeyUsesApplyURLWhenAvailable(t *testing.T) {
	a := DedupeKey(domain.Job{
		Company:  "Tencent",
		Title:    "Backend Engineer",
		City:     "Shenzhen",
		ApplyURL: " HTTPS://Example.com/Apply/ ",
	})
	b := DedupeKey(domain.Job{
		Company:  "Other",
		Title:    "Other",
		City:     "Other",
		ApplyURL: "https://example.com/apply",
	})

	if a != b {
		t.Fatalf("expected URL keys to match, got %q and %q", a, b)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
