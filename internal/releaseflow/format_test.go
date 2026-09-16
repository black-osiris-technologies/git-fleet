package releaseflow

import (
	"strings"
	"testing"
)

func TestPlanStartRejectsIncompatibleBranchFormat(t *testing.T) {
	clone, _ := createStartRepo(t, []string{"v2.3.5"}, nil)

	result := PlanStart(clone, StartOptions{BranchFormat: "stable-{major}.{minor}"})
	if result.Action != ActionFailed {
		t.Fatalf("PlanStart() = %#v, want FAILED for incompatible branch format", result)
	}
	if !strings.Contains(result.Message, "incompatible release branch") {
		t.Fatalf("PlanStart() message = %q, want compatibility guidance", result.Message)
	}
}

func TestPlanTagRejectsIncompatibleTagFormat(t *testing.T) {
	clone, _ := createTagRepo(t, nil, []string{"release-2.4"})

	result := PlanTag(clone, TagOptions{TagFormat: "release-{major}.{minor}.{patch}"})
	if result.Action != ActionFailed {
		t.Fatalf("PlanTag() = %#v, want FAILED for incompatible tag format", result)
	}
	if !strings.Contains(result.Message, "incompatible with release version discovery") {
		t.Fatalf("PlanTag() message = %q, want compatibility guidance", result.Message)
	}
}
