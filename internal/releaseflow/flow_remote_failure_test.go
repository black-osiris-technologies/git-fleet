package releaseflow

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanPRAutomaticOriginFailureIsFailed(t *testing.T) {
	repoPath := createReleaseRepo(t)
	missing := filepath.Join(t.TempDir(), "missing-origin.git")
	runGit(t, repoPath, "remote", "set-url", "origin", missing)

	result := PlanPR(repoPath, "latest-release", "master")
	if result.Action != ActionFailed {
		t.Fatalf("PlanPR() = %#v, want FAILED when origin cannot be queried", result)
	}
}

func TestPlanMergeRejectsExplicitLocalOnlyRelease(t *testing.T) {
	repoPath := createReleaseRepo(t)
	runGit(t, repoPath, "branch", "release-1.3", "master")

	result := PlanMerge(repoPath, "release-1.3", "master")
	if result.Action != ActionSkipped {
		t.Fatalf("PlanMerge() = %#v, want SKIPPED for local-only release", result)
	}
	if !strings.Contains(result.Message, "not found locally or on origin") {
		t.Fatalf("PlanMerge() message = %q, want missing remote release explanation", result.Message)
	}
}
