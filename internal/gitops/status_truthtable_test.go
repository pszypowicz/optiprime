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

func TestStatus_SafeToUpdate(t *testing.T) {
	cases := []struct {
		name string
		s    Status
		want bool
	}{
		{
			name: "in-progress op blocks",
			s:    Status{BranchIsDefault: true, CanFF: true, InProgress: OpMerging},
			want: false,
		},
		{
			name: "dirty blocks",
			s:    Status{BranchIsDefault: true, CanFF: true, Unstaged: 1},
			want: false,
		},
		{
			name: "detached blocks",
			s:    Status{Detached: true, BranchIsDefault: false, MergedInDefault: true},
			want: false,
		},
		{
			name: "default branch + CanFF",
			s:    Status{BranchIsDefault: true, CanFF: true},
			want: true,
		},
		{
			name: "default branch + !CanFF",
			s:    Status{BranchIsDefault: true, CanFF: false},
			want: false,
		},
		{
			name: "feature + merged upstream",
			s:    Status{BranchIsDefault: false, MergedInDefault: true},
			want: true,
		},
		{
			name: "feature + not merged",
			s:    Status{BranchIsDefault: false, MergedInDefault: false},
			want: false,
		},
		{
			name: "feature + merged but in-progress gate wins",
			s:    Status{BranchIsDefault: false, MergedInDefault: true, InProgress: OpRebasing},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.SafeToUpdate())
		})
	}
}
