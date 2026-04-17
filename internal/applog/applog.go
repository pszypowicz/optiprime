// Package applog writes a JSON-lines error log for inspection from outside
// the TUI. The in-TUI header can only show a single truncated line per
// error, so anything interesting gets recorded here with its full body.
//
// Path resolution:
//   $XDG_STATE_HOME/optiprime-sync/errors.log
//   ~/.local/state/optiprime-sync/errors.log (fallback)
//   $TMPDIR/optiprime-sync/errors.log        (last resort)
package applog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Op      string    `json:"op"`
	Target  string    `json:"target,omitempty"`
	Message string    `json:"message"`
}

var (
	mu   sync.Mutex
	path string
	f    *os.File
)

// Init creates the log file (if missing) and opens it for append. Safe to
// call more than once; subsequent calls are no-ops. Returns the resolved
// log path (even if opening failed, so it can be shown to the user).
func Init() (string, error) {
	dir := stateDir()
	p := filepath.Join(dir, "errors.log")

	mu.Lock()
	defer mu.Unlock()
	if f != nil {
		return path, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return p, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	fh, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return p, fmt.Errorf("open %s: %w", p, err)
	}
	f = fh
	path = p
	return p, nil
}

func stateDir() string {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "optiprime-sync")
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".local", "state", "optiprime-sync")
	}
	return filepath.Join(os.TempDir(), "optiprime-sync")
}

// Path returns the resolved log file path. Safe to call before Init.
func Path() string {
	mu.Lock()
	defer mu.Unlock()
	if path != "" {
		return path
	}
	return filepath.Join(stateDir(), "errors.log")
}

func Errorf(op, target, format string, args ...any) {
	write(Entry{
		Time:    time.Now(),
		Level:   "error",
		Op:      op,
		Target:  target,
		Message: fmt.Sprintf(format, args...),
	})
}

func Warnf(op, target, format string, args ...any) {
	write(Entry{
		Time:    time.Now(),
		Level:   "warn",
		Op:      op,
		Target:  target,
		Message: fmt.Sprintf(format, args...),
	})
}

func Infof(op, target, format string, args ...any) {
	write(Entry{
		Time:    time.Now(),
		Level:   "info",
		Op:      op,
		Target:  target,
		Message: fmt.Sprintf(format, args...),
	})
}

func write(e Entry) {
	mu.Lock()
	defer mu.Unlock()
	if f == nil {
		return
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = f.Write(b)
	_, _ = f.Write([]byte("\n"))
}
