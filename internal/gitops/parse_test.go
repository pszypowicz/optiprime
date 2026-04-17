package gitops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePorcelainV2(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Status
	}{
		{
			name: "empty",
			in:   "",
			want: Status{},
		},
		{
			name: "normal branch with upstream and ab",
			in: "# branch.oid abc\n" +
				"# branch.head main\n" +
				"# branch.upstream origin/main\n" +
				"# branch.ab +2 -3\n",
			want: Status{Branch: "main", Upstream: "origin/main", Ahead: 2, Behind: 3},
		},
		{
			name: "detached HEAD",
			in:   "# branch.head (detached)\n",
			want: Status{Branch: "(detached)", Detached: true},
		},
		{
			name: "staged only",
			in: "# branch.head main\n" +
				"1 M. N... 100644 100644 100644 1 1 a.go\n",
			want: Status{Branch: "main", Staged: 1},
		},
		{
			name: "unstaged only",
			in: "# branch.head main\n" +
				"1 .M N... 100644 100644 100644 1 1 a.go\n",
			want: Status{Branch: "main", Unstaged: 1},
		},
		{
			name: "both staged and unstaged on same file",
			in: "# branch.head main\n" +
				"1 MM N... 100644 100644 100644 1 1 a.go\n",
			want: Status{Branch: "main", Staged: 1, Unstaged: 1},
		},
		{
			name: "rename line prefix 2 increments staged",
			in: "# branch.head main\n" +
				"2 R. N... 100644 100644 100644 1 1 R100 new\told\n",
			want: Status{Branch: "main", Staged: 1},
		},
		{
			name: "untracked",
			in: "# branch.head main\n" +
				"? weird.txt\n",
			want: Status{Branch: "main", Untracked: 1},
		},
		{
			name: "conflict",
			in: "# branch.head main\n" +
				"u UU N... 100644 100644 100644 100644 a b c d conflict.go\n",
			want: Status{Branch: "main", Conflicts: 1},
		},
		{
			name: "combined multi-line",
			in: "# branch.head main\n" +
				"# branch.upstream origin/main\n" +
				"# branch.ab +0 -1\n" +
				"1 M. N... 100644 100644 100644 1 1 a.go\n" +
				"1 .M N... 100644 100644 100644 1 1 b.go\n" +
				"u UU N... 100644 100644 100644 100644 x y z w c.go\n" +
				"? d.go\n",
			want: Status{
				Branch:    "main",
				Upstream:  "origin/main",
				Behind:    1,
				Staged:    1,
				Unstaged:  1,
				Conflicts: 1,
				Untracked: 1,
			},
		},
		{
			name: "malformed short line skipped",
			in: "# branch.head main\n" +
				"1 X\n",
			want: Status{Branch: "main"},
		},
		{
			name: "no branch.ab keeps zero ahead/behind",
			in: "# branch.head main\n" +
				"# branch.upstream origin/main\n",
			want: Status{Branch: "main", Upstream: "origin/main"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got Status
			parsePorcelainV2(tc.in, &got)
			assert.Equal(t, tc.want, got)
		})
	}
}
