package releaseflow

import (
	"errors"
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

func TestCreatePRAutomaticReleaseDoesNotPushLocalCopy(t *testing.T) {
	repoPath := createReleaseRepo(t)
	runGit(t, repoPath, "branch", "release-1.2", "origin/release-1.2")

	runner := &fakeRunner{outputs: map[string]string{
		"git fetch --prune": "",
	}}
	result := CreatePR(repoPath, "latest-release", "master", runner)
	if result.Action != ActionDone {
		t.Fatalf("CreatePR() = %#v, want DONE", result)
	}
	if runner.sawPush {
		t.Fatalf("CreatePR(latest-release) pushed a local copy of an existing origin release: %s", runner.pushArgs)
	}
}

func TestCreatePRExplicitOriginBranchDoesNotPushLocalCopy(t *testing.T) {
	repoPath := createReleaseRepo(t)
	runGit(t, repoPath, "branch", "release-1.2", "origin/release-1.2")

	runner := &fakeRunner{outputs: map[string]string{
		"git fetch --prune": "",
	}}
	result := CreatePR(repoPath, "release-1.2", "master", runner)
	if result.Action != ActionDone {
		t.Fatalf("CreatePR() = %#v, want DONE", result)
	}
	if runner.sawPush {
		t.Fatalf("CreatePR(explicit origin branch) pushed local state unexpectedly: %s", runner.pushArgs)
	}
}

func TestCreatePRPublishesExplicitLocalOnlyBranchCreateOnly(t *testing.T) {
	repoPath := createReleaseRepo(t)
	runGit(t, repoPath, "branch", "release-1.3", "master")

	runner := &fakeRunner{outputs: map[string]string{
		"git fetch --prune": "",
	}}
	result := CreatePR(repoPath, "release-1.3", "master", runner)
	if result.Action != ActionDone {
		t.Fatalf("CreatePR() = %#v, want DONE", result)
	}
	if !runner.sawPush {
		t.Fatal("CreatePR(local-only source) did not publish the branch")
	}
	if !strings.Contains(runner.pushArgs, "--force-with-lease=refs/heads/release-1.3:") {
		t.Fatalf("CreatePR() push = %q, want create-only lease", runner.pushArgs)
	}
	if !strings.Contains(runner.pushArgs, "release-1.3:refs/heads/release-1.3") {
		t.Fatalf("CreatePR() push = %q, want explicit release refspec", runner.pushArgs)
	}
}

func TestPlanPRLatestReleaseReadsLiveOrigin(t *testing.T) {
	repoPath := createReleaseRepo(t)
	remote := strings.TrimSpace(testGitOutput(t, repoPath, "remote", "get-url", "origin"))

	// Create a higher remote branch, fetch it so the clone has a remote-tracking
	// ref, then delete it directly in the bare remote. The local tracking ref is
	// intentionally left stale.
	runGit(t, repoPath, "--git-dir", remote, "branch", "release-1.3", "release-1.2")
	runGit(t, repoPath, "fetch", "origin")
	runGit(t, repoPath, "--git-dir", remote, "branch", "-D", "release-1.3")

	result := PlanPR(repoPath, "latest-release", "master")
	if result.Action != ActionPlanned {
		t.Fatalf("PlanPR() = %#v, want PLANNED", result)
	}
	if !strings.Contains(result.Message, "release-1.2 -> master") || strings.Contains(result.Message, "release-1.3") {
		t.Fatalf("PlanPR() message = %q, want live origin release-1.2", result.Message)
	}
}

func TestPlanPRTargetValidationReadsLiveOrigin(t *testing.T) {
	repoPath := createReleaseRepo(t)
	remote := strings.TrimSpace(testGitOutput(t, repoPath, "remote", "get-url", "origin"))

	// Remove master directly on origin while leaving origin/master stale locally.
	runGit(t, repoPath, "--git-dir", remote, "branch", "-D", "master")

	result := PlanPR(repoPath, "latest-release", "master")
	if result.Action != ActionSkipped {
		t.Fatalf("PlanPR() = %#v, want SKIPPED for target deleted on origin", result)
	}
	if !strings.Contains(result.Message, "not found on origin") {
		t.Fatalf("PlanPR() message = %q, want missing-origin target explanation", result.Message)
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
	errors     map[string]error
	sawMerge   bool
	usedSquash bool
	sawDelete  bool
	sawPush    bool
	pushArgs   string
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
	if command == "git" && len(args) >= 1 && args[0] == "push" {
		f.sawPush = true
		f.pushArgs = strings.Join(args, " ")
		for _, arg := range args {
			if arg == "--delete" || strings.HasPrefix(arg, ":refs/heads/") {
				f.sawDelete = true
			}
		}
	}
	key := strings.TrimSpace(command + " " + strings.Join(args, " "))
	if err := f.errors[key]; err != nil {
		return "", err
	}
	return f.outputs[key], nil
}

func runnerError(message string) error {
	return errors.New(message)
}
