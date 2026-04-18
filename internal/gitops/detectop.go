package gitops

import (
	"os"
	"path/filepath"
)

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
