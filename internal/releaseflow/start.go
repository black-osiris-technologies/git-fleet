package releaseflow

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/black-osiris-technologies/git-fleet/internal/repo"
	"github.com/black-osiris-technologies/git-fleet/internal/semver"
)

// DefaultBranchFormat is the naming git-fleet uses for a new release line when
// the caller does not override it. It yields names such as "release-2.4".
// Teams that follow a three-part convention can pass, for example,
// "release/{major}.{minor}.{patch}" to produce "release/2.4.0".
const DefaultBranchFormat = "release-{major}.{minor}"

// DefaultBaseBranch is the integration branch a release line is cut from.
const DefaultBaseBranch = "develop"

// errNoTags signals that a repository has no eligible stable tag to derive the
// next version from. It maps to a skip, not a failure: the repository simply has
// no release history yet and the caller must supply an explicit version.
var errNoTags = errors.New("no eligible release tag found; pass --version to seed the first release")

// StartOptions configures a single-repository release-start computation.
type StartOptions struct {
	// Bump selects minor (default) or major advancement of the latest tag.
	Bump semver.Bump
	// BranchFormat templates the branch name using {major}, {minor}, {patch}.
	BranchFormat string
	// BaseBranch is the integration branch the release line is cut from.
	BaseBranch string
	// ExplicitVersion overrides tag-derived version resolution when set.
	ExplicitVersion string
}

func (o StartOptions) withDefaults() StartOptions {
	if o.Bump == "" {
		o.Bump = semver.BumpMinor
	}
	if o.BranchFormat == "" {
		o.BranchFormat = DefaultBranchFormat
	}
	if o.BaseBranch == "" {
		o.BaseBranch = DefaultBaseBranch
	}
	return o
}

// PlanStart reports what StartRelease would do using the repository's current
// local state. It performs no fetch and mutates nothing, so a version derived
// here can lag the remote until StartRelease refreshes tags.
func PlanStart(repoPath string, opts StartOptions) Result {
	opts = opts.withDefaults()

	branchName, basis, err := resolveTarget(repoPath, opts)
	if err != nil {
		return skipOrFail(repoPath, err)
	}

	if exists, err := branchExists(repoPath, branchName); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	} else if exists {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: fmt.Sprintf("%s already exists", branchName)}
	}

	return Result{
		RepoPath: repoPath,
		Action:   ActionPlanned,
		Message:  fmt.Sprintf("would create %s from origin/%s (based on %s)", branchName, opts.BaseBranch, basis),
	}
}

// StartRelease creates the next release line for a single repository. It skips
// dirty worktrees, refreshes tags and branches with a pruning fetch, resolves
// the next version from the highest stable tag (or an explicit override), and is
// idempotent: an existing target branch is reported as skipped, never recreated.
func StartRelease(repoPath string, opts StartOptions, runner Runner) Result {
	opts = opts.withDefaults()

	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "fetch", "--prune", "--tags"); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	baseRef := "origin/" + opts.BaseBranch
	if exists, err := branchExists(repoPath, baseRef); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	} else if !exists {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: fmt.Sprintf("base branch %s not found", baseRef)}
	}

	branchName, _, err := resolveTarget(repoPath, opts)
	if err != nil {
		return skipOrFail(repoPath, err)
	}

	if exists, err := branchExists(repoPath, branchName); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	} else if exists {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: fmt.Sprintf("%s already exists", branchName)}
	}

	// Create the branch without checking it out so a fleet-wide run never leaves
	// repositories parked on a freshly created release branch.
	if _, err := runner.Run(repoPath, "git", "branch", branchName, baseRef); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "push", "-u", "origin", branchName); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	return Result{
		RepoPath: repoPath,
		Action:   ActionDone,
		Message:  fmt.Sprintf("created %s from %s", branchName, baseRef),
	}
}

// resolveTarget computes the release branch name and a human-readable basis for
// how the version was chosen (an explicit override or the highest tag).
func resolveTarget(repoPath string, opts StartOptions) (branchName string, basis string, err error) {
	version, basis, err := resolveVersion(repoPath, opts)
	if err != nil {
		return "", "", err
	}
	return formatBranch(opts.BranchFormat, version), basis, nil
}

func resolveVersion(repoPath string, opts StartOptions) (semver.Version, string, error) {
	if opts.ExplicitVersion != "" {
		version, err := semver.ParseVersion(opts.ExplicitVersion)
		if err != nil {
			return semver.Version{}, "", err
		}
		return version, "explicit version " + version.String(), nil
	}

	tags, err := repo.ListTags(repoPath)
	if err != nil {
		return semver.Version{}, "", err
	}
	highest, ok := semver.Highest(tags)
	if !ok {
		return semver.Version{}, "", errNoTags
	}
	next, err := semver.Next(highest, opts.Bump)
	if err != nil {
		return semver.Version{}, "", err
	}
	return next, fmt.Sprintf("tag %s, %s bump", highest, opts.Bump), nil
}

func formatBranch(format string, version semver.Version) string {
	replacer := strings.NewReplacer(
		"{major}", strconv.Itoa(version.Major),
		"{minor}", strconv.Itoa(version.Minor),
		"{patch}", strconv.Itoa(version.Patch),
	)
	return replacer.Replace(format)
}

// branchExists reports whether a local branch or remote-tracking ref of the
// given name is present. The name may be a bare branch ("release-2.4"), in
// which case the matching remote ref "origin/release-2.4" also counts, or an
// already-qualified remote ref ("origin/develop").
func branchExists(repoPath, name string) (bool, error) {
	branches, err := repo.ListBranches(repoPath)
	if err != nil {
		return false, err
	}
	if contains(branches.Local, name) || contains(branches.Remote, name) {
		return true, nil
	}
	return contains(branches.Remote, "origin/"+name), nil
}

func skipOrFail(repoPath string, err error) Result {
	if errors.Is(err, errNoTags) {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
}
