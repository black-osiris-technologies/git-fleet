package releaseflow

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/black-osiris-technologies/git-fleet/internal/branch"
	"github.com/black-osiris-technologies/git-fleet/internal/repo"
	"github.com/black-osiris-technologies/git-fleet/internal/semver"
)

// DefaultTagFormat is the tag naming git-fleet uses when the caller does not
// override it. It yields names such as "v2.4.0".
const DefaultTagFormat = "v{major}.{minor}.{patch}"

// releaseLinePattern extracts the MAJOR.MINOR line from a release branch name,
// accepting both "release-2.4" and "release/2.4.0" style names.
var releaseLinePattern = regexp.MustCompile(`^release[-/](\d+)\.(\d+)`)

// TagOptions configures a single-repository release-tag computation.
type TagOptions struct {
	// From selects the release branch: an explicit name or "latest-release".
	From string
	// TagFormat templates the tag name using {major}, {minor}, {patch}.
	TagFormat string
	// Message is the annotation message; when empty a default is used.
	Message string
	// ExplicitVersion pins the tag to a MAJOR.MINOR.PATCH value instead of
	// deriving the next patch from existing tags on the line.
	ExplicitVersion string
}

func (o TagOptions) withDefaults() TagOptions {
	if o.From == "" {
		o.From = "latest-release"
	}
	if o.TagFormat == "" {
		o.TagFormat = DefaultTagFormat
	}
	return o
}

// PlanTag reports what CreateTag would do using the repository's current local
// state. It performs no fetch and mutates nothing.
func PlanTag(repoPath string, opts TagOptions) Result {
	opts = opts.withDefaults()

	tagName, ref, err := resolveTag(repoPath, opts)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	return Result{
		RepoPath: repoPath,
		Action:   ActionPlanned,
		Message:  fmt.Sprintf("would tag %s as %s", ref, tagName),
	}
}

// CreateTag cuts the next patch tag on a repository's release line and pushes it.
// It refreshes tags with a pruning fetch, tags the authoritative origin tip of
// the release branch (so a stabilized commit is promoted, never a stale local
// one), and is idempotent for an explicitly pinned version: a tag that already
// exists is reported as skipped rather than recreated.
func CreateTag(repoPath string, opts TagOptions, runner Runner) Result {
	opts = opts.withDefaults()

	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "fetch", "--prune", "--tags"); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	tagName, ref, err := resolveTag(repoPath, opts)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}

	message := opts.Message
	if message == "" {
		message = "Release " + tagName
	}
	if _, err := runner.Run(repoPath, "git", "tag", "-a", tagName, "-m", message, ref); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "push", "origin", tagName); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	return Result{
		RepoPath: repoPath,
		Action:   ActionDone,
		Message:  fmt.Sprintf("tagged %s as %s", ref, tagName),
	}
}

// resolveTag determines the tag name to create and the origin ref to tag. It
// returns an error (mapped to a skip by callers) when there is no release
// branch on origin, the line cannot be parsed, or the computed tag already
// exists.
func resolveTag(repoPath string, opts TagOptions) (tagName string, ref string, err error) {
	branches, err := repo.ListBranches(repoPath)
	if err != nil {
		return "", "", err
	}
	resolution, err := branch.Resolve(opts.From, branches)
	if err != nil {
		return "", "", err
	}

	ref = "origin/" + resolution.Target
	if !contains(branches.Remote, ref) {
		return "", "", fmt.Errorf("release branch %s not found on origin", resolution.Target)
	}

	major, minor, ok := parseReleaseLine(resolution.Target)
	if !ok {
		return "", "", fmt.Errorf("cannot parse release line from branch %q", resolution.Target)
	}

	tags, err := repo.ListTags(repoPath)
	if err != nil {
		return "", "", err
	}

	version, err := nextTagVersion(tags, major, minor, opts.ExplicitVersion)
	if err != nil {
		return "", "", err
	}

	tagName = formatBranch(opts.TagFormat, version)
	if contains(tags, tagName) {
		return "", "", fmt.Errorf("tag %s already exists", tagName)
	}
	return tagName, ref, nil
}

// nextTagVersion computes the version to tag. With an explicit version it must
// match the release line and is used verbatim; otherwise the highest existing
// patch on the MAJOR.MINOR line is advanced by one, or .0 is used when the line
// has no tags yet.
func nextTagVersion(tags []string, major, minor int, explicit string) (semver.Version, error) {
	if explicit != "" {
		version, err := semver.ParseVersion(explicit)
		if err != nil {
			return semver.Version{}, err
		}
		if version.Major != major || version.Minor != minor {
			return semver.Version{}, fmt.Errorf("version %s is not on release line %d.%d", version, major, minor)
		}
		return version, nil
	}

	highestPatch := -1
	for _, tag := range tags {
		version, ok := semver.ParseTag(tag)
		if !ok || version.Major != major || version.Minor != minor {
			continue
		}
		if version.Patch > highestPatch {
			highestPatch = version.Patch
		}
	}
	return semver.Version{Major: major, Minor: minor, Patch: highestPatch + 1}, nil
}

func parseReleaseLine(name string) (major int, minor int, ok bool) {
	matches := releaseLinePattern.FindStringSubmatch(name)
	if matches == nil {
		return 0, 0, false
	}
	return mustAtoi(matches[1]), mustAtoi(matches[2]), true
}

func mustAtoi(value string) int {
	number, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("releaseflow: unexpected non-numeric component %q", value))
	}
	return number
}
