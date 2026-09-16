package releaseflow

import (
	"fmt"
	"strings"
)

// FinishOptions configures a single-repository release-finish operation.
type FinishOptions struct {
	// From selects the release branch: an explicit name or "latest-release".
	From string
	// MasterBranch is the production branch the release is merged into.
	MasterBranch string
	// DevelopBranch is the integration branch the release is merged into.
	DevelopBranch string
	// DeleteBranch removes the release branch on origin once both merges land.
	DeleteBranch bool
}

func (o FinishOptions) withDefaults() FinishOptions {
	if o.From == "" {
		o.From = "latest-release"
	}
	if o.MasterBranch == "" {
		o.MasterBranch = "master"
	}
	if o.DevelopBranch == "" {
		o.DevelopBranch = "develop"
	}
	return o
}

// PlanFinish reports what FinishRelease would do. Release selection is resolved
// against live origin state so a local-only or stale branch cannot be planned for
// GitHub PR promotion. Planning does not mutate refs or pull requests.
func PlanFinish(repoPath string, opts FinishOptions) Result {
	opts = opts.withDefaults()

	release, err := resolveRemoteRelease(repoPath, opts.From)
	if err != nil {
		return Result{RepoPath: repoPath, Action: releaseErrorAction(err), Message: err.Error()}
	}

	message := fmt.Sprintf("would merge open PRs %s -> %s and %s -> %s with merge commits",
		release.LocalBranch, opts.MasterBranch, release.LocalBranch, opts.DevelopBranch)
	if opts.DeleteBranch {
		message += fmt.Sprintf(", then delete %s if origin has not advanced it", release.LocalBranch)
	}
	return Result{RepoPath: repoPath, Action: ActionPlanned, Message: message}
}

// FinishRelease completes a GitFlow release by merging the release branch into
// both the production and integration branches with merge commits (never a
// squash), preserving release history on both lines per the branching model.
// Merges go through pull requests, so branch protection and review are honored;
// only a git or gh failure produces a failed action. An already-integrated line
// (no open PR) is reported as skipped, making the operation safe to re-run. The
// release branch is deleted only when --delete-branch is set and both merges
// succeeded in this run. Deletion is protected by a force-with-lease tied to the
// fetched release SHA, so a branch that advanced during the operation is kept.
func FinishRelease(repoPath string, opts FinishOptions, runner Runner) Result {
	opts = opts.withDefaults()

	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "fetch", "--prune", "--tags"); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	release, err := resolveRemoteRelease(repoPath, opts.From)
	if err != nil {
		return Result{RepoPath: repoPath, Action: releaseErrorAction(err), Message: err.Error()}
	}

	var parts []string
	merged := 0
	failed := false
	for _, target := range []string{opts.MasterBranch, opts.DevelopBranch} {
		action, message := mergeReleaseInto(repoPath, release.LocalBranch, target, runner)
		parts = append(parts, fmt.Sprintf("%s: %s", target, message))
		switch action {
		case ActionDone:
			merged++
		case ActionFailed:
			failed = true
		}
	}

	if opts.DeleteBranch && !failed && merged == 2 {
		releaseRef := "origin/" + release.LocalBranch
		expectedSHA, err := runner.Run(repoPath, "git", "rev-parse", releaseRef)
		if err != nil {
			parts = append(parts, "delete precondition failed: "+err.Error())
			failed = true
		} else if strings.TrimSpace(expectedSHA) == "" {
			parts = append(parts, "delete precondition failed: empty release SHA")
			failed = true
		} else {
			lease := fmt.Sprintf("--force-with-lease=refs/heads/%s:%s", release.LocalBranch, strings.TrimSpace(expectedSHA))
			deleteRefspec := fmt.Sprintf(":refs/heads/%s", release.LocalBranch)
			if _, err := runner.Run(repoPath, "git", "push", lease, "origin", deleteRefspec); err != nil {
				parts = append(parts, "delete refused: "+err.Error())
				failed = true
			} else {
				parts = append(parts, "deleted "+release.LocalBranch)
			}
		}
	}

	action := ActionSkipped
	switch {
	case failed:
		action = ActionFailed
	case merged > 0:
		action = ActionDone
	}
	return Result{RepoPath: repoPath, Action: action, Message: strings.Join(parts, "; ")}
}
