package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/pszypowicz/optiprime/internal/applog"
	"github.com/pszypowicz/optiprime/internal/config"
	"github.com/pszypowicz/optiprime/internal/tui"
)

// version is overridable at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(versionString())
		return
	}

	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrScopeUnresolved) {
			fmt.Fprintln(os.Stderr, "optiprime:", err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "optiprime:", err)
		os.Exit(1)
	}

	// Best-effort: if the log can't be opened we still want the TUI to run.
	if logPath, err := applog.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "optiprime: could not open log at %s: %v\n", logPath, err)
	}
	applog.Infof("start", "", "optiprime started; log path=%s", applog.Path())

	if err := tui.Run(cfg); err != nil {
		applog.Errorf("tui", "", "run failed: %v", err)
		fmt.Fprintln(os.Stderr, "optiprime:", err)
		fmt.Fprintln(os.Stderr, "see", applog.Path(), "for details")
		os.Exit(1)
	}
}

func versionString() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	var rev, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				modified = "-dirty"
			}
		}
	}
	if rev == "" {
		return version
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	return version + "-" + rev + modified
}
