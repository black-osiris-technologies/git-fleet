package releaseflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/black-osiris-technologies/git-fleet/internal/semver"
)

func TestStartReleaseCreatesNextMinor(t *testing.T) {
	clone, remote := createStartRepo(t, []string{"v2.3.5"}, nil)

	result := StartRelease(clone, StartOptions{}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("StartRelease() = %#v, want DONE", result)
	}
	if !strings.Contains(result.Message, "release-2.4") {
		t.Fatalf("StartRelease() message = %q, want it to mention release-2.4", result.Message)
	}
	if !remoteHasBranch(t, remote, "release-2.4") {
		t.Fatal("StartRelease() did not push release-2.4 to origin")
	}
}

func TestStartReleaseMajorStartsNextMajor(t *testing.T) {
	clone, remote := createStartRepo(t, []string{"v2.3.5"}, nil)

	result := StartRelease(clone, StartOptions{Bump: semver.BumpMajor}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("StartRelease() = %#v, want DONE", result)
	}
	if !remoteHasBranch(t, remote, "release-3.0") {
		t.Fatal("StartRelease(major) did not push release-3.0 to origin")
	}
}

func TestStartReleaseIsIdempotentWhenBranchExists(t *testing.T) {
	// The natural target already exists on origin, so the run must skip rather
	// than advance to release-2.5 or recreate the branch.
	clone, _ := createStartRepo(t, []string{"v2.3.5"}, []string{"release-2.4"})

	result := StartRelease(clone, StartOptions{}, RealRunner{})
	if result.Action != ActionSkipped {
		t.Fatalf("StartRelease() = %#v, want SKIPPED", result)
	}
	if !strings.Contains(result.Message, "release-2.4 already exists") {
		t.Fatalf("StartRelease() message = %q, want existing-branch skip", result.Message)
	}
}

func TestStartReleaseSkipsWithoutTags(t *testing.T) {
	clone, _ := createStartRepo(t, nil, nil)

	result := StartRelease(clone, StartOptions{}, RealRunner{})
	if result.Action != ActionSkipped {
		t.Fatalf("StartRelease() = %#v, want SKIPPED", result)
	}
	if !strings.Contains(result.Message, "--version") {
		t.Fatalf("StartRelease() message = %q, want guidance to pass --version", result.Message)
	}
}

func TestStartReleaseUsesExplicitVersionWithoutTags(t *testing.T) {
	clone, remote := createStartRepo(t, nil, nil)

	result := StartRelease(clone, StartOptions{ExplicitVersion: "1.0"}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("StartRelease() = %#v, want DONE", result)
	}
	if !remoteHasBranch(t, remote, "release-1.0") {
		t.Fatal("StartRelease(--version 1.0) did not push release-1.0 to origin")
	}
}

func TestStartReleaseHonorsBranchFormat(t *testing.T) {
	clone, remote := createStartRepo(t, []string{"v2.3.5"}, nil)

	result := StartRelease(clone, StartOptions{BranchFormat: "release/{major}.{minor}.{patch}"}, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("StartRelease() = %#v, want DONE", result)
	}
	if !remoteHasBranch(t, remote, "release/2.4.0") {
		t.Fatal("StartRelease() did not honor the three-part branch format")
	}
}

func TestPlanStartDoesNotMutate(t *testing.T) {
	clone, remote := createStartRepo(t, []string{"v2.3.5"}, nil)

	result := PlanStart(clone, StartOptions{})
	if result.Action != ActionPlanned {
		t.Fatalf("PlanStart() = %#v, want PLANNED", result)
	}
	if remoteHasBranch(t, remote, "release-2.4") {
		t.Fatal("PlanStart() created release-2.4, want no mutation")
	}
}

// createStartRepo builds a bare origin with a develop branch, applies the given
// tags and extra branches, and returns a fresh clone plus the remote path.
func createStartRepo(t *testing.T, tags, branches []string) (clone string, remote string) {
	t.Helper()
	root := t.TempDir()
	remote = filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	clone = filepath.Join(root, "clone")

	runGit(t, root, "init", "--bare", remote)
	runGit(t, root, "--git-dir", remote, "symbolic-ref", "HEAD", "refs/heads/develop")

	mustMkdir(t, seed)
	runGit(t, seed, "init", "-b", "develop")
	runGit(t, seed, "config", "user.email", "test@example.com")
	runGit(t, seed, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runGit(t, seed, "add", "README.md")
	runGit(t, seed, "commit", "-m", "Initial commit")
	for _, tag := range tags {
		runGit(t, seed, "tag", tag)
	}
	for _, branch := range branches {
		runGit(t, seed, "branch", branch, "develop")
	}
	runGit(t, seed, "remote", "add", "origin", remote)
	runGit(t, seed, "push", "-u", "origin", "develop")
	if len(tags) > 0 {
		runGit(t, seed, "push", "origin", "--tags")
	}
	for _, branch := range branches {
		runGit(t, seed, "push", "origin", branch)
	}

	runGit(t, root, "clone", remote, clone)
	return clone, remote
}

func remoteHasBranch(t *testing.T, remote, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "--git-dir", remote, "branch", "--list", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list in %q error = %v: %s", remote, err, string(output))
	}
	return strings.Contains(string(output), branch)
}
