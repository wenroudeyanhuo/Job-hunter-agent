package config

import "testing"

func TestParseSourceURLs(t *testing.T) {
	got := parseSourceURLs("https://a.example/jobs, https://b.example\nhttps://a.example/jobs")
	want := []string{"https://a.example/jobs", "https://b.example"}
	if len(got) != len(want) {
		t.Fatalf("expected %d URLs, got %#v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("at %d expected %q, got %q", i, want[i], got[i])
		}
	}
}
