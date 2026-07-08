package repo

import (
	"io/fs"
	"path/filepath"
	"sort"
)

type Repository struct {
	Path string
}

func Discover(root string) ([]Repository, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	var repos []Repository
	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if entry.Name() == ".git" {
			repoPath := filepath.Dir(path)
			if isGitRepository(repoPath) {
				repos = append(repos, Repository{Path: repoPath})
			}
			return filepath.SkipDir
		}
		if shouldSkipDir(entry.Name()) && path != absRoot {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Path < repos[j].Path
	})
	return repos, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case "node_modules", "vendor", "dist", "build", "target", ".idea", ".vscode":
		return true
	default:
		return false
	}
}

func isGitRepository(path string) bool {
	output, err := gitOutput(path, "rev-parse", "--is-inside-work-tree")
	return err == nil && output == "true"
}
