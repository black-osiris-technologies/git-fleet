package syncplan

import (
	"os"
	"os/exec"
	"path/filepath"
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

	runGit(t, root, "clone", remote, clone)

	plan := Execute(clone, "develop")
	if plan.Action != ActionDone {
		t.Fatalf("Execute() = %#v, want DONE", plan)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %q error = %v: %s", args, dir, err, string(output))
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}
