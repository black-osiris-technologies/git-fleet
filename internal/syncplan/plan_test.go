package syncplan

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummaryAdd(t *testing.T) {
	summary := Summary{}
	summary.Add(RepoPlan{Action: ActionReady})
	summary.Add(RepoPlan{Action: ActionDone})
	summary.Add(RepoPlan{Action: ActionSkipped})
	summary.Add(RepoPlan{Action: ActionFailed})

	if summary.Total != 4 || summary.Ready != 1 || summary.Done != 1 || summary.Skipped != 1 || summary.Failed != 1 {
		t.Fatalf("summary = %#v, want one of each action", summary)
	}
}

func TestExecuteSyncsCleanRepository(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	clone := filepath.Join(root, "clone")

	initRemoteAndSeed(t, root, remote, seed)
	runGit(t, root, "clone", remote, clone)

	plan := Execute(clone, "develop")
	if plan.Action != ActionDone {
		t.Fatalf("Execute() = %#v, want DONE", plan)
	}
}

func TestExecuteSyncsLocalBranchWithoutUpstreamFromOrigin(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	clone := filepath.Join(root, "clone")

	initRemoteAndSeed(t, root, remote, seed)
	runGit(t, seed, "branch", "release-2.4", "develop")
	runGit(t, seed, "push", "origin", "release-2.4")
	runGit(t, root, "clone", remote, clone)
	// Create the local branch deliberately without tracking configuration.
	runGit(t, clone, "branch", "--no-track", "release-2.4", "origin/release-2.4")

	plan := Execute(clone, "latest-release")
	if plan.Action != ActionDone {
		t.Fatalf("Execute() = %#v, want DONE without relying on local upstream config", plan)
	}
	if got := runGitOutput(t, clone, "rev-parse", "HEAD"); got != runGitOutput(t, clone, "rev-parse", "origin/release-2.4") {
		t.Fatalf("HEAD = %s, want origin/release-2.4", got)
	}
}

func TestPlanLatestReleaseReadsLiveOrigin(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	clone := filepath.Join(root, "clone")

	initRemoteAndSeed(t, root, remote, seed)
	runGit(t, seed, "branch", "release-2.3", "develop")
	runGit(t, seed, "branch", "release-2.4", "develop")
	runGit(t, seed, "push", "origin", "release-2.3", "release-2.4")
	runGit(t, root, "clone", remote, clone)

	// Delete the higher release directly in the bare remote. The clone keeps its
	// stale origin/release-2.4 remote-tracking ref until a fetch occurs.
	runGit(t, root, "--git-dir", remote, "branch", "-D", "release-2.4")
	if got := runGitOutput(t, clone, "branch", "-r", "--list", "origin/release-2.4"); got == "" {
		t.Fatal("test setup lost stale origin/release-2.4 tracking ref")
	}

	plan := Plan(clone, "latest-release")
	if plan.Action != ActionReady {
		t.Fatalf("Plan() = %#v, want READY", plan)
	}
	if !strings.Contains(plan.Message, "release-2.3") || strings.Contains(plan.Message, "release-2.4") {
		t.Fatalf("Plan() message = %q, want live origin release-2.3", plan.Message)
	}
}

func TestExecutePreservesLocalTagDeletedFromOrigin(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	clone := filepath.Join(root, "clone")

	initRemoteAndSeed(t, root, remote, seed)
	runGit(t, seed, "tag", "v1.0.0")
	runGit(t, seed, "push", "origin", "v1.0.0")
	runGit(t, root, "clone", remote, clone)

	if got := runGitOutput(t, clone, "tag", "--list", "v1.0.0"); got != "v1.0.0" {
		t.Fatalf("tag before remote deletion = %q, want v1.0.0", got)
	}

	runGit(t, seed, "push", "origin", ":refs/tags/v1.0.0")

	plan := Execute(clone, "develop")
	if plan.Action != ActionDone {
		t.Fatalf("Execute() = %#v, want DONE", plan)
	}
	if got := runGitOutput(t, clone, "tag", "--list", "v1.0.0"); got != "v1.0.0" {
		t.Fatalf("tag after sync = %q, want preserved local tag v1.0.0", got)
	}
}

func initRemoteAndSeed(t *testing.T, root, remote, seed string) {
	t.Helper()
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
	runGit(t, seed, "remote", "add", "origin", remote)
	runGit(t, seed, "push", "-u", "origin", "develop")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %q error = %v: %s", args, dir, err, string(output))
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %q error = %v: %s", args, dir, err, string(output))
	}
	return strings.TrimSpace(string(output))
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}
