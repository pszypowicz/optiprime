package gitops

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeResult struct {
	stdout string
	stderr string
	err    error
}

type fakeCall struct {
	dir  string
	env  []string
	args []string
}

// fakeRunner is a GitRunner that returns canned results keyed by the joined
// args string. A queue of results per key is supported: each call pops the
// next result. Unqueued calls fail the test loudly.
type fakeRunner struct {
	t          *testing.T
	queues     map[string][]fakeResult
	calls      []fakeCall
	respectCtx bool
}

func newFakeRunner(t *testing.T) *fakeRunner {
	t.Helper()
	return &fakeRunner{t: t, queues: map[string][]fakeResult{}}
}

func key(args []string) string { return strings.Join(args, " ") }

// Set queues a single result for the exact args. Call multiple times to
// queue successive responses for repeated calls with the same args.
func (f *fakeRunner) Set(stdout, stderr string, err error, args ...string) {
	k := key(args)
	f.queues[k] = append(f.queues[k], fakeResult{stdout: stdout, stderr: stderr, err: err})
}

func (f *fakeRunner) pop(k string, args []string, dir string, env []string) fakeResult {
	f.calls = append(f.calls, fakeCall{dir: dir, env: append([]string(nil), env...), args: append([]string(nil), args...)})
	q, ok := f.queues[k]
	if !ok || len(q) == 0 {
		f.t.Fatalf("unexpected git call: %v (dir=%s)", args, dir)
	}
	res := q[0]
	f.queues[k] = q[1:]
	return res
}

func (f *fakeRunner) Run(dir string, args ...string) (string, string, error) {
	r := f.pop(key(args), args, dir, nil)
	return r.stdout, r.stderr, r.err
}

func (f *fakeRunner) RunCtx(ctx context.Context, dir string, env []string, args ...string) (string, string, error) {
	if f.respectCtx {
		// Block until the caller's context expires or a safety cap elapses.
		// Exists purely to make timeout tests deterministic: a race between
		// a microsecond deadline and the test goroutine would otherwise
		// sometimes see ctx.Err() == nil on entry.
		select {
		case <-ctx.Done():
			f.calls = append(f.calls, fakeCall{dir: dir, env: append([]string(nil), env...), args: append([]string(nil), args...)})
			return "", "", ctx.Err()
		case <-time.After(1 * time.Second):
			// Fall through; treat as a normal call.
		}
	}
	r := f.pop(key(args), args, dir, env)
	return r.stdout, r.stderr, r.err
}
