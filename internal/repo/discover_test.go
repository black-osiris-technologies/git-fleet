package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDiscoverFindsGitRepositories(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "nested", "repo-b")

	mkdirGit(t, repoA)
	mkdirGit(t, repoB)
	mustMkdir(t, filepath.Join(root, "not-a-repo"))

	repos, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(repos) != 2 {
		t.Fatalf("Discover() found %d repos, want 2: %#v", len(repos), repos)
	}
	if repos[0].Path != repoA && repos[1].Path != repoA {
		t.Fatalf("Discover() did not find repo %q: %#v", repoA, repos)
	}
	if repos[0].Path != repoB && repos[1].Path != repoB {
		t.Fatalf("Discover() did not find nested repo %q: %#v", repoB, repos)
	}
}

func mkdirGit(t *testing.T, path string) {
	t.Helper()
	mustMkdir(t, path)
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init in %q error = %v: %s", path, err, string(output))
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}
