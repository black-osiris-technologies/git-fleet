package branch

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/black-osiris-technologies/git-fleet/internal/repo"
)

type Resolution struct {
	Target      string
	LocalBranch string
	RemoteRef   string
	Source      string
}

func Resolve(target string, branches repo.Branches) (Resolution, error) {
	if target == "" {
		return Resolution{}, fmt.Errorf("target branch is required")
	}

	switch target {
	case "latest-release":
		return resolveActiveRelease(branches, 0)
	case "previous-release":
		return resolveActiveRelease(branches, 1)
	default:
		return resolveExplicit(target, branches)
	}
}

func resolveExplicit(target string, branches repo.Branches) (Resolution, error) {
	remoteRef := "origin/" + target
	localExists := contains(branches.Local, target)
	remoteExists := contains(branches.Remote, remoteRef)

	if localExists {
		resolution := Resolution{Target: target, LocalBranch: target, Source: "local"}
		if remoteExists {
			resolution.RemoteRef = remoteRef
		}
		return resolution, nil
	}

	if remoteExists {
		return Resolution{Target: target, LocalBranch: target, RemoteRef: remoteRef, Source: "remote"}, nil
	}

	return Resolution{}, fmt.Errorf("target %q not found locally or on origin", target)
}

// resolveActiveRelease resolves an automatically selected release branch using
// origin as the source of truth. Local-only release branches are deliberately
// ignored: they may be stale branches whose remote counterpart was deleted.
// rank 0 selects the latest active release and rank 1 the previous active one.
func resolveActiveRelease(branches repo.Branches, rank int) (Resolution, error) {
	var candidates []releaseCandidate

	for _, remoteRef := range branches.Remote {
		if !strings.HasPrefix(remoteRef, "origin/") {
			continue
		}
		branchName := strings.TrimPrefix(remoteRef, "origin/")
		if candidate, ok := parseReleaseBranch(branchName, remoteRef, "remote"); ok {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		return Resolution{}, fmt.Errorf("no active release branches found on origin")
	}

	sort.Slice(candidates, func(i, j int) bool {
		comparison := compareVersions(candidates[i].version, candidates[j].version)
		if comparison == 0 {
			return candidates[i].name < candidates[j].name
		}
		return comparison > 0
	})

	if rank >= len(candidates) {
		return Resolution{}, fmt.Errorf("previous-release requires at least 2 active release branches on origin")
	}

	selected := candidates[rank]
	if contains(branches.Local, selected.name) {
		return Resolution{
			Target:      selected.name,
			LocalBranch: selected.name,
			RemoteRef:   selected.ref,
			Source:      "local",
		}, nil
	}

	return Resolution{
		Target:      selected.name,
		LocalBranch: selected.name,
		RemoteRef:   selected.ref,
		Source:      "remote",
	}, nil
}

type releaseCandidate struct {
	name    string
	ref     string
	source  string
	version []int
}

var releasePattern = regexp.MustCompile(`^release[-/](\d+(?:\.\d+)*)$`)

func parseReleaseBranch(name, ref, source string) (releaseCandidate, bool) {
	matches := releasePattern.FindStringSubmatch(name)
	if matches == nil {
		return releaseCandidate{}, false
	}

	parts := strings.Split(matches[1], ".")
	version := make([]int, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return releaseCandidate{}, false
		}
		version = append(version, number)
	}

	return releaseCandidate{name: name, ref: ref, source: source, version: version}, true
}

func compareVersions(left, right []int) int {
	length := len(left)
	if len(right) > length {
		length = len(right)
	}

	for i := 0; i < length; i++ {
		leftValue := 0
		rightValue := 0
		if i < len(left) {
			leftValue = left[i]
		}
		if i < len(right) {
			rightValue = right[i]
		}
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return 0
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
