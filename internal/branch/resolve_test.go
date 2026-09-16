package branch

import (
	"testing"

	"github.com/black-osiris-technologies/git-fleet/internal/repo"
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

func TestResolveLatestReleaseUsesHighestActiveRemoteVersion(t *testing.T) {
	resolution, err := Resolve("latest-release", repo.Branches{
		Local:  []string{"release-9.9", "release-1.10"},
		Remote: []string{"origin/release-1.10", "origin/release-1.9"},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Target != "release-1.10" || resolution.LocalBranch != "release-1.10" || resolution.Source != "local" {
		t.Fatalf("Resolve() = %#v, want active release-1.10 using existing local branch", resolution)
	}
	if resolution.RemoteRef != "origin/release-1.10" {
		t.Fatalf("Resolve() RemoteRef = %q, want origin/release-1.10", resolution.RemoteRef)
	}
}

func TestResolvePreviousReleaseUsesSecondHighestActiveRemoteVersion(t *testing.T) {
	resolution, err := Resolve("previous-release", repo.Branches{
		Local: []string{"release-2.3"},
		Remote: []string{
			"origin/release-2.4",
			"origin/release-2.3",
			"origin/release-1.12",
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Target != "release-2.3" || resolution.Source != "local" {
		t.Fatalf("Resolve() = %#v, want previous active release-2.3", resolution)
	}
}

func TestResolveLatestReleaseIgnoresLocalOnlyRelease(t *testing.T) {
	resolution, err := Resolve("latest-release", repo.Branches{
		Local:  []string{"release-9.9"},
		Remote: []string{"origin/release-2.4"},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Target != "release-2.4" || resolution.Source != "remote" {
		t.Fatalf("Resolve() = %#v, want origin release-2.4", resolution)
	}
}

func TestResolvePreviousReleaseRequiresTwoActiveRemoteBranches(t *testing.T) {
	_, err := Resolve("previous-release", repo.Branches{
		Remote: []string{"origin/release-2.4"},
	})
	if err == nil {
		t.Fatal("Resolve() error = nil, want previous-release error")
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
