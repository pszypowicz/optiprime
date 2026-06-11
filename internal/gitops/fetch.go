package gitops

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pszypowicz/optiprime-sync/internal/applog"
)

const fetchTimeout = 15 * time.Second

func Fetch(dir string) error { return fetch(defaultRunner, dir) }

func fetch(r GitRunner, dir string) error {
	return fetchWith(r, dir, fetchTimeout)
}

func fetchWith(r GitRunner, dir string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	env := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=/bin/true",
		"SSH_ASKPASS=/bin/true",
		"GCM_INTERACTIVE=Never",
	}
	env = append(env, traceEnv()...)
	_, stderr, err := r.RunCtx(ctx, dir, env, "fetch", "--quiet", "--prune", "origin")
	traceLog("git.fetch", dir, stderr)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		applog.Errorf("git.fetch", dir, "timed out after %s; stderr=%s", timeout, stderr)
		return fmt.Errorf("fetch timed out (%s)", timeout)
	}
	if err != nil {
		applog.Errorf("git.fetch", dir, "%v; stderr=%s", err, stderr)
		return fmt.Errorf("git fetch: %s: %w", stderr, err)
	}
	return nil
}
