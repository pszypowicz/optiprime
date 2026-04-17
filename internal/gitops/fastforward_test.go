package gitops

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFastForward_OnDefaultSucceeds(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("main", "", nil, "rev-parse", "--abbrev-ref", "HEAD")
	f.Set("", "", nil, "merge", "--ff-only", "origin/main")

	require.NoError(t, fastForward(f, "/repo"))
}

func TestFastForward_NotOnDefaultRefuses(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("feature/x", "", nil, "rev-parse", "--abbrev-ref", "HEAD")

	err := fastForward(f, "/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `on branch "feature/x"`)
	assert.Contains(t, err.Error(), `not "main"`)
	// Merge must not have been attempted.
	for _, c := range f.calls {
		if len(c.args) > 0 && c.args[0] == "merge" {
			t.Fatalf("merge must not be called when branch != default")
		}
	}
}

func TestFastForward_RevParseFails(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("", "fatal", errors.New("exit 128"), "rev-parse", "--abbrev-ref", "HEAD")

	err := fastForward(f, "/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rev-parse:")
}

func TestFastForward_MergeFails(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("main", "", nil, "rev-parse", "--abbrev-ref", "HEAD")
	f.Set("", "Not possible to fast-forward", errors.New("exit 128"),
		"merge", "--ff-only", "origin/main")

	err := fastForward(f, "/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Not possible to fast-forward")
}
