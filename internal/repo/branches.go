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

// ListOriginBranches reads branch names directly from origin without updating
// local refs. This is used by dry-run planning when origin must be authoritative
// even if local remote-tracking refs are stale.
func ListOriginBranches(path string) ([]string, error) {
	output, err := gitOutput(path, "ls-remote", "--heads", "origin")
	if err != nil {
		return nil, err
	}
	return parseRemoteRefs(output, "refs/heads/"), nil
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

func parseRemoteRefs(output, prefix string) []string {
	lines := splitLines(output)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[1], prefix) {
			continue
		}
		result = append(result, strings.TrimPrefix(fields[1], prefix))
	}
	return result
}
