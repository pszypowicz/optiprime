package gitops

import "fmt"

// RemoteURL returns the URL of the origin remote of the repo at dir.
func RemoteURL(dir string) (string, error) { return remoteURLOf(defaultRunner, dir) }

func remoteURLOf(r GitRunner, dir string) (string, error) {
	out, stderr, err := r.Run(dir, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("git remote get-url origin: %s: %w", stderr, err)
	}
	return out, nil
}
