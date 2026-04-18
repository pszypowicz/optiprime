package gitops

import (
	"fmt"
	"strings"
)

// SwitchAndFF checks out the default branch (creating a tracking branch if
// needed) and fast-forwards it to origin/<default>. Refuses to run on a
// dirty working tree so the switch can't silently clobber user edits.
func SwitchAndFF(dir string) error { return switchAndFF(defaultRunner, dir) }

func switchAndFF(r GitRunner, dir string) error {
	def := defaultBranch(r, dir)

	out, _, _ := r.Run(dir, "status", "--porcelain")
	if len(out) > 0 {
		return fmt.Errorf("working tree dirty")
	}

	// Local <default> may not exist (user never checked it out). In that
	// case create a tracking branch from origin/<default>.
	if _, _, err := r.Run(dir, "show-ref", "--verify", "--quiet", "refs/heads/"+def); err != nil {
		if _, stderr, err := r.Run(dir, "checkout", "-b", def, "--track", "origin/"+def); err != nil {
			return fmt.Errorf("create local %s: %s: %w", def, strings.TrimSpace(stderr), err)
		}
	} else {
		if _, stderr, err := r.Run(dir, "checkout", def); err != nil {
			return fmt.Errorf("checkout %s: %s: %w", def, strings.TrimSpace(stderr), err)
		}
	}

	if _, stderr, err := r.Run(dir, "merge", "--ff-only", "origin/"+def); err != nil {
		return fmt.Errorf("ff %s: %s: %w", def, strings.TrimSpace(stderr), err)
	}
	return nil
}
