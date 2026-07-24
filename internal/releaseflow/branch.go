package releaseflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/black-osiris-technologies/git-fleet/internal/repo"
)

type nextRelease struct {
	Branch string
	Source string
}

var (
	releaseBranchPattern = regexp.MustCompile(`^release[-/](\d+)\.(\d+)(?:\.\d+)?$`)
	semverTagPattern    = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)
)

func PlanBranch(repoPath string) Result {
	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	release, err := nextReleaseBranch(repoPath)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if err := ensureDevelopSynced(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	return Result{
		RepoPath: repoPath,
		Action:   ActionPlanned,
		Message:  fmt.Sprintf("would create %s from develop (%s)", release.Branch, release.Source),
	}
}

func CreateBranch(repoPath string, runner Runner) Result {
	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "fetch", "--prune", "--tags", "origin"); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}
	release, err := nextReleaseBranch(repoPath)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if err := ensureDevelopSynced(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "checkout", "develop"); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "checkout", "-b", release.Branch); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}
	return Result{
		RepoPath: repoPath,
		Action:   ActionDone,
		Message:  fmt.Sprintf("created %s from develop (%s)", release.Branch, release.Source),
	}
}

func ensureDevelopSynced(repoPath string) error {
	branches, err := repo.ListBranches(repoPath)
	if err != nil {
		return err
	}
	if !contains(branches.Local, "develop") {
		return fmt.Errorf("local develop branch not found")
	}
	if !contains(branches.Remote, "origin/develop") {
		return fmt.Errorf("origin/develop not found")
	}

	local, err := repo.GitOutput(repoPath, "rev-parse", "develop")
	if err != nil {
		return err
	}
	remote, err := repo.GitOutput(repoPath, "rev-parse", "origin/develop")
	if err != nil {
		return err
	}
	if local != remote {
		return fmt.Errorf("local develop is not up to date with origin/develop")
	}
	return nil
}

func nextReleaseBranch(repoPath string) (nextRelease, error) {
	branches, err := repo.ListBranches(repoPath)
	if err != nil {
		return nextRelease{}, err
	}

	if release, ok := nextFromReleaseBranches(branches); ok {
		return release, nil
	}

	tags, err := repo.GitOutput(repoPath, "tag", "--list")
	if err != nil {
		return nextRelease{}, err
	}
	if release, ok := nextFromTags(splitReleaseLines(tags)); ok {
		return release, nil
	}

	return nextRelease{}, fmt.Errorf("no release branches or semver tags found")
}

func nextFromReleaseBranches(branches repo.Branches) (nextRelease, bool) {
	best := version{}
	found := false
	for _, branch := range append(branches.Local, remoteBranchNames(branches.Remote)...) {
		current, ok := parseReleaseBranchVersion(branch)
		if !ok {
			continue
		}
		if !found || compareReleaseVersions(current, best) > 0 {
			best = current
			found = true
		}
	}
	if !found {
		return nextRelease{}, false
	}
	return nextRelease{
		Branch: fmt.Sprintf("release-%d.%d", best.Major, best.Minor+1),
		Source: fmt.Sprintf("next after release-%d.%d", best.Major, best.Minor),
	}, true
}

func nextFromTags(tags []string) (nextRelease, bool) {
	best := version{}
	bestTag := ""
	found := false
	for _, tag := range tags {
		current, ok := parseSemverTag(tag)
		if !ok {
			continue
		}
		if !found || compareReleaseVersions(current, best) > 0 {
			best = current
			bestTag = tag
			found = true
		}
	}
	if !found {
		return nextRelease{}, false
	}
	return nextRelease{
		Branch: fmt.Sprintf("release-%d.%d", best.Major, best.Minor+1),
		Source: fmt.Sprintf("next after tag %s", bestTag),
	}, true
}

type version struct {
	Major int
	Minor int
	Patch int
}

func parseReleaseBranchVersion(name string) (version, bool) {
	matches := releaseBranchPattern.FindStringSubmatch(name)
	if matches == nil {
		return version{}, false
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return version{}, false
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return version{}, false
	}
	return version{Major: major, Minor: minor}, true
}

func parseSemverTag(tag string) (version, bool) {
	matches := semverTagPattern.FindStringSubmatch(tag)
	if matches == nil {
		return version{}, false
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return version{}, false
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return version{}, false
	}
	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return version{}, false
	}
	return version{Major: major, Minor: minor, Patch: patch}, true
}

func compareReleaseVersions(left, right version) int {
	switch {
	case left.Major != right.Major:
		return compareInts(left.Major, right.Major)
	case left.Minor != right.Minor:
		return compareInts(left.Minor, right.Minor)
	default:
		return compareInts(left.Patch, right.Patch)
	}
}

func compareInts(left, right int) int {
	switch {
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}

func remoteBranchNames(branches []string) []string {
	names := make([]string, 0, len(branches))
	for _, branch := range branches {
		names = append(names, strings.TrimPrefix(branch, "origin/"))
	}
	return names
}

func splitReleaseLines(value string) []string {
	if value == "" {
		return nil
	}
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
