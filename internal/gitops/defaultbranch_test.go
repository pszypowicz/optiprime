package gitops

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultBranch_HappyPath(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("origin/main", "", nil, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	assert.Equal(t, "main", defaultBranch(f, "/repo"))
}

func TestDefaultBranch_SetHeadRecovery(t *testing.T) {
	f := newFakeRunner(t)
	// First symbolic-ref fails.
	f.Set("", "fatal", errors.New("fail"), "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	// set-head --auto succeeds.
	f.Set("", "", nil, "remote", "set-head", "origin", "--auto")
	// Second symbolic-ref (same args as first) returns origin/master.
	f.Set("origin/master", "", nil, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")

	assert.Equal(t, "master", defaultBranch(f, "/repo"))
}

func TestDefaultBranch_BothPathsFail(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("", "", errors.New("fail-1"), "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	f.Set("", "", errors.New("fail-2"), "remote", "set-head", "origin", "--auto")
	assert.Equal(t, "main", defaultBranch(f, "/repo"))
}

func TestDefaultBranch_OutputWithoutSlashFallsThrough(t *testing.T) {
	f := newFakeRunner(t)
	// First symbolic-ref succeeds but output has no "/".
	f.Set("weird-no-slash", "", nil, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	// set-head then second symbolic-ref produce a proper name.
	f.Set("", "", nil, "remote", "set-head", "origin", "--auto")
	f.Set("origin/main", "", nil, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	assert.Equal(t, "main", defaultBranch(f, "/repo"))
}

func TestDefaultBranch_NestedSlashesInOutput(t *testing.T) {
	f := newFakeRunner(t)
	// Cut splits on first "/" only, so "origin/release/1.0" -> "release/1.0".
	f.Set("origin/release/1.0", "", nil, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	assert.Equal(t, "release/1.0", defaultBranch(f, "/repo"))
}
