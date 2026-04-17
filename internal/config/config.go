package config

import (
	"errors"
	"fmt"
	"os"
)

type Config struct {
	Org     string
	Project string
	PAT     string

	ScopeRoot string
}

var ErrMissingEnv = errors.New("required env var not set")

func Load() (*Config, error) {
	org := os.Getenv("ADO_ORG")
	project := os.Getenv("ADO_PROJECT")
	pat := os.Getenv("AZURE_DEVOPS_EXT_PAT")

	var missing []string
	if org == "" {
		missing = append(missing, "ADO_ORG")
	}
	if project == "" {
		missing = append(missing, "ADO_PROJECT")
	}
	if pat == "" {
		missing = append(missing, "AZURE_DEVOPS_EXT_PAT")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %v", ErrMissingEnv, missing)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve cwd: %w", err)
	}

	return &Config{
		Org:       org,
		Project:   project,
		PAT:       pat,
		ScopeRoot: cwd,
	}, nil
}
