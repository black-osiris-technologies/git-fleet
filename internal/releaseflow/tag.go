package releaseflow

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/black-osiris-technologies/git-fleet/internal/branch"
	"github.com/black-osiris-technologies/git-fleet/internal/repo"
	"github.com/black-osiris-technologies/git-fleet/internal/semver"
)

// DefaultTagFormat is the naming git-fleet uses when the caller does not
// override it. It yields names such as "v2.4.0".
const DefaultTagFormat = "v{major}.{minor}.{patch}"

// releaseLinePattern extracts the MAJOR.MINOR line from a release branch name,
// accepting both "release-2.4" and "release/2.4.0" style names.
var releaseLinePattern = regexp.MustCompile(`^release[-/](\d+)\.(\d+)`)

// TagOptions configures a single-repository release-tag computation.
type TagOptions struct {
	// From selects the release branch: an explicit name, "latest-release", or
	// "previous-release".
	From string
	// TagFormat templates the tag name using {major}, {minor}, {patch}. To remain
	// compatible with release-start, only stable SemVer tags with optional "v"
	// prefix are supported.
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

func validateTagFormat(format string) error {
	if format == "v{major}.{minor}.{patch}" || format == "{major}.{minor}.{patch}" {
		return nil
	}
	return fmt.Errorf(
		"tag format %q is incompatible with release version discovery; use v{major}.{minor}.{patch} or {major}.{minor}.{patch}",
		format,
	)
}

// PlanTag reports what CreateTag would do using origin as the source of truth.
// It reads branches and tags with ls-remote, so dry-run does not mutate local
// refs and cannot be influenced by stale local tags or remote-tracking branches.
func PlanTag(repoPath string, opts TagOptions) Result {
	opts = opts.withDefaults()
	if err := validateTagFormat(opts.TagFormat); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

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
// It prunes stale local tags, derives the patch sequence from tags currently on
// origin, and tags the fetched origin tip of the release branch. An explicitly
// pinned version that already exists on origin is reported as skipped.
func CreateTag(repoPath string, opts TagOptions, runner Runner) Result {
	opts = opts.withDefaults()
	if err := validateTagFormat(opts.TagFormat); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "fetch", "--prune", "--prune-tags", "--tags"); err != nil {
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

// resolveTag determines the tag name to create and the fetched origin ref to tag.
// Branch and tag discovery is performed directly against origin so stale local
// state cannot affect release selection or patch sequencing.
func resolveTag(repoPath string, opts TagOptions) (tagName string, ref string, err error) {
	originBranches, err := repo.ListOriginBranches(repoPath)
	if err != nil {
		return "", "", err
	}

	remoteRefs := make([]string, 0, len(originBranches))
	for _, name := range originBranches {
		remoteRefs = append(remoteRefs, "origin/"+name)
	}
	resolution, err := branch.Resolve(opts.From, repo.Branches{Remote: remoteRefs})
	if err != nil {
		return "", "", err
	}

	ref = "origin/" + resolution.Target
	major, minor, ok := parseReleaseLine(resolution.Target)
	if !ok {
		return "", "", fmt.Errorf("cannot parse release line from branch %q", resolution.Target)
	}

	tags, err := repo.ListOriginTags(repoPath)
	if err != nil {
		return "", "", err
	}

	version, err := nextTagVersion(tags, major, minor, opts.ExplicitVersion)
	if err != nil {
		return "", "", err
	}

	tagName = formatBranch(opts.TagFormat, version)
	if parsed, ok := semver.ParseTag(tagName); !ok || parsed != version {
		return "", "", fmt.Errorf("tag format %q produced incompatible tag %q", opts.TagFormat, tagName)
	}
	if contains(tags, tagName) {
		return "", "", fmt.Errorf("tag %s already exists on origin", tagName)
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
