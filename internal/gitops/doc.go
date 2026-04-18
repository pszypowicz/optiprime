// Package gitops wraps the small set of git subcommands the TUI needs.
// All shelling out goes through GitRunner so tests can substitute
// deterministic output; the default runner uses os/exec.
package gitops
