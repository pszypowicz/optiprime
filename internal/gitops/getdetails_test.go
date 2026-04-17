package gitops

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDetails_HappyPath(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("abc1234\tFix bug\tAlice\t2 hours ago", "", nil,
		"log", "-1", "--pretty=format:%h\t%s\t%an\t%ar")
	f.Set(" M src/a.go\n?? new.txt", "", nil, "status", "--porcelain=v1")
	f.Set("stash@{0}\tWIP on main\t3 days ago\nstash@{1}\texperiment\t1 week ago",
		"", nil, "stash", "list", "--format=%gd\t%gs\t%cr")
	f.Set("origin/main", "", nil, "rev-parse", "--abbrev-ref", "@{upstream}")
	f.Set("git@ssh.dev.azure.com:v3/org/proj/repo", "", nil, "remote", "get-url", "origin")

	d, err := getDetails(f, "/repo")
	require.NoError(t, err)

	assert.Equal(t, "abc1234", d.LastCommitSHA)
	assert.Equal(t, "Fix bug", d.LastCommitSubject)
	assert.Equal(t, "Alice", d.LastCommitAuthor)
	assert.Equal(t, "2 hours ago", d.LastCommitAge)

	require.Len(t, d.DirtyFiles, 2)
	assert.Equal(t, " M", d.DirtyFiles[0].XY)
	assert.Equal(t, "src/a.go", d.DirtyFiles[0].Path)
	assert.Equal(t, "??", d.DirtyFiles[1].XY)
	assert.Equal(t, "new.txt", d.DirtyFiles[1].Path)

	require.Len(t, d.Stashes, 2)
	assert.Equal(t, "stash@{0}", d.Stashes[0].Ref)
	assert.Equal(t, "WIP on main", d.Stashes[0].Subject)

	assert.Equal(t, "origin/main", d.UpstreamBranch)
	assert.Equal(t, "git@ssh.dev.azure.com:v3/org/proj/repo", d.RemoteURL)
}

func TestGetDetails_EmptyLogHandled(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("", "", nil, "log", "-1", "--pretty=format:%h\t%s\t%an\t%ar")
	f.Set("", "", nil, "status", "--porcelain=v1")
	f.Set("", "", nil, "stash", "list", "--format=%gd\t%gs\t%cr")
	f.Set("", "", errors.New("no upstream"), "rev-parse", "--abbrev-ref", "@{upstream}")
	f.Set("", "", errors.New("no origin"), "remote", "get-url", "origin")

	d, err := getDetails(f, "/repo")
	require.NoError(t, err)
	assert.Empty(t, d.LastCommitSHA)
	assert.Empty(t, d.DirtyFiles)
	assert.Empty(t, d.Stashes)
	assert.Empty(t, d.UpstreamBranch)
	assert.Empty(t, d.RemoteURL)
}

func TestGetDetails_ShortStatusLineSkipped(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("", "", nil, "log", "-1", "--pretty=format:%h\t%s\t%an\t%ar")
	// One valid line, one too-short line that must be skipped.
	f.Set(" M a.go\nxx", "", nil, "status", "--porcelain=v1")
	f.Set("", "", nil, "stash", "list", "--format=%gd\t%gs\t%cr")
	f.Set("origin/main", "", nil, "rev-parse", "--abbrev-ref", "@{upstream}")
	f.Set("", "", nil, "remote", "get-url", "origin")

	d, err := getDetails(f, "/repo")
	require.NoError(t, err)
	require.Len(t, d.DirtyFiles, 1)
	assert.Equal(t, "a.go", d.DirtyFiles[0].Path)
}

func TestGetDetails_StashLineWithoutTabsSkipped(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("", "", nil, "log", "-1", "--pretty=format:%h\t%s\t%an\t%ar")
	f.Set("", "", nil, "status", "--porcelain=v1")
	// Good line, then a malformed line missing tabs.
	f.Set("stash@{0}\tWIP\t1 day ago\nmalformed no tabs here",
		"", nil, "stash", "list", "--format=%gd\t%gs\t%cr")
	f.Set("", "", nil, "rev-parse", "--abbrev-ref", "@{upstream}")
	f.Set("", "", nil, "remote", "get-url", "origin")

	d, err := getDetails(f, "/repo")
	require.NoError(t, err)
	require.Len(t, d.Stashes, 1)
	assert.Equal(t, "WIP", d.Stashes[0].Subject)
}
