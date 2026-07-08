package branch

import (
	"testing"

	"github.com/black-osiris-technologies/omp-git-fleet/internal/repo"
)

func TestResolveExplicitLocalBranch(t *testing.T) {
	resolution, err := Resolve("develop", repo.Branches{
		Local: []string{"develop"},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Source != "local" || resolution.LocalBranch != "develop" {
		t.Fatalf("Resolve() = %#v, want local develop", resolution)
	}
}

func TestResolveExplicitRemoteBranch(t *testing.T) {
	resolution, err := Resolve("develop", repo.Branches{
		Remote: []string{"origin/develop"},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Source != "remote" || resolution.RemoteRef != "origin/develop" {
		t.Fatalf("Resolve() = %#v, want remote origin/develop", resolution)
	}
}

func TestResolveLatestReleaseUsesHighestSemanticVersion(t *testing.T) {
	resolution, err := Resolve("latest-release", repo.Branches{
		Local:  []string{"release-1.2"},
		Remote: []string{"origin/release-1.10", "origin/release-1.9"},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Target != "release-1.10" || resolution.RemoteRef != "origin/release-1.10" {
		t.Fatalf("Resolve() = %#v, want release-1.10", resolution)
	}
}

func TestResolveMissingBranch(t *testing.T) {
	_, err := Resolve("develop", repo.Branches{
		Local:  []string{"main"},
		Remote: []string{"origin/main"},
	})
	if err == nil {
		t.Fatal("Resolve() error = nil, want missing branch error")
	}
}
