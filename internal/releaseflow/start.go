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

// PlanStart reports what StartRelease would do using origin as the source of
// truth. It reads remote refs with ls-remote, so dry-run stays accurate without
// fetching or mutating local tags, branches, or remote-tracking refs.
func PlanStart(repoPath string, opts StartOptions) Result {
	opts = opts.withDefaults()

	branches, err := repo.ListOriginBranches(repoPath)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}
	if !contains(branches, opts.BaseBranch) {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: fmt.Sprintf("base branch origin/%s not found", opts.BaseBranch)}
	}

	branchName, basis, err := resolveTarget(repoPath, opts)
	if err != nil {
		return skipOrFail(repoPath, err)
	}
	if contains(branches, branchName) {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: fmt.Sprintf("%s already exists on origin", branchName)}
	}

	return Result{
		RepoPath: repoPath,
		Action:   ActionPlanned,
		Message:  fmt.Sprintf("would create %s from origin/%s (based on %s)", branchName, opts.BaseBranch, basis),
	}
}

// StartRelease creates the next release line for a single repository. It skips
// dirty worktrees, prunes stale remote-tracking refs and tags, resolves the next
// version from tags currently present on origin, and creates the remote branch
// directly from origin/<base>. A same-named local-only branch is deliberately
// ignored and never deleted or pushed.
func StartRelease(repoPath string, opts StartOptions, runner Runner) Result {
	opts = opts.withDefaults()

	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "fetch", "--prune", "--prune-tags", "--tags"); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	branches, err := repo.ListOriginBranches(repoPath)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}
	if !contains(branches, opts.BaseBranch) {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: fmt.Sprintf("base branch origin/%s not found", opts.BaseBranch)}
	}

	branchName, _, err := resolveTarget(repoPath, opts)
	if err != nil {
		return skipOrFail(repoPath, err)
	}
	if contains(branches, branchName) {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: fmt.Sprintf("%s already exists on origin", branchName)}
	}

	baseRef := "origin/" + opts.BaseBranch
	refspec := fmt.Sprintf("%s:refs/heads/%s", baseRef, branchName)
	// Empty expected value means the destination ref must not exist. This keeps
	// the create race-safe: if another actor creates the release branch after our
	// ls-remote check, the push is rejected instead of advancing that branch.
	lease := fmt.Sprintf("--force-with-lease=refs/heads/%s:", branchName)
	if _, err := runner.Run(repoPath, "git", "push", lease, "origin", refspec); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	return Result{
		RepoPath: repoPath,
		Action:   ActionDone,
		Message:  fmt.Sprintf("created %s from %s", branchName, baseRef),
	}
}

// resolveTarget computes the release branch name and a human-readable basis for
// how the version was chosen (an explicit override or the highest origin tag).
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

	tags, err := repo.ListOriginTags(repoPath)
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

func skipOrFail(repoPath string, err error) Result {
	if errors.Is(err, errNoTags) {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
}
