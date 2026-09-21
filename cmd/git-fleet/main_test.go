package main

import "testing"

func TestRenderActionReportReturnsErrorWhenAnyRepositoryFails(t *testing.T) {
	lines := []repoLine{
		{Repo: "a", Action: "DONE", Message: "ok"},
		{Repo: "b", Action: "FAILED", Message: "boom"},
		{Repo: "c", Action: "SKIPPED", Message: "not applicable"},
	}

	if err := renderActionReport(false, lines, []string{"done", "skipped", "failed"}); err == nil {
		t.Fatal("renderActionReport() error = nil, want non-nil when a repository failed")
	}
}

func TestRenderActionReportKeepsSkippedAsSuccessfulProcessOutcome(t *testing.T) {
	lines := []repoLine{
		{Repo: "a", Action: "DONE", Message: "ok"},
		{Repo: "b", Action: "SKIPPED", Message: "not applicable"},
	}

	if err := renderActionReport(false, lines, []string{"done", "skipped", "failed"}); err != nil {
		t.Fatalf("renderActionReport() error = %v, want nil when no repository failed", err)
	}
}
