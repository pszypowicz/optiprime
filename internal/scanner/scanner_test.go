package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pszypowicz/optiprime/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(p, 0o755))
}

func mustWriteFile(t *testing.T, p, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(p, []byte(contents), 0o644))
}

func TestFindRepos_MixedTree(t *testing.T) {
	root := t.TempDir()

	// alpha/.git/ -> included (dir)
	mustMkdir(t, filepath.Join(root, "alpha", ".git"))
	// bravo/.git (regular file, gitfile pattern) -> included
	mustMkdir(t, filepath.Join(root, "bravo"))
	mustWriteFile(t, filepath.Join(root, "bravo", ".git"), "gitdir: /elsewhere\n")
	// charlie/ (no .git) -> excluded
	mustMkdir(t, filepath.Join(root, "charlie"))
	// echo.txt (plain file at root) -> excluded
	mustWriteFile(t, filepath.Join(root, "echo.txt"), "hi")

	repos, err := scanner.FindRepos(root)
	require.NoError(t, err)
	require.Len(t, repos, 2)

	assert.Equal(t, "alpha", repos[0].Name)
	assert.Equal(t, filepath.Join(root, "alpha"), repos[0].Path)
	assert.Equal(t, "bravo", repos[1].Name)
	assert.Equal(t, filepath.Join(root, "bravo"), repos[1].Path)
}

func TestFindRepos_SortOrder(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zeta", "alpha", "mu"} {
		mustMkdir(t, filepath.Join(root, name, ".git"))
	}

	repos, err := scanner.FindRepos(root)
	require.NoError(t, err)
	names := []string{repos[0].Name, repos[1].Name, repos[2].Name}
	assert.Equal(t, []string{"alpha", "mu", "zeta"}, names)
}

func TestFindRepos_NonexistentRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	repos, err := scanner.FindRepos(missing)
	assert.Error(t, err)
	assert.Nil(t, repos)
}

func TestFindRepos_EmptyRoot(t *testing.T) {
	root := t.TempDir()
	repos, err := scanner.FindRepos(root)
	require.NoError(t, err)
	assert.Empty(t, repos)
}

// TestFindRepos_HiddenDirIncluded documents current behavior: there is no
// filter for dot-prefixed directory names, so a hidden repo is picked up.
func TestFindRepos_HiddenDirIncluded(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".hidden", ".git"))

	repos, err := scanner.FindRepos(root)
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, ".hidden", repos[0].Name)
}
