package gitops

import "strings"

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
