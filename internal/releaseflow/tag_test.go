package releaseflow

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCreateTagCutsFirstPatchOnFreshLine(t *testing.T) {
	// The release-2.4 line has no tags yet, so the first tag is v2.4.0.
	clone, remote := createTagRepo(t, []string{"v2.3.5"}, []string{"release-2.4"})

	result := CreateTag(clone, TagOptions{}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("CreateTag() = %#v, want DONE", result)
	}
	if !strings.Contains(result.Message, "v2.4.0") {
		t.Fatalf("CreateTag() message = %q, want it to mention v2.4.0", result.Message)
	}
	if !remoteHasTag(t, remote, "v2.4.0") {
		t.Fatal("CreateTag() did not push v2.4.0 to origin")
	}
}

func TestCreateTagAdvancesPatch(t *testing.T) {
	// The line already has v2.4.0 and v2.4.1, so the next tag is v2.4.2.
	clone, remote := createTagRepo(t, []string{"v2.4.0", "v2.4.1"}, []string{"release-2.4"})

	result := CreateTag(clone, TagOptions{}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("CreateTag() = %#v, want DONE", result)
	}
	if !remoteHasTag(t, remote, "v2.4.2") {
		t.Fatal("CreateTag() did not push v2.4.2 to origin")
	}
}

func TestCreateTagIgnoresOtherLines(t *testing.T) {
	// A higher tag on a different line must not affect the 2.4 patch sequence.
	clone, remote := createTagRepo(t, []string{"v2.4.0", "v3.1.9"}, []string{"release-2.4"})

	result := CreateTag(clone, TagOptions{}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("CreateTag() = %#v, want DONE", result)
	}
	if !remoteHasTag(t, remote, "v2.4.1") {
		t.Fatal("CreateTag() did not cut v2.4.1 for the 2.4 line")
	}
}

func TestCreateTagPinsExplicitVersion(t *testing.T) {
	clone, remote := createTagRepo(t, nil, []string{"release-2.4"})

	result := CreateTag(clone, TagOptions{ExplicitVersion: "2.4.3"}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("CreateTag() = %#v, want DONE", result)
	}
	if !remoteHasTag(t, remote, "v2.4.3") {
		t.Fatal("CreateTag(--version 2.4.3) did not push v2.4.3")
	}
}

func TestCreateTagRejectsExplicitVersionOffLine(t *testing.T) {
	clone, _ := createTagRepo(t, nil, []string{"release-2.4"})

	result := CreateTag(clone, TagOptions{ExplicitVersion: "2.5.0"}, RealRunner{})
	if result.Action != ActionSkipped {
		t.Fatalf("CreateTag() = %#v, want SKIPPED for off-line version", result)
	}
	if !strings.Contains(result.Message, "release line") {
		t.Fatalf("CreateTag() message = %q, want off-line explanation", result.Message)
	}
}

func TestCreateTagSkipsWhenNoReleaseBranch(t *testing.T) {
	clone, _ := createTagRepo(t, []string{"v2.3.5"}, nil)

	result := CreateTag(clone, TagOptions{}, RealRunner{})
	if result.Action != ActionSkipped {
		t.Fatalf("CreateTag() = %#v, want SKIPPED without a release branch", result)
	}
}

func TestCreateTagHonorsTagFormat(t *testing.T) {
	clone, remote := createTagRepo(t, nil, []string{"release-2.4"})

	result := CreateTag(clone, TagOptions{TagFormat: "{major}.{minor}.{patch}"}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("CreateTag() = %#v, want DONE", result)
	}
	if !remoteHasTag(t, remote, "2.4.0") {
		t.Fatal("CreateTag() did not honor the v-less tag format")
	}
}

func TestPlanTagDoesNotMutate(t *testing.T) {
	clone, remote := createTagRepo(t, nil, []string{"release-2.4"})

	result := PlanTag(clone, TagOptions{})
	if result.Action != ActionPlanned {
		t.Fatalf("PlanTag() = %#v, want PLANNED", result)
	}
	if remoteHasTag(t, remote, "v2.4.0") {
		t.Fatal("PlanTag() created v2.4.0, want no mutation")
	}
}

// createTagRepo builds a bare origin with a develop branch, the given tags, and
// the given release branches, then returns a fresh clone plus the remote path.
func createTagRepo(t *testing.T, tags, branches []string) (clone string, remote string) {
	t.Helper()
	return createStartRepo(t, tags, branches)
}

func remoteHasTag(t *testing.T, remote, tag string) bool {
	t.Helper()
	cmd := exec.Command("git", "--git-dir", remote, "tag", "--list", tag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git tag --list in %q error = %v: %s", remote, err, string(output))
	}
	return strings.TrimSpace(string(output)) != ""
}
