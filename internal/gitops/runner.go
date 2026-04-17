package gitops

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

// GitRunner abstracts the execution of `git` subcommands so tests can
// substitute deterministic output without shelling out.
type GitRunner interface {
	// Run executes `git <args...>` inside dir with no timeout and the
	// inherited environment. Returns trimmed stdout, trimmed stderr, err.
	Run(dir string, args ...string) (stdout string, stderr string, err error)

	// RunCtx executes `git <args...>` with a caller-provided context and
	// extra environment variables appended to os.Environ(). dir == ""
	// leaves the child process in the parent's working directory (used
	// by Clone).
	RunCtx(ctx context.Context, dir string, env []string, args ...string) (stdout string, stderr string, err error)
}

type execRunner struct{}

func (execRunner) Run(dir string, args ...string) (string, string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func (execRunner) RunCtx(ctx context.Context, dir string, env []string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

var defaultRunner GitRunner = execRunner{}
