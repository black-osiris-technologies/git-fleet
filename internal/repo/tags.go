package repo

// ListTags returns the repository's local tags. The order is not significant;
// callers select the relevant tag themselves (for example the highest semantic
// version).
func ListTags(path string) ([]string, error) {
	output, err := gitOutput(path, "tag", "--list")
	if err != nil {
		return nil, err
	}
	return splitLines(output), nil
}

// ListOriginTags reads tag names directly from origin without mutating local
// refs. --refs excludes peeled annotated-tag entries such as refs/tags/v1^{}.
func ListOriginTags(path string) ([]string, error) {
	output, err := gitOutput(path, "ls-remote", "--tags", "--refs", "origin")
	if err != nil {
		return nil, err
	}
	return parseRemoteRefs(output, "refs/tags/"), nil
}
