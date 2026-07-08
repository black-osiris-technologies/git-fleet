package repo

import (
	"bytes"
	"os/exec"
	"strings"
)

type RepoStatus struct {
	Branch string
	Dirty  bool
}

func Status(path string) (RepoStatus, error) {
	branch, err := gitOutput(path, "branch", "--show-current")
	if err != nil {
		return RepoStatus{}, err
	}
	if branch == "" {
		branch = "detached"
	}

	porcelain, err := gitOutput(path, "status", "--porcelain")
	if err != nil {
		return RepoStatus{}, err
	}

	return RepoStatus{
		Branch: branch,
		Dirty:  porcelain != "",
	}, nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}
