package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pszypowicz/optiprime-sync/internal/applog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var traceVars = []string{"GIT_TRACE=1", "GIT_SSH_COMMAND=ssh -v"}

func TestTraceEnabled(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"1", true},
		{"true", true},
		{"yes", true},
	}
	for _, tc := range cases {
		t.Run("value="+tc.value, func(t *testing.T) {
			t.Setenv(TraceEnvVar, tc.value)
			assert.Equal(t, tc.want, traceEnabled())
		})
	}
}

func TestFetch_TraceOptIn(t *testing.T) {
	t.Setenv(TraceEnvVar, "1")
	f := newFakeRunner(t)
	f.Set("", "", nil, "fetch", "--quiet", "--prune", "origin")

	require.NoError(t, fetch(f, "/repo"))

	require.Len(t, f.calls, 1)
	for _, want := range traceVars {
		assert.Contains(t, f.calls[0].env, want)
	}
}

func TestFetch_TraceOffByDefault(t *testing.T) {
	t.Setenv(TraceEnvVar, "")
	f := newFakeRunner(t)
	f.Set("", "", nil, "fetch", "--quiet", "--prune", "origin")

	require.NoError(t, fetch(f, "/repo"))

	require.Len(t, f.calls, 1)
	for _, unwanted := range traceVars {
		assert.NotContains(t, f.calls[0].env, unwanted)
	}
}

func TestClone_TraceOptIn(t *testing.T) {
	t.Setenv(TraceEnvVar, "1")
	f := newFakeRunner(t)
	f.Set("", "", nil, "clone", "git@host:repo", "/dest")

	require.NoError(t, clone(f, "git@host:repo", "/dest"))

	require.Len(t, f.calls, 1)
	for _, want := range traceVars {
		assert.Contains(t, f.calls[0].env, want)
	}
}

func TestClone_TraceOffByDefault(t *testing.T) {
	t.Setenv(TraceEnvVar, "")
	f := newFakeRunner(t)
	f.Set("", "", nil, "clone", "git@host:repo", "/dest")

	require.NoError(t, clone(f, "git@host:repo", "/dest"))

	require.Len(t, f.calls, 1)
	assert.Empty(t, f.calls[0].env)
}

// TestFetch_TraceIntegration exercises the whole chain with a real git
// process: trace env reaches the child, git writes trace output to stderr,
// and the output lands in the applog file. Uses a local bare repo as
// origin so no network or ssh agent is involved.
func TestFetch_TraceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("real git process")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	t.Setenv(TraceEnvVar, "1")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	logPath, err := applog.Init()
	require.NoError(t, err)

	origin := filepath.Join(t.TempDir(), "origin.git")
	work := filepath.Join(t.TempDir(), "work")
	gitMust(t, "", "init", "--bare", origin)
	gitMust(t, "", "init", work)
	gitMust(t, work, "remote", "add", "origin", origin)

	require.NoError(t, Fetch(work))

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"op":"git.fetch.trace"`)
	assert.Contains(t, string(data), "trace: ", "GIT_TRACE output should be captured")
}

func gitMust(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}
