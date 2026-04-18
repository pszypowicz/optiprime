package gitops

import (
	"fmt"
	"strings"
)

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
