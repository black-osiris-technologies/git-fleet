package syncplan

import (
	"fmt"

	"github.com/black-osiris-technologies/git-fleet/internal/branch"
	"github.com/black-osiris-technologies/git-fleet/internal/repo"
)

type Action string

const (
	ActionReady   Action = "READY"
	ActionDone    Action = "DONE"
	ActionSkipped Action = "SKIPPED"
	ActionFailed  Action = "FAILED"
)

type RepoPlan struct {
	RepoPath string
	Action   Action
	Message  string
}

type Summary struct {
	Total   int
	Ready   int
	Done    int
	Skipped int
	Failed  int
}

func Plan(repoPath, target string) RepoPlan {
	status, err := repo.Status(repoPath)
	if err != nil {
		return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	if status.Dirty {
		return RepoPlan{
			RepoPath: repoPath,
			Action:   ActionSkipped,
			Message:  fmt.Sprintf("dirty worktree on %s", status.Branch),
		}
	}

	branches, err := repo.ListBranches(repoPath)
	if err != nil {
		return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	// Automatic release selectors must reflect the live remote even in dry-run.
	// Keep local branch names so the plan can describe checkout vs tracking-branch
	// creation, but replace potentially stale remote-tracking refs with ls-remote
	// results from origin.
	if target == "latest-release" || target == "previous-release" {
		originBranches, err := repo.ListOriginBranches(repoPath)
		if err != nil {
			return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
		}
		branches.Remote = make([]string, 0, len(originBranches))
		for _, name := range originBranches {
			branches.Remote = append(branches.Remote, "origin/"+name)
		}
	}

	resolution, err := branch.Resolve(target, branches)
	if err != nil {
		return RepoPlan{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}

	update := "pull --ff-only"
	if resolution.RemoteRef != "" {
		update = fmt.Sprintf("merge --ff-only %s", resolution.RemoteRef)
	}

	if resolution.Source == "local" {
		return RepoPlan{
			RepoPath: repoPath,
			Action:   ActionReady,
			Message:  fmt.Sprintf("would fetch --prune --prune-tags --tags, checkout %s, %s", resolution.LocalBranch, update),
		}
	}

	return RepoPlan{
		RepoPath: repoPath,
		Action:   ActionReady,
		Message:  fmt.Sprintf("would fetch --prune --prune-tags --tags, create tracking branch %s from %s, %s", resolution.LocalBranch, resolution.RemoteRef, update),
	}
}

func Execute(repoPath, target string) RepoPlan {
	status, err := repo.Status(repoPath)
	if err != nil {
		return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	if status.Dirty {
		return RepoPlan{
			RepoPath: repoPath,
			Action:   ActionSkipped,
			Message:  fmt.Sprintf("dirty worktree on %s", status.Branch),
		}
	}

	if err := repo.GitRun(repoPath, "fetch", "--prune", "--prune-tags", "--tags"); err != nil {
		return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	branches, err := repo.ListBranches(repoPath)
	if err != nil {
		return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	resolution, err := branch.Resolve(target, branches)
	if err != nil {
		return RepoPlan{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}

	if resolution.Source == "local" {
		if err := repo.GitRun(repoPath, "checkout", resolution.LocalBranch); err != nil {
			return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
		}
	} else {
		if err := repo.GitRun(repoPath, "checkout", "--track", "-b", resolution.LocalBranch, resolution.RemoteRef); err != nil {
			return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
		}
	}

	if resolution.RemoteRef != "" {
		if err := repo.GitRun(repoPath, "merge", "--ff-only", resolution.RemoteRef); err != nil {
			return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
		}
	} else if err := repo.GitRun(repoPath, "pull", "--ff-only"); err != nil {
		// Backward-compatible fallback for an explicit local-only branch. Automatic
		// release selectors and normal origin-backed branches never use this path.
		return RepoPlan{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	return RepoPlan{
		RepoPath: repoPath,
		Action:   ActionDone,
		Message:  fmt.Sprintf("synced %s", resolution.LocalBranch),
	}
}

func (s *Summary) Add(plan RepoPlan) {
	s.Total++
	switch plan.Action {
	case ActionReady:
		s.Ready++
	case ActionDone:
		s.Done++
	case ActionSkipped:
		s.Skipped++
	case ActionFailed:
		s.Failed++
	}
}
