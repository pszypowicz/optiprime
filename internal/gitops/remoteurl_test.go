package gitops

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteURL_Success(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("git@ssh.dev.azure.com:v3/org/proj/repo", "", nil, "remote", "get-url", "origin")

	url, err := remoteURLOf(f, "/repo")
	require.NoError(t, err)
	assert.Equal(t, "git@ssh.dev.azure.com:v3/org/proj/repo", url)
}

func TestRemoteURL_NoOrigin(t *testing.T) {
	underlying := errors.New("exit 2")
	f := newFakeRunner(t)
	f.Set("", "error: No such remote 'origin'", underlying, "remote", "get-url", "origin")

	_, err := remoteURLOf(f, "/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No such remote")
	assert.True(t, errors.Is(err, underlying))
}
