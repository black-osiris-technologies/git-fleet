package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestListOriginRefsReadRemoteWithoutMutatingLocalRefs(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	clone := filepath.Join(root, "clone")

	runRepoGit(t, root, "init", "--bare", remote)
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	runRepoGit(t, seed, "init", "-b", "develop")
	runRepoGit(t, seed, "config", "user.email", "test@example.com")
	runRepoGit(t, seed, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runRepoGit(t, seed, "add", "README.md")
	runRepoGit(t, seed, "commit", "-m", "initial")
	runRepoGit(t, seed, "branch", "release-2.4")
	runRepoGit(t, seed, "tag", "v2.3.5")
	runRepoGit(t, seed, "remote", "add", "origin", remote)
	runRepoGit(t, seed, "push", "origin", "develop", "release-2.4", "--tags")
	runRepoGit(t, root, "clone", remote, clone)

	branches, err := ListOriginBranches(clone)
	if err != nil {
		t.Fatalf("ListOriginBranches() error = %v", err)
	}
	if !containsRepoValue(branches, "develop") || !containsRepoValue(branches, "release-2.4") {
		t.Fatalf("ListOriginBranches() = %#v, want develop and release-2.4", branches)
	}

	tags, err := ListOriginTags(clone)
	if err != nil {
		t.Fatalf("ListOriginTags() error = %v", err)
	}
	if !containsRepoValue(tags, "v2.3.5") {
		t.Fatalf("ListOriginTags() = %#v, want v2.3.5", tags)
	}
}

func runRepoGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %q error = %v: %s", args, dir, err, string(output))
	}
}

func containsRepoValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
