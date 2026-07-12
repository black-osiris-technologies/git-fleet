package branch

import (
	"fmt"
	"regexp"
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

	if target == "latest-release" {
		return resolveLatestRelease(branches)
	}

	return resolveExplicit(target, branches)
}

func resolveExplicit(target string, branches repo.Branches) (Resolution, error) {
	if contains(branches.Local, target) {
		return Resolution{Target: target, LocalBranch: target, Source: "local"}, nil
	}

	remoteRef := "origin/" + target
	if contains(branches.Remote, remoteRef) {
		return Resolution{Target: target, LocalBranch: target, RemoteRef: remoteRef, Source: "remote"}, nil
	}

	return Resolution{}, fmt.Errorf("target %q not found locally or on origin", target)
}

func resolveLatestRelease(branches repo.Branches) (Resolution, error) {
	var candidates []releaseCandidate

	for _, branch := range branches.Local {
		if candidate, ok := parseReleaseBranch(branch, branch, "local"); ok {
			candidates = append(candidates, candidate)
		}
	}
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
		return Resolution{}, fmt.Errorf("no release branches found")
	}

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if compareVersions(candidate.version, best.version) > 0 {
			best = candidate
		}
	}

	resolution := Resolution{
		Target:      best.name,
		LocalBranch: best.name,
		Source:      best.source,
	}
	if best.source == "remote" {
		resolution.RemoteRef = best.ref
	}
	return resolution, nil
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
