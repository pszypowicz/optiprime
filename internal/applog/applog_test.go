package applog_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pszypowicz/optiprime-sync/internal/applog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath_XDGStateHomeOverrides(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	assert.Equal(t, filepath.Join(dir, "optiprime-sync", "errors.log"), applog.Path())
}

func TestPath_PlatformDefault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	want := filepath.Join(home, ".local", "state", "optiprime-sync", "errors.log")
	if runtime.GOOS == "darwin" {
		want = filepath.Join(home, "Library", "Logs", "optiprime-sync", "errors.log")
	}
	assert.Equal(t, want, applog.Path())
}
