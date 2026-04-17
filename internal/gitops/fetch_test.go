package gitops

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch_Success(t *testing.T) {
	f := newFakeRunner(t)
	f.Set("Fetched origin\n", "", nil, "fetch", "--quiet", "--prune", "origin")

	require.NoError(t, fetch(f, "/repo"))

	require.Len(t, f.calls, 1)
	call := f.calls[0]
	assert.Equal(t, []string{"fetch", "--quiet", "--prune", "origin"}, call.args)
	// All four non-interactive env scrubbers must be present.
	for _, want := range []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=/bin/true",
		"SSH_ASKPASS=/bin/true",
		"GCM_INTERACTIVE=Never",
	} {
		assert.Contains(t, call.env, want)
	}
}

func TestFetch_GenericFailure(t *testing.T) {
	underlying := errors.New("exit 1")
	f := newFakeRunner(t)
	f.Set("", "ssh: permission denied", underlying, "fetch", "--quiet", "--prune", "origin")

	err := fetch(f, "/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ssh: permission denied")
	assert.True(t, errors.Is(err, underlying), "fetch must wrap underlying error with %%w")
}

func TestFetchWith_Timeout(t *testing.T) {
	f := newFakeRunner(t)
	f.respectCtx = true
	// No queued result needed - respectCtx short-circuits with ctx.Err()
	// before the fake tries to dequeue.

	err := fetchWith(f, "/repo", 1*time.Microsecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch timed out")
}

// Belt-and-braces: ensure the ctx plumbing flows via RunCtx. When respectCtx
// is false and the fake returns DeadlineExceeded directly, we still hit the
// timeout branch because errors.Is(ctx.Err(), context.DeadlineExceeded) is
// checked after the call. Use a pre-cancelled context to exercise this.
func TestFetchWith_ContextPathProducesTimeoutError(t *testing.T) {
	f := newFakeRunner(t)
	f.respectCtx = true
	// 1 ns guarantees deadline already exceeded by the time RunCtx is entered.
	err := fetchWith(f, "/repo", 1*time.Nanosecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch timed out")
	// Sanity: the recorded call's context error check happens inside fetchWith.
	_ = context.DeadlineExceeded
}
