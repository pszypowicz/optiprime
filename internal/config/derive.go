package config

import (
	"net/url"
	"strings"

	"github.com/pszypowicz/optiprime-sync/internal/gitops"
	"github.com/pszypowicz/optiprime-sync/internal/scanner"
)

// adoRemote is an origin URL parsed into ADO coordinates.
type adoRemote struct {
	org     string
	project string
}

// scopeRemotes reads and parses the origin URL of every git repo directly
// under root. Repos without an origin remote and remotes that are not ADO
// URLs are skipped.
func scopeRemotes(root string) []adoRemote {
	repos, err := scanner.FindRepos(root)
	if err != nil {
		return nil
	}
	var out []adoRemote
	for _, repo := range repos {
		raw, err := gitops.RemoteURL(repo.Path)
		if err != nil {
			continue
		}
		if r, ok := parseADORemote(raw); ok {
			out = append(out, r)
		}
	}
	return out
}

// parseADORemote extracts ADO coordinates from a remote URL. Recognized
// forms (an optional <user>@ before the host is accepted):
//
//	https://dev.azure.com/<org>/<project>/_git/<repo>
//	git@ssh.dev.azure.com:v3/<org>/<project>/<repo>
//	ssh://git@ssh.dev.azure.com/v3/<org>/<project>/<repo>
func parseADORemote(raw string) (adoRemote, bool) {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "http://"):
		u, err := url.Parse(raw)
		if err != nil || !strings.EqualFold(u.Hostname(), "dev.azure.com") {
			return adoRemote{}, false
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) != 4 || !strings.EqualFold(parts[2], "_git") {
			return adoRemote{}, false
		}
		return adoRemote{org: parts[0], project: parts[1]}, true

	case strings.HasPrefix(lower, "ssh://"):
		u, err := url.Parse(raw)
		if err != nil || !strings.EqualFold(u.Hostname(), "ssh.dev.azure.com") {
			return adoRemote{}, false
		}
		return parseV3Path(strings.Trim(u.Path, "/"))

	default:
		// scp-like syntax: [user@]host:path
		host, path, ok := strings.Cut(raw, ":")
		if !ok {
			return adoRemote{}, false
		}
		if i := strings.IndexByte(host, '@'); i >= 0 {
			host = host[i+1:]
		}
		if !strings.EqualFold(host, "ssh.dev.azure.com") {
			return adoRemote{}, false
		}
		return parseV3Path(strings.Trim(path, "/"))
	}
}

// parseV3Path parses "v3/<org>/<project>/<repo>". ADO percent-encodes
// spaces in ssh URLs, so each segment is unescaped.
func parseV3Path(path string) (adoRemote, bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 4 || !strings.EqualFold(parts[0], "v3") {
		return adoRemote{}, false
	}
	return adoRemote{org: unescape(parts[1]), project: unescape(parts[2])}, true
}

func unescape(s string) string {
	if u, err := url.PathUnescape(s); err == nil {
		return u
	}
	return s
}
