package releaseflow

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/black-osiris-technologies/git-fleet/internal/branch"
	"github.com/black-osiris-technologies/git-fleet/internal/repo"
)

type Action string

const (
	ActionPlanned Action = "PLANNED"
	ActionDone    Action = "DONE"
	ActionSkipped Action = "SKIPPED"
	ActionFailed  Action = "FAILED"
)

type Result struct {
	RepoPath string
	Action   Action
	Message  string
}

type Summary struct {
	Total   int
	Planned int
	Done    int
	Skipped int
	Failed  int
}

type Runner interface {
	Run(dir string, command string, args ...string) (string, error)
}

type RealRunner struct{}

func (RealRunner) Run(dir string, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", command, strings.Join(args, " "), message)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func PlanPR(repoPath, from, to string) Result {
	release, err := resolveRelease(repoPath, from)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if err := validateTarget(repoPath, to); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	return Result{
		RepoPath: repoPath,
		Action:   ActionPlanned,
		Message:  fmt.Sprintf("would create PR %s -> %s using merge commits", release.LocalBranch, to),
	}
}

func CreatePR(repoPath, from, to string, runner Runner) Result {
	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "fetch", "--prune"); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	release, err := resolveRelease(repoPath, from)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if err := validateTarget(repoPath, to); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if release.Source == "local" {
		if _, err := runner.Run(repoPath, "git", "push", "-u", "origin", release.LocalBranch); err != nil {
			return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
		}
	}

	title := fmt.Sprintf("Release %s into %s", release.LocalBranch, to)
	body := "Created by git-fleet release-pr. Merge with a merge commit; do not squash."
	output, err := runner.Run(repoPath, "gh", "pr", "create", "--base", to, "--head", release.LocalBranch, "--title", title, "--body", body)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return Result{RepoPath: repoPath, Action: ActionSkipped, Message: "release PR already exists"}
		}
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}
	return Result{RepoPath: repoPath, Action: ActionDone, Message: firstLine(output, "created release PR")}
}

func PlanMerge(repoPath, from, to string) Result {
	release, err := resolveRelease(repoPath, from)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if err := validateTarget(repoPath, to); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	return Result{
		RepoPath: repoPath,
		Action:   ActionPlanned,
		Message:  fmt.Sprintf("would merge open PR %s -> %s with merge commit", release.LocalBranch, to),
	}
}

func MergePR(repoPath, from, to string, runner Runner) Result {
	if err := ensureClean(repoPath); err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}
	if _, err := runner.Run(repoPath, "git", "fetch", "--prune"); err != nil {
		return Result{RepoPath: repoPath, Action: ActionFailed, Message: err.Error()}
	}

	release, err := resolveRelease(repoPath, from)
	if err != nil {
		return Result{RepoPath: repoPath, Action: ActionSkipped, Message: err.Error()}
	}

	action, message := mergeReleaseInto(repoPath, release.LocalBranch, to, runner)
	return Result{RepoPath: repoPath, Action: action, Message: message}
}

// mergeReleaseInto merges the open pull request from releaseBranch into the
// target branch with a merge commit. A missing target or absent open PR yields a
// skip so callers can treat an already-integrated line as safe to re-run; only a
// git or gh failure yields a failed action.
func mergeReleaseInto(repoPath, releaseBranch, to string, runner Runner) (Action, string) {
	if err := validateTarget(repoPath, to); err != nil {
		return ActionSkipped, err.Error()
	}

	number, err := runner.Run(repoPath, "gh", "pr", "list", "--base", to, "--head", releaseBranch, "--state", "open", "--json", "number", "--jq", ".[0].number")
	if err != nil {
		return ActionFailed, err.Error()
	}
	if strings.TrimSpace(number) == "" {
		return ActionSkipped, fmt.Sprintf("no open release PR %s -> %s", releaseBranch, to)
	}

	output, err := runner.Run(repoPath, "gh", "pr", "merge", strings.TrimSpace(number), "--merge")
	if err != nil {
		return ActionFailed, err.Error()
	}
	return ActionDone, firstLine(output, fmt.Sprintf("merged %s -> %s", releaseBranch, to))
}

func (s *Summary) Add(result Result) {
	s.Total++
	switch result.Action {
	case ActionPlanned:
		s.Planned++
	case ActionDone:
		s.Done++
	case ActionSkipped:
		s.Skipped++
	case ActionFailed:
		s.Failed++
	}
}

func ensureClean(repoPath string) error {
	status, err := repo.Status(repoPath)
	if err != nil {
		return err
	}
	if status.Dirty {
		return fmt.Errorf("dirty worktree on %s", status.Branch)
	}
	return nil
}

func resolveRelease(repoPath, from string) (branch.Resolution, error) {
	branches, err := repo.ListBranches(repoPath)
	if err != nil {
		return branch.Resolution{}, err
	}
	return branch.Resolve(from, branches)
}

func validateTarget(repoPath, to string) error {
	branches, err := repo.ListBranches(repoPath)
	if err != nil {
		return err
	}
	if contains(branches.Remote, "origin/"+to) {
		return nil
	}
	return fmt.Errorf("target branch %q not found on origin", to)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func firstLine(output, fallback string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return fallback
	}
	return strings.Split(output, "\n")[0]
}
