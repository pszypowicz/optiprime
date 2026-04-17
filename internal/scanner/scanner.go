package scanner

import (
	"os"
	"path/filepath"
	"sort"
)

type LocalRepo struct {
	Name string
	Path string
}

func FindRepos(root string) ([]LocalRepo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var repos []LocalRepo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() == "." || e.Name() == ".." {
			continue
		}
		p := filepath.Join(root, e.Name())
		gitPath := filepath.Join(p, ".git")
		if fi, err := os.Stat(gitPath); err == nil {
			if fi.IsDir() || fi.Mode().IsRegular() {
				repos = append(repos, LocalRepo{Name: e.Name(), Path: p})
			}
		}
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	return repos, nil
}
