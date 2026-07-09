package repo

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
