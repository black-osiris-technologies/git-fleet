package releaseflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFinishReleaseMergesBothTargets(t *testing.T) {
	repoPath := createFinishRepo(t)
	runner := &fakeRunner{
		outputs: map[string]string{
			"git fetch --prune --tags": "",
			"gh pr list --base master --head release-2.4 --state open --json number --jq .[0].number":  "10",
			"gh pr list --base develop --head release-2.4 --state open --json number --jq .[0].number": "11",
			"gh pr merge 10 --merge": "Merged into master",
			"gh pr merge 11 --merge": "Merged into develop",
		},
	}

	result := FinishRelease(repoPath, FinishOptions{}, runner)
	if result.Action != ActionDone {
		t.Fatalf("FinishRelease() = %#v, want DONE", result)
	}
	if runner.usedSquash {
		t.Fatal("FinishRelease() used squash, want merge commits only")
	}
	if !strings.Contains(result.Message, "master") || !strings.Contains(result.Message, "develop") {
		t.Fatalf("FinishRelease() message = %q, want both targets mentioned", result.Message)
	}
	if runner.sawDelete {
		t.Fatal("FinishRelease() deleted the branch without --delete-branch")
	}
}

func TestFinishReleaseSkipsWhenNoOpenPRs(t *testing.T) {
	repoPath := createFinishRepo(t)
	// No pr list outputs configured, so every list returns empty: nothing to do.
	runner := &fakeRunner{outputs: map[string]string{"git fetch --prune --tags": ""}}

	result := FinishRelease(repoPath, FinishOptions{}, runner)
	if result.Action != ActionSkipped {
		t.Fatalf("FinishRelease() = %#v, want SKIPPED when no open PRs", result)
	}
	if runner.sawMerge {
		t.Fatal("FinishRelease() merged despite no open PRs")
	}
}

func TestFinishReleaseDeletesBranchWhenRequested(t *testing.T) {
	repoPath := createFinishRepo(t)
	runner := &fakeRunner{
		outputs: map[string]string{
			"git fetch --prune --tags": "",
			"gh pr list --base master --head release-2.4 --state open --json number --jq .[0].number":  "10",
			"gh pr list --base develop --head release-2.4 --state open --json number --jq .[0].number": "11",
			"gh pr merge 10 --merge":               "Merged",
			"gh pr merge 11 --merge":               "Merged",
			"git push origin --delete release-2.4": "",
		},
	}

	result := FinishRelease(repoPath, FinishOptions{DeleteBranch: true}, runner)
	if result.Action != ActionDone {
		t.Fatalf("FinishRelease() = %#v, want DONE", result)
	}
	if !runner.sawDelete {
		t.Fatal("FinishRelease(--delete-branch) did not delete the branch after both merges")
	}
}

func TestFinishReleaseDoesNotDeleteWhenOneTargetHasNoPR(t *testing.T) {
	repoPath := createFinishRepo(t)
	// Only master has an open PR; develop has none, so the branch must survive.
	runner := &fakeRunner{
		outputs: map[string]string{
			"git fetch --prune --tags": "",
			"gh pr list --base master --head release-2.4 --state open --json number --jq .[0].number": "10",
			"gh pr merge 10 --merge": "Merged",
		},
	}

	result := FinishRelease(repoPath, FinishOptions{DeleteBranch: true}, runner)
	if result.Action != ActionDone {
		t.Fatalf("FinishRelease() = %#v, want DONE (master merged)", result)
	}
	if runner.sawDelete {
		t.Fatal("FinishRelease() deleted the branch though develop was not merged")
	}
}

func TestPlanFinishDoesNotMutate(t *testing.T) {
	repoPath := createFinishRepo(t)

	result := PlanFinish(repoPath, FinishOptions{DeleteBranch: true})
	if result.Action != ActionPlanned {
		t.Fatalf("PlanFinish() = %#v, want PLANNED", result)
	}
	if !strings.Contains(result.Message, "master") || !strings.Contains(result.Message, "develop") {
		t.Fatalf("PlanFinish() message = %q, want both targets mentioned", result.Message)
	}
	if !strings.Contains(result.Message, "delete") {
		t.Fatalf("PlanFinish() message = %q, want delete noted", result.Message)
	}
}

// createFinishRepo builds a bare origin with master, develop, and a release-2.4
// branch, then returns a fresh clone whose worktree is clean.
func createFinishRepo(t *testing.T) string {
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
	runGit(t, seed, "branch", "develop")
	runGit(t, seed, "branch", "release-2.4")
	runGit(t, seed, "remote", "add", "origin", remote)
	runGit(t, seed, "push", "-u", "origin", "master")
	runGit(t, seed, "push", "origin", "develop")
	runGit(t, seed, "push", "origin", "release-2.4")
	runGit(t, root, "clone", remote, clone)
	return clone
}
