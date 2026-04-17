package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/pszypowicz/optiprime-sync/internal/config"
	"github.com/pszypowicz/optiprime-sync/internal/tui"
)

func main() {
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

	if err := tui.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "optiprime-sync:", err)
		os.Exit(1)
	}
}
