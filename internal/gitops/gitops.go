package gitops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pszypowicz/optiprime-sync/internal/applog"
)

const fetchTimeout = 15 * time.Second

type InProgressOp string

const (
	OpNone       InProgressOp = ""
	OpMerging    InProgressOp = "MERGING"
	OpRebasing   InProgressOp = "REBASING"
	OpCherryPick InProgressOp = "CHERRY-PICK"
	OpReverting  InProgressOp = "REVERTING"
	OpBisecting  InProgressOp = "BISECTING"
	OpAMSession  InProgressOp = "APPLYING"
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

	// MergedInDefault: every commit unique to HEAD has a patch-equivalent
	// commit already in origin/<default>. Only computed on non-default
	// branches. Catches plain merges, squash merges, and rebases.
	MergedInDefault bool

	InProgress InProgressOp
	CanFF      bool // behind > 0, ahead == 0, clean, on default
}

func (s Status) Dirty() bool { return s.Staged+s.Unstaged+s.Conflicts > 0 }

// SafeToUpdate is true when one keystroke can bring this repo to the tip of
// origin/<default> without surprises. On the default branch that's a plain
// fast-forward; on a feature branch it means the work is already upstream,
// so switching to default and ff-merging is safe.
func (s Status) SafeToUpdate() bool {
	if s.InProgress != OpNone || s.Dirty() || s.Detached {
		return false
	}
	if s.BranchIsDefault {
		return s.CanFF
	}
	return s.MergedInDefault
}

func DefaultBranch(dir string) string { return defaultBranch(defaultRunner, dir) }

func defaultBranch(r GitRunner, dir string) string {
	out, _, err := r.Run(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err == nil && out != "" {
		if _, rest, ok := strings.Cut(out, "/"); ok {
			return rest
		}
	}
	if _, _, err := r.Run(dir, "remote", "set-head", "origin", "--auto"); err == nil {
		out, _, err := r.Run(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
		if err == nil && out != "" {
			if _, rest, ok := strings.Cut(out, "/"); ok {
				return rest
			}
		}
	}
	return "main"
}

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
	_, stderr, err := r.RunCtx(ctx, dir, env, "fetch", "--quiet", "--prune", "origin")
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		applog.Errorf("git.fetch", dir, "timed out after %s; stderr=%s", timeout, stderr)
		return fmt.Errorf("fetch timed out (%s)", timeout)
	}
	if err != nil {
		applog.Errorf("git.fetch", dir, "%v; stderr=%s", err, stderr)
		return fmt.Errorf("%s: %w", stderr, err)
	}
	return nil
}

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

func stashCount(r GitRunner, dir string) int {
	out, _, err := r.Run(dir, "stash", "list", "--format=%gd")
	if err != nil || out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}

func detectOp(r GitRunner, dir string) InProgressOp {
	gitDir, _, err := r.Run(dir, "rev-parse", "--git-dir")
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
		return fmt.Errorf("%s: %w", stderr, err)
	}
	return nil
}

// Details is an extended, on-demand view of a repo shown in the info panel.
// Purposefully cheap to compute - only reads local git state.
type Details struct {
	LastCommitSHA     string
	LastCommitSubject string
	LastCommitAuthor  string
	LastCommitAge     string
	DirtyFiles        []DirtyFile
	Stashes           []StashEntry
	UpstreamBranch    string
	RemoteURL         string
}

type DirtyFile struct {
	XY   string // two-char porcelain status (e.g. " M", "??")
	Path string
}

type StashEntry struct {
	Ref     string // stash@{0}
	Subject string
	Age     string // relative
}

func GetDetails(dir string) (Details, error) { return getDetails(defaultRunner, dir) }

func getDetails(r GitRunner, dir string) (Details, error) {
	d := Details{}

	if out, _, err := r.Run(dir, "log", "-1", "--pretty=format:%h\t%s\t%an\t%ar"); err == nil && out != "" {
		parts := strings.SplitN(out, "\t", 4)
		if len(parts) == 4 {
			d.LastCommitSHA = parts[0]
			d.LastCommitSubject = parts[1]
			d.LastCommitAuthor = parts[2]
			d.LastCommitAge = parts[3]
		}
	}

	if out, _, err := r.Run(dir, "status", "--porcelain=v1"); err == nil && out != "" {
		for _, line := range strings.Split(out, "\n") {
			if len(line) < 3 {
				continue
			}
			d.DirtyFiles = append(d.DirtyFiles, DirtyFile{
				XY:   line[:2],
				Path: strings.TrimSpace(line[3:]),
			})
		}
	}

	if out, _, err := r.Run(dir, "stash", "list", "--format=%gd\t%gs\t%cr"); err == nil && out != "" {
		for _, line := range strings.Split(out, "\n") {
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) == 3 {
				d.Stashes = append(d.Stashes, StashEntry{
					Ref:     parts[0],
					Subject: parts[1],
					Age:     parts[2],
				})
			}
		}
	}

	if out, _, err := r.Run(dir, "rev-parse", "--abbrev-ref", "@{upstream}"); err == nil {
		d.UpstreamBranch = out
	}
	if out, _, err := r.Run(dir, "remote", "get-url", "origin"); err == nil {
		d.RemoteURL = out
	}

	return d, nil
}

func Clone(sshURL, dest string) error { return clone(defaultRunner, sshURL, dest) }

func clone(r GitRunner, sshURL, dest string) error {
	_, stderr, err := r.RunCtx(context.Background(), "", nil, "clone", sshURL, dest)
	if err != nil {
		applog.Errorf("git.clone", dest, "%v; stderr=%s", err, stderr)
		return fmt.Errorf("%s: %w", stderr, err)
	}
	return nil
}

