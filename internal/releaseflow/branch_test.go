package releaseflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/black-osiris-technologies/git-fleet/internal/repo"
)

func TestNextFromReleaseBranchesUsesNextMinor(t *testing.T) {
	release, ok := nextFromReleaseBranches(repo.Branches{
		Local:  []string{"develop", "release-1.2"},
		Remote: []string{"origin/release-1.10", "origin/release-1.9"},
	})
	if !ok {
		t.Fatal("nextFromReleaseBranches() did not find a release branch")
	}
	if release.Branch != "release-1.11" {
		t.Fatalf("next branch = %q, want release-1.11", release.Branch)
	}
}

func TestNextFromTagsUsesLatestSemverTag(t *testing.T) {
	release, ok := nextFromTags([]string{"not-a-version", "1.2.0", "v1.10.3", "v1.9.9"})
	if !ok {
		t.Fatal("nextFromTags() did not find a semver tag")
	}
	if release.Branch != "release-1.11" {
		t.Fatalf("next branch = %q, want release-1.11", release.Branch)
	}
}

func TestCreateBranchFromLatestTag(t *testing.T) {
	fixture := createDevelopRepo(t, "v1.2.0")

	result := CreateBranch(fixture.clone, RealRunner{})
	if result.Action != ActionDone {
		t.Fatalf("CreateBranch() = %#v, want DONE", result)
	}

	branch, err := repo.GitOutput(fixture.clone, "branch", "--show-current")
	if err != nil {
		t.Fatalf("GitOutput() error = %v", err)
	}
	if branch != "release-1.3" {
		t.Fatalf("current branch = %q, want release-1.3", branch)
	}
}

func TestCreateBranchSkipsWhenLocalDevelopBehindOrigin(t *testing.T) {
	fixture := createDevelopRepo(t, "v1.2.0")

	if err := os.WriteFile(filepath.Join(fixture.seed, "CHANGELOG.md"), []byte("change\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runGit(t, fixture.seed, "add", "CHANGELOG.md")
	runGit(t, fixture.seed, "commit", "-m", "Advance develop")
	runGit(t, fixture.seed, "push", "origin", "develop")

	result := CreateBranch(fixture.clone, RealRunner{})
	if result.Action != ActionSkipped {
		t.Fatalf("CreateBranch() = %#v, want SKIPPED", result)
	}
	if result.Message != "local develop is not up to date with origin/develop" {
		t.Fatalf("message = %q, want local develop sync warning", result.Message)
	}
	if _, err := repo.GitOutput(fixture.clone, "show-ref", "--verify", "refs/heads/release-1.3"); err == nil {
		t.Fatal("release-1.3 exists, want no release branch created")
	}
}

type developFixture struct {
	seed  string
	clone string
}

func createDevelopRepo(t *testing.T, tag string) developFixture {
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
	runGit(t, seed, "checkout", "-b", "develop")
	runGit(t, seed, "tag", tag)
	runGit(t, seed, "remote", "add", "origin", remote)
	runGit(t, seed, "push", "-u", "origin", "master")
	runGit(t, seed, "push", "-u", "origin", "develop")
	runGit(t, seed, "push", "origin", tag)
	runGit(t, root, "clone", remote, clone)
	runGit(t, clone, "checkout", "-b", "develop", "origin/develop")
	return developFixture{seed: seed, clone: clone}
}
