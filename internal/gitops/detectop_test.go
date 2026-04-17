package gitops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGitDir(t *testing.T) (repoDir, gitDir string, fake *fakeRunner) {
	t.Helper()
	repoDir = t.TempDir()
	gitDir = filepath.Join(repoDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	fake = newFakeRunner(t)
	// rev-parse --git-dir returns the absolute path.
	fake.Set(gitDir, "", nil, "rev-parse", "--git-dir")
	return repoDir, gitDir, fake
}

func TestDetectOp_None(t *testing.T) {
	repo, _, f := setupGitDir(t)
	assert.Equal(t, OpNone, detectOp(f, repo))
}

func TestDetectOp_RebaseMergeDir(t *testing.T) {
	repo, gitDir, f := setupGitDir(t)
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "rebase-merge"), 0o755))
	assert.Equal(t, OpRebasing, detectOp(f, repo))
}

func TestDetectOp_RebaseApplyDirWithoutApplying(t *testing.T) {
	repo, gitDir, f := setupGitDir(t)
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "rebase-apply"), 0o755))
	assert.Equal(t, OpRebasing, detectOp(f, repo))
}

func TestDetectOp_RebaseApplyWithApplyingFile(t *testing.T) {
	repo, gitDir, f := setupGitDir(t)
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "rebase-apply"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "rebase-apply", "applying"), []byte{}, 0o644))
	assert.Equal(t, OpAMSession, detectOp(f, repo))
}

func TestDetectOp_Merging(t *testing.T) {
	repo, gitDir, f := setupGitDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "MERGE_HEAD"), []byte{}, 0o644))
	assert.Equal(t, OpMerging, detectOp(f, repo))
}

func TestDetectOp_CherryPick(t *testing.T) {
	repo, gitDir, f := setupGitDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte{}, 0o644))
	assert.Equal(t, OpCherryPick, detectOp(f, repo))
}

func TestDetectOp_Reverting(t *testing.T) {
	repo, gitDir, f := setupGitDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "REVERT_HEAD"), []byte{}, 0o644))
	assert.Equal(t, OpReverting, detectOp(f, repo))
}

func TestDetectOp_Bisecting(t *testing.T) {
	repo, gitDir, f := setupGitDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "BISECT_LOG"), []byte{}, 0o644))
	assert.Equal(t, OpBisecting, detectOp(f, repo))
}

// Precedence: rebase-merge + MERGE_HEAD simultaneously -> rebase wins
// (switch statement at gitops.go:265 checks rebase first).
func TestDetectOp_PrecedenceRebaseOverMerge(t *testing.T) {
	repo, gitDir, f := setupGitDir(t)
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "rebase-merge"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "MERGE_HEAD"), []byte{}, 0o644))
	assert.Equal(t, OpRebasing, detectOp(f, repo))
}

// When rev-parse returns a relative path (".git"), the function joins it
// with the repo dir to produce an absolute path for Stat.
func TestDetectOp_RelativeGitDirJoinedWithRepoDir(t *testing.T) {
	repo := t.TempDir()
	gitDir := filepath.Join(repo, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "MERGE_HEAD"), []byte{}, 0o644))

	f := newFakeRunner(t)
	f.Set(".git", "", nil, "rev-parse", "--git-dir")

	assert.Equal(t, OpMerging, detectOp(f, repo))
}

// rev-parse itself failing -> OpNone.
func TestDetectOp_RevParseFails(t *testing.T) {
	repo := t.TempDir()
	f := newFakeRunner(t)
	f.Set("", "", errors.New("not a git repo"), "rev-parse", "--git-dir")
	assert.Equal(t, OpNone, detectOp(f, repo))
}
