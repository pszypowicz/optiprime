package gitops

import (
	"os"

	"github.com/pszypowicz/optiprime/internal/applog"
)

// TraceEnvVar opts into verbose connection diagnostics for the git
// operations that talk to the remote (fetch, clone). Any value other
// than "", "0", or "false" enables it. Off by default because a single
// traced fetch produces dozens of log lines.
const TraceEnvVar = "OPTIPRIME_GIT_TRACE"

func traceEnabled() bool {
	switch os.Getenv(TraceEnvVar) {
	case "", "0", "false":
		return false
	}
	return true
}

// traceEnv returns the extra environment that makes git and ssh narrate
// their work, or nil when tracing is off. GIT_SSH_COMMAND takes
// precedence over core.sshCommand, so a custom ssh wrapper configured in
// git config is bypassed while tracing.
func traceEnv() []string {
	if !traceEnabled() {
		return nil
	}
	return []string{"GIT_TRACE=1", "GIT_SSH_COMMAND=ssh -v"}
}

// traceLog records the full stderr of a traced command at info level.
// git and ssh write trace output to stderr even when the command
// succeeds, so this runs on every traced call, not just failures.
func traceLog(op, target, stderr string) {
	if !traceEnabled() {
		return
	}
	applog.Infof(op+".trace", target, "stderr=%s", stderr)
}
