package gitops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const fetchTimeout = 15 * time.Second

type InProgressOp string

const (
	OpNone        InProgressOp = ""
	OpMerging     InProgressOp = "MERGING"
	OpRebasing    InProgressOp = "REBASING"
	OpCherryPick  InProgressOp = "CHERRY-PICK"
	OpReverting   InProgressOp = "REVERTING"
	OpBisecting   InProgressOp = "BISECTING"
	OpAMSession   InProgressOp = "APPLYING"
)

type Status struct {
	Branch          string
	DefaultBranch   string
	Upstream        string // e.g. "origin/main" - may be empty
	Detached        bool
	BranchIsDefault bool

	Staged    int
	Unstaged  int
	Untracked int
	Conflicts int
	Stashes   int

	Ahead  int // vs Upstream if set, else vs origin/default
	Behind int

	InProgress InProgressOp
	CanFF      bool // behind > 0, ahead == 0, clean, on default
}

func (s Status) Dirty() bool { return s.Staged+s.Unstaged+s.Conflicts > 0 }

func run(dir string, args ...string) (string, string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func DefaultBranch(dir string) string {
	out, _, err := run(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err == nil && out != "" {
		if _, rest, ok := strings.Cut(out, "/"); ok {
			return rest
		}
	}
	if _, _, err := run(dir, "remote", "set-head", "origin", "--auto"); err == nil {
		out, _, err := run(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
		if err == nil && out != "" {
			if _, rest, ok := strings.Cut(out, "/"); ok {
				return rest
			}
		}
	}
	return "main"
}

func Fetch(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "fetch", "--quiet", "--prune", "origin")
	cmd.Dir = dir
	// Prevent hanging on interactive auth prompts from dead SSH remotes.
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=/bin/true",
		"SSH_ASKPASS=/bin/true",
		"GCM_INTERACTIVE=Never",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("fetch timed out (%s)", fetchTimeout)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

func GetStatus(dir string) (Status, error) {
	s := Status{DefaultBranch: DefaultBranch(dir)}

	out, _, err := run(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=normal")
	if err != nil {
		return s, fmt.Errorf("git status: %w", err)
	}
	parsePorcelainV2(out, &s)

	s.BranchIsDefault = !s.Detached && s.Branch == s.DefaultBranch

	// If upstream missing but we know default branch, compare to origin/<default>.
	if s.Upstream == "" && !s.Detached {
		target := "origin/" + s.DefaultBranch
		if ab, _, err := run(dir, "rev-list", "--left-right", "--count", "HEAD..."+target); err == nil {
			var a, b int
			fmt.Sscanf(ab, "%d\t%d", &a, &b)
			s.Ahead, s.Behind = a, b
		}
	}

	s.Stashes = stashCount(dir)
	s.InProgress = detectOp(dir)

	s.CanFF = !s.Detached && s.BranchIsDefault && !s.Dirty() && s.Behind > 0 && s.Ahead == 0

	return s, nil
}

func parsePorcelainV2(out string, s *Status) {
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			name := strings.TrimPrefix(line, "# branch.head ")
			if name == "(detached)" {
				s.Detached = true
				s.Branch = "(detached)"
			} else {
				s.Branch = name
			}
		case strings.HasPrefix(line, "# branch.upstream "):
			s.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab "):
			rest := strings.TrimPrefix(line, "# branch.ab ")
			var a, b int
			fmt.Sscanf(rest, "+%d -%d", &a, &b)
			s.Ahead, s.Behind = a, b
		case line[0] == '1' || line[0] == '2':
			// "1 XY ..." or "2 XY ..."
			if len(line) < 4 {
				continue
			}
			x, y := line[2], line[3]
			if x != '.' {
				s.Staged++
			}
			if y != '.' {
				s.Unstaged++
			}
		case line[0] == 'u':
			s.Conflicts++
		case line[0] == '?':
			s.Untracked++
		}
	}
}

func stashCount(dir string) int {
	out, _, err := run(dir, "stash", "list", "--format=%gd")
	if err != nil || out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}

func detectOp(dir string) InProgressOp {
	gitDir, _, err := run(dir, "rev-parse", "--git-dir")
	if err != nil {
		return OpNone
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(gitDir, rel))
		return err == nil
	}
	switch {
	case exists("rebase-merge"), exists("rebase-apply"):
		if exists("rebase-apply/applying") {
			return OpAMSession
		}
		return OpRebasing
	case exists("MERGE_HEAD"):
		return OpMerging
	case exists("CHERRY_PICK_HEAD"):
		return OpCherryPick
	case exists("REVERT_HEAD"):
		return OpReverting
	case exists("BISECT_LOG"):
		return OpBisecting
	}
	return OpNone
}

func FastForward(dir string) error {
	def := DefaultBranch(dir)
	branch, _, err := run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("rev-parse: %w", err)
	}
	if branch != def {
		return fmt.Errorf("on branch %q, not %q", branch, def)
	}
	_, stderr, err := run(dir, "merge", "--ff-only", "origin/"+def)
	if err != nil {
		return fmt.Errorf("%s: %w", stderr, err)
	}
	return nil
}

func Clone(sshURL, dest string) error {
	cmd := exec.Command("git", "clone", sshURL, dest)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}
