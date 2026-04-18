package gitops

import (
	"fmt"
	"strings"
)

func GetStatus(dir string) (Status, error) { return getStatus(defaultRunner, dir) }

func getStatus(r GitRunner, dir string) (Status, error) {
	s := Status{DefaultBranch: defaultBranch(r, dir)}

	out, _, err := r.Run(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=normal")
	if err != nil {
		return s, fmt.Errorf("git status: %w", err)
	}
	parsePorcelainV2(out, &s)

	s.BranchIsDefault = !s.Detached && s.Branch == s.DefaultBranch

	// If upstream missing but we know default branch, compare to origin/<default>.
	if s.Upstream == "" && !s.Detached {
		target := "origin/" + s.DefaultBranch
		if ab, _, err := r.Run(dir, "rev-list", "--left-right", "--count", "HEAD..."+target); err == nil {
			var a, b int
			fmt.Sscanf(ab, "%d\t%d", &a, &b)
			s.Ahead, s.Behind = a, b
		}
	}

	s.Stashes = stashCount(r, dir)
	s.InProgress = detectOp(r, dir)

	s.CanFF = !s.Detached && s.BranchIsDefault && !s.Dirty() && s.Behind > 0 && s.Ahead == 0

	// On a feature branch, check whether every unique commit on HEAD already
	// has a patch-equivalent commit in origin/<default>. git cherry marks
	// upstream-present commits with "- <sha>" and missing ones with "+ <sha>".
	if !s.Detached && !s.BranchIsDefault {
		target := "origin/" + s.DefaultBranch
		if out, _, err := r.Run(dir, "cherry", target, "HEAD"); err == nil {
			merged := true
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "+ ") {
					merged = false
					break
				}
			}
			s.MergedInDefault = merged
		}
	}

	return s, nil
}

func stashCount(r GitRunner, dir string) int {
	out, _, err := r.Run(dir, "stash", "list", "--format=%gd")
	if err != nil || out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}
