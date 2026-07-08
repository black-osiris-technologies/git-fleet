package repo

import (
	"io/fs"
	"os"
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
			repos = append(repos, Repository{Path: filepath.Dir(path)})
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

func gitDirExists(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && info.IsDir()
}
