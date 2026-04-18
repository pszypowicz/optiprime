package gitops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus_Dirty(t *testing.T) {
	cases := []struct {
		name string
		s    Status
		want bool
	}{
		{"zero", Status{}, false},
		{"staged", Status{Staged: 1}, true},
		{"unstaged", Status{Unstaged: 1}, true},
		{"conflicts", Status{Conflicts: 1}, true},
		{"untracked only not dirty", Status{Untracked: 5}, false},
		{"stashes only not dirty", Status{Stashes: 5}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.Dirty())
		})
	}
}

func TestStatus_UpdateAction(t *testing.T) {
	cases := []struct {
		name string
		s    Status
		want UpdateAction
	}{
		{
			name: "in-progress op blocks",
			s:    Status{BranchIsDefault: true, CanFF: true, InProgress: OpMerging},
			want: UpdateSkip,
		},
		{
			name: "dirty blocks",
			s:    Status{BranchIsDefault: true, CanFF: true, Unstaged: 1},
			want: UpdateSkip,
		},
		{
			name: "detached blocks",
			s:    Status{Detached: true, BranchIsDefault: false, MergedInDefault: true},
			want: UpdateSkip,
		},
		{
			name: "default branch + CanFF",
			s:    Status{BranchIsDefault: true, CanFF: true},
			want: UpdateFastForward,
		},
		{
			name: "default branch + !CanFF",
			s:    Status{BranchIsDefault: true, CanFF: false},
			want: UpdateSkip,
		},
		{
			name: "feature + merged upstream",
			s:    Status{BranchIsDefault: false, MergedInDefault: true},
			want: UpdateSwitchAndFF,
		},
		{
			name: "feature + not merged",
			s:    Status{BranchIsDefault: false, MergedInDefault: false},
			want: UpdateSkip,
		},
		{
			name: "feature + merged but in-progress gate wins",
			s:    Status{BranchIsDefault: false, MergedInDefault: true, InProgress: OpRebasing},
			want: UpdateSkip,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.UpdateAction())
			// SafeToUpdate must agree: true iff UpdateAction != UpdateSkip.
			assert.Equal(t, tc.want != UpdateSkip, tc.s.SafeToUpdate())
		})
	}
}
