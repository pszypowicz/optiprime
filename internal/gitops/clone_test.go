package gitops

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClone_Success(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("", "", nil, "clone", "git@example.com:org/repo", "/tmp/repo")

	require.NoError(t, clone(f, "git@example.com:org/repo", "/tmp/repo"))

	require.Len(t, f.calls, 1)
	assert.Equal(t, []string{"clone", "git@example.com:org/repo", "/tmp/repo"}, f.calls[0].args)
	assert.Empty(t, f.calls[0].dir, "clone runs in parent cwd, dir must be empty")
}

func TestClone_Failure(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("", "fatal: repository not found", errors.New("exit 128"),
		"clone", "git@example.com:org/nope", "/tmp/nope")

	err := clone(f, "git@example.com:org/nope", "/tmp/nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fatal: repository not found")
}
