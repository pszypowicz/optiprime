package gitops

import "strings"

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
