package gitops

import (
	"fmt"

	"github.com/pszypowicz/optiprime-sync/internal/applog"
)

func FastForward(dir string) error { return fastForward(defaultRunner, dir) }

func fastForward(r GitRunner, dir string) error {
	def := defaultBranch(r, dir)
	branch, _, err := r.Run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("rev-parse: %w", err)
	}
	if branch != def {
		return fmt.Errorf("on branch %q, not %q", branch, def)
	}
	_, stderr, err := r.Run(dir, "merge", "--ff-only", "origin/"+def)
	if err != nil {
		applog.Errorf("git.ff", dir, "%v; stderr=%s", err, stderr)
		return fmt.Errorf("git merge --ff-only: %s: %w", stderr, err)
	}
	return nil
}
