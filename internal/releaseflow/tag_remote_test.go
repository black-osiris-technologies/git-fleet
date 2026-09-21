package releaseflow

import (
	"strings"
	"testing"
)

func TestPlanTagIgnoresTagDeletedFromOriginButStillLocal(t *testing.T) {
	clone, _ := createTagRepo(t, []string{"v2.4.0", "v2.4.1"}, []string{"release-2.4"})
	runGit(t, clone, "push", "origin", ":refs/tags/v2.4.1")

	if !localHasTag(t, clone, "v2.4.1") {
		t.Fatal("test setup lost stale local tag v2.4.1")
	}

	result := PlanTag(clone, TagOptions{})
	if result.Action != ActionPlanned {
		t.Fatalf("PlanTag() = %#v, want PLANNED", result)
	}
	if !strings.Contains(result.Message, "v2.4.1") || strings.Contains(result.Message, "v2.4.2") {
		t.Fatalf("PlanTag() message = %q, want origin tag v2.4.0 to yield v2.4.1", result.Message)
	}
}

func TestCreateTagPreservesUnrelatedLocalTagsDeletedFromOrigin(t *testing.T) {
	clone, remote := createTagRepo(t, []string{"v2.4.0", "v9.9.9"}, []string{"release-2.4"})
	runGit(t, clone, "push", "origin", ":refs/tags/v9.9.9")

	if !localHasTag(t, clone, "v9.9.9") {
		t.Fatal("test setup lost stale local tag v9.9.9")
	}

	result := CreateTag(clone, TagOptions{}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("CreateTag() = %#v, want DONE", result)
	}
	if !remoteHasTag(t, remote, "v2.4.1") {
		t.Fatal("CreateTag() did not create v2.4.1 from origin tag state")
	}
	if !localHasTag(t, clone, "v9.9.9") {
		t.Fatal("CreateTag() removed unrelated local tag v9.9.9")
	}
}

func TestPlanTagIgnoresLocalOnlyHigherReleaseBranch(t *testing.T) {
	clone, _ := createTagRepo(t, []string{"v2.4.0"}, []string{"release-2.4"})
	runGit(t, clone, "branch", "release-9.9", "origin/develop")

	result := PlanTag(clone, TagOptions{})
	if result.Action != ActionPlanned {
		t.Fatalf("PlanTag() = %#v, want PLANNED", result)
	}
	if !strings.Contains(result.Message, "origin/release-2.4") {
		t.Fatalf("PlanTag() message = %q, want active origin/release-2.4", result.Message)
	}
}
