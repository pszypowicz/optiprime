package gitops

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
	return s.UpdateAction() != UpdateSkip
}

// UpdateAction is the operation a "u" keystroke would dispatch for this repo.
type UpdateAction int

const (
	// UpdateSkip: not safe to touch, leave it alone.
	UpdateSkip UpdateAction = iota
	// UpdateFastForward: on the default branch, behind upstream, clean -
	// run `git merge --ff-only origin/<default>`.
	UpdateFastForward
	// UpdateSwitchAndFF: on a feature branch whose unique commits are
	// already in origin/<default> - check out default and ff-merge.
	UpdateSwitchAndFF
)

// UpdateAction encodes the dispatch decision SafeToUpdate makes implicitly:
// which command to run when the user picks this repo for an update.
func (s Status) UpdateAction() UpdateAction {
	if s.InProgress != OpNone || s.Dirty() || s.Detached {
		return UpdateSkip
	}
	if s.BranchIsDefault {
		if s.CanFF {
			return UpdateFastForward
		}
		return UpdateSkip
	}
	if s.MergedInDefault {
		return UpdateSwitchAndFF
	}
	return UpdateSkip
}
