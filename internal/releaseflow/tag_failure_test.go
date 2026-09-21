package releaseflow

import "testing"

func TestPlanTagFailsWhenOriginCannotBeRead(t *testing.T) {
	clone, _ := createTagRepo(t, []string{"v2.4.0"}, []string{"release-2.4"})
	runGit(t, clone, "remote", "remove", "origin")

	result := PlanTag(clone, TagOptions{})
	if result.Action != ActionFailed {
		t.Fatalf("PlanTag() = %#v, want FAILED when origin cannot be queried", result)
	}
}
