package syncplan

import (
	"fmt"

	"github.com/black-osiris-technologies/omp-git-fleet/internal/branch"
	"github.com/black-osiris-technologies/omp-git-fleet/internal/repo"
)

type Action string

const (
	ActionReady   Action = "READY"
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

	resolution, err := branch.Resolve(target, branches)
	if err != nil {
		return RepoPlan{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}

	if resolution.Source == "local" {
		return RepoPlan{
			RepoPath: repoPath,
			Action:   ActionReady,
			Message:  fmt.Sprintf("would fetch --prune, checkout %s, pull --ff-only", resolution.LocalBranch),
		}
	}

	return RepoPlan{
		RepoPath: repoPath,
		Action:   ActionReady,
		Message:  fmt.Sprintf("would fetch --prune, create tracking branch %s from %s, pull --ff-only", resolution.LocalBranch, resolution.RemoteRef),
	}
}

func (s *Summary) Add(plan RepoPlan) {
	s.Total++
	switch plan.Action {
	case ActionReady:
		s.Ready++
	case ActionSkipped:
		s.Skipped++
	case ActionFailed:
		s.Failed++
	}
}
