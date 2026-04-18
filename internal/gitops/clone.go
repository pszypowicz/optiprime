package gitops

import (
	"context"
	"fmt"

	"github.com/pszypowicz/optiprime-sync/internal/applog"
)

func Clone(sshURL, dest string) error { return clone(defaultRunner, sshURL, dest) }

func clone(r GitRunner, sshURL, dest string) error {
	_, stderr, err := r.RunCtx(context.Background(), "", nil, "clone", sshURL, dest)
	if err != nil {
		applog.Errorf("git.clone", dest, "%v; stderr=%s", err, stderr)
		return fmt.Errorf("git clone: %s: %w", stderr, err)
	}
	return nil
}
