package gitops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupStatusBaseline pre-seeds a fake runner with "neutral" responses for
// every call that getStatus makes besides the ones a specific test wants
// to vary. Tests can then override individual queues by calling Set again.
func setupStatusBaseline(t *testing.T, f *fakeRunner, repo, gitDir string) {
	t.Helper()
	// DefaultBranch -> "main".
	f.Set("origin/main", "", nil, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	// Stashes: none.
	f.Set("", "", nil, "stash", "list", "--format=%gd")
	// detectOp: rev-parse returns gitDir (absolute), nothing in it.
	f.Set(gitDir, "", nil, "rev-parse", "--git-dir")
}

func TestGetStatus_CleanDefaultBehindTwo(t *testing.T) {
	repo := t.TempDir()
	gitDir := filepath.Join(repo, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))

	f := newFakeRunner(t)
	setupStatusBaseline(t, f, repo, gitDir)
	// Main branch, upstream origin/main, behind by 2.
	f.Set(
		"# branch.head main\n# branch.upstream origin/main\n# branch.ab +0 -2\n",
		"", nil,
		"status", "--porcelain=v2", "--branch", "--untracked-files=normal",
	)

	s, err := getStatus(f, repo)
	require.NoError(t, err)
	assert.Equal(t, "main", s.Branch)
	assert.Equal(t, "main", s.DefaultBranch)
	assert.True(t, s.BranchIsDefault)
	assert.True(t, s.CanFF)
	assert.Equal(t, 2, s.Behind)
	assert.False(t, s.MergedInDefault, "MergedInDefault not computed on default branch")
}

func TestGetStatus_FeatureMergedInDefault(t *testing.T) {
	repo := t.TempDir()
	gitDir := filepath.Join(repo, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))

	f := newFakeRunner(t)
	setupStatusBaseline(t, f, repo, gitDir)
	f.Set(
		"# branch.head feature/x\n# branch.upstream origin/feature/x\n# branch.ab +3 -0\n",
		"", nil,
		"status", "--porcelain=v2", "--branch", "--untracked-files=normal",
	)
	// cherry: all commits marked "-" (present upstream).
	f.Set("- abc\n- def\n- ghi", "", nil, "cherry", "origin/main", "HEAD")

	s, err := getStatus(f, repo)
	require.NoError(t, err)
	assert.False(t, s.BranchIsDefault)
	assert.True(t, s.MergedInDefault)
	assert.True(t, s.SafeToUpdate())
}

func TestGetStatus_FeatureNotMerged(t *testing.T) {
	repo := t.TempDir()
	gitDir := filepath.Join(repo, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))

	f := newFakeRunner(t)
	setupStatusBaseline(t, f, repo, gitDir)
	f.Set(
		"# branch.head feature/y\n# branch.upstream origin/feature/y\n# branch.ab +2 -0\n",
		"", nil,
		"status", "--porcelain=v2", "--branch", "--untracked-files=normal",
	)
	// At least one commit marked "+" -> not fully merged.
	f.Set("- abc\n+ def", "", nil, "cherry", "origin/main", "HEAD")

	s, err := getStatus(f, repo)
	require.NoError(t, err)
	assert.False(t, s.MergedInDefault)
	assert.False(t, s.SafeToUpdate())
}

func TestGetStatus_NoUpstreamFallsBackToRevList(t *testing.T) {
	repo := t.TempDir()
	gitDir := filepath.Join(repo, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))

	f := newFakeRunner(t)
	setupStatusBaseline(t, f, repo, gitDir)
	// No upstream line in porcelain output.
	f.Set(
		"# branch.head main\n",
		"", nil,
		"status", "--porcelain=v2", "--branch", "--untracked-files=normal",
	)
	// rev-list fallback returns "1\t4".
	f.Set("1\t4", "", nil, "rev-list", "--left-right", "--count", "HEAD...origin/main")

	s, err := getStatus(f, repo)
	require.NoError(t, err)
	assert.Equal(t, "", s.Upstream)
	assert.Equal(t, 1, s.Ahead)
	assert.Equal(t, 4, s.Behind)
}

func TestGetStatus_DetachedHEAD(t *testing.T) {
	repo := t.TempDir()
	gitDir := filepath.Join(repo, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))

	f := newFakeRunner(t)
	setupStatusBaseline(t, f, repo, gitDir)
	f.Set(
		"# branch.head (detached)\n",
		"", nil,
		"status", "--porcelain=v2", "--branch", "--untracked-files=normal",
	)
	// No cherry call, no rev-list fallback (both gated on !Detached).

	s, err := getStatus(f, repo)
	require.NoError(t, err)
	assert.True(t, s.Detached)
	assert.False(t, s.BranchIsDefault)
	assert.False(t, s.CanFF)
	assert.False(t, s.MergedInDefault)
	assert.False(t, s.SafeToUpdate())
}

func TestGetStatus_ActiveRebase(t *testing.T) {
	repo := t.TempDir()
	gitDir := filepath.Join(repo, ".git")
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "rebase-merge"), 0o755))

	f := newFakeRunner(t)
	setupStatusBaseline(t, f, repo, gitDir)
	f.Set(
		"# branch.head main\n# branch.upstream origin/main\n# branch.ab +0 -1\n",
		"", nil,
		"status", "--porcelain=v2", "--branch", "--untracked-files=normal",
	)

	s, err := getStatus(f, repo)
	require.NoError(t, err)
	assert.Equal(t, OpRebasing, s.InProgress)
	assert.False(t, s.SafeToUpdate(), "in-progress op should block SafeToUpdate")
}

func TestGetStatus_GitStatusFails(t *testing.T) {
	repo := t.TempDir()
	f := newFakeRunner(t)
	f.Set("origin/main", "", nil, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	f.Set("", "fatal: not a git repo", errors.New("exit 128"),
		"status", "--porcelain=v2", "--branch", "--untracked-files=normal")

	_, err := getStatus(f, repo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git status:")
}
