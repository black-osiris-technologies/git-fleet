package repo

import "strings"

type Branches struct {
	Local  []string
	Remote []string
}

func ListBranches(path string) (Branches, error) {
	localOutput, err := gitOutput(path, "branch", "--format=%(refname:short)")
	if err != nil {
		return Branches{}, err
	}

	remoteOutput, err := gitOutput(path, "branch", "-r", "--format=%(refname:short)")
	if err != nil {
		return Branches{}, err
	}

	return Branches{
		Local:  splitLines(localOutput),
		Remote: filterRemoteBranches(splitLines(remoteOutput)),
	}, nil
}

func splitLines(value string) []string {
	if value == "" {
		return nil
	}

	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func filterRemoteBranches(branches []string) []string {
	result := make([]string, 0, len(branches))
	for _, branch := range branches {
		if strings.Contains(branch, " -> ") {
			continue
		}
		result = append(result, branch)
	}
	return result
}
