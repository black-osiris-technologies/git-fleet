package releaseflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummaryAdd(t *testing.T) {
	summary := Summary{}
	summary.Add(Result{Action: ActionPlanned})
	summary.Add(Result{Action: ActionDone})
	summary.Add(Result{Action: ActionSkipped})
	summary.Add(Result{Action: ActionFailed})

	if summary.Total != 4 || summary.Planned != 1 || summary.Done != 1 || summary.Skipped != 1 || summary.Failed != 1 {
		t.Fatalf("summary = %#v, want one of each action", summary)
	}
}

func TestMergePRUsesMergeCommit(t *testing.T) {
	repoPath := createReleaseRepo(t)
	runner := &fakeRunner{
		outputs: map[string]string{
			"git fetch --prune": "",
			"gh pr list --base master --head release-1.2 --state open --json number --jq .[0].number": "42",
			"gh pr merge 42 --merge": "Merged",
		},
	}

	result := MergePR(repoPath, "release-1.2", "master", runner)
	if result.Action != ActionDone {
		t.Fatalf("MergePR() = %#v, want DONE", result)
	}
	if runner.usedSquash {
		t.Fatal("MergePR() used squash, want merge commit only")
	}
	if !runner.sawMerge {
		t.Fatal("MergePR() did not call gh pr merge with --merge")
	}
}

func createReleaseRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	clone := filepath.Join(root, "clone")

	runGit(t, root, "init", "--bare", remote)
	mustMkdir(t, seed)
	runGit(t, seed, "init", "-b", "master")
	runGit(t, seed, "config", "user.email", "test@example.com")
	runGit(t, seed, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runGit(t, seed, "add", "README.md")
	runGit(t, seed, "commit", "-m", "Initial commit")
	runGit(t, seed, "checkout", "-b", "release-1.2")
	runGit(t, seed, "remote", "add", "origin", remote)
	runGit(t, seed, "push", "-u", "origin", "master")
	runGit(t, seed, "push", "-u", "origin", "release-1.2")
	runGit(t, root, "clone", remote, clone)
	return clone
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

type fakeRunner struct {
	outputs    map[string]string
	sawMerge   bool
	usedSquash bool
	sawDelete  bool
}

func (f *fakeRunner) Run(_ string, command string, args ...string) (string, error) {
	if command == "gh" && len(args) >= 3 && args[0] == "pr" && args[1] == "merge" {
		for _, arg := range args {
			if arg == "--merge" {
				f.sawMerge = true
			}
			if arg == "--squash" {
				f.usedSquash = true
			}
		}
	}
	if command == "git" && len(args) >= 2 && args[0] == "push" && contains(args, "--delete") {
		f.sawDelete = true
	}
	key := strings.TrimSpace(command + " " + strings.Join(args, " "))
	return f.outputs[key], nil
}
