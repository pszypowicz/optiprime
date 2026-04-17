package gitops

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// primeDefaultBranch configures the fake to resolve default branch to "main".
func primeDefaultBranch(f *fakeRunner) {
	f.Set("origin/main", "", nil, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
}

func TestSwitchAndFF_DirtyTreeRefuses(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	// status --porcelain returns non-empty -> dirty.
	f.Set(" M somefile.go", "", nil, "status", "--porcelain")

	err := switchAndFF(f, "/repo")
	require.Error(t, err)
	assert.Equal(t, "working tree dirty", err.Error())
	// No checkout/merge calls should have followed.
	assert.Len(t, f.calls, 2, "only symbolic-ref + status --porcelain should have been called")
}

func TestSwitchAndFF_LocalDefaultExistsHappyPath(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("", "", nil, "status", "--porcelain")                                   // clean
	f.Set("", "", nil, "show-ref", "--verify", "--quiet", "refs/heads/main")     // exists
	f.Set("", "", nil, "checkout", "main")                                        // checkout succeeds
	f.Set("", "", nil, "merge", "--ff-only", "origin/main")                       // ff succeeds

	require.NoError(t, switchAndFF(f, "/repo"))
	// Verify checkout -b was NOT called.
	for _, c := range f.calls {
		if len(c.args) >= 2 && c.args[0] == "checkout" && c.args[1] == "-b" {
			t.Fatalf("checkout -b must not be called when local default exists")
		}
	}
}

func TestSwitchAndFF_LocalDefaultMissingCreatesTracking(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("", "", nil, "status", "--porcelain")
	// show-ref fails -> local default missing.
	f.Set("", "", errors.New("not a ref"), "show-ref", "--verify", "--quiet", "refs/heads/main")
	f.Set("", "", nil, "checkout", "-b", "main", "--track", "origin/main")
	f.Set("", "", nil, "merge", "--ff-only", "origin/main")

	require.NoError(t, switchAndFF(f, "/repo"))
}

func TestSwitchAndFF_CheckoutBFails(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("", "", nil, "status", "--porcelain")
	f.Set("", "", errors.New("not a ref"), "show-ref", "--verify", "--quiet", "refs/heads/main")
	f.Set("", "could not create", errors.New("exit 1"),
		"checkout", "-b", "main", "--track", "origin/main")

	err := switchAndFF(f, "/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create local main")
}

func TestSwitchAndFF_CheckoutFails(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("", "", nil, "status", "--porcelain")
	f.Set("", "", nil, "show-ref", "--verify", "--quiet", "refs/heads/main")
	f.Set("", "conflict", errors.New("exit 1"), "checkout", "main")

	err := switchAndFF(f, "/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checkout main")
}

func TestSwitchAndFF_MergeFFFails(t *testing.T) {
	f := newFakeRunner(t)
	primeDefaultBranch(f)
	f.Set("", "", nil, "status", "--porcelain")
	f.Set("", "", nil, "show-ref", "--verify", "--quiet", "refs/heads/main")
	f.Set("", "", nil, "checkout", "main")
	f.Set("", "Not possible to fast-forward", errors.New("exit 128"),
		"merge", "--ff-only", "origin/main")

	err := switchAndFF(f, "/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ff main")
	assert.Contains(t, err.Error(), "Not possible to fast-forward")
}
