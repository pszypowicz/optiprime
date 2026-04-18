package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/pszypowicz/optiprime-sync/internal/applog"
	"github.com/pszypowicz/optiprime-sync/internal/config"
	"github.com/pszypowicz/optiprime-sync/internal/tui"
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
		if errors.Is(err, config.ErrMissingEnv) {
			fmt.Fprintln(os.Stderr, "optiprime-sync:", err)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Required env vars:")
			fmt.Fprintln(os.Stderr, "  ADO_ORG               - Azure DevOps organisation name")
			fmt.Fprintln(os.Stderr, "  ADO_PROJECT           - Azure DevOps project name")
			fmt.Fprintln(os.Stderr, "  AZURE_DEVOPS_EXT_PAT  - Personal Access Token with Code (read) scope")
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "optiprime-sync:", err)
		os.Exit(1)
	}

	// Best-effort: if the log can't be opened we still want the TUI to run.
	if logPath, err := applog.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "optiprime-sync: could not open log at %s: %v\n", logPath, err)
	}
	applog.Infof("start", "", "optiprime-sync started; log path=%s", applog.Path())

	if err := tui.Run(cfg); err != nil {
		applog.Errorf("tui", "", "run failed: %v", err)
		fmt.Fprintln(os.Stderr, "optiprime-sync:", err)
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
