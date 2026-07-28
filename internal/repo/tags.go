package repo

// ListTags returns the repository's tags. The order is not significant; callers
// select the relevant tag themselves (for example the highest semantic version).
func ListTags(path string) ([]string, error) {
	output, err := gitOutput(path, "tag", "--list")
	if err != nil {
		return nil, err
	}
	return splitLines(output), nil
}
