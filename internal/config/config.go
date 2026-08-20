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

// ErrScopeUnresolved: org or project is neither set in the environment nor
// derivable from the origin remotes of the repos in the scope.
var ErrScopeUnresolved = errors.New("cannot resolve ADO scope")

// Load resolves the runtime configuration. Only AZURE_DEVOPS_EXT_PAT is
// required. ADO_ORG and ADO_PROJECT act as overrides; when either is unset
// it is derived from the ADO origin remotes of the git repos directly under
// the working directory. Derivation fails when no ADO remote exists or when
// the remotes disagree.
func Load() (*Config, error) {
	pat := os.Getenv("AZURE_DEVOPS_EXT_PAT")
	if pat == "" {
		return nil, fmt.Errorf("%w: [AZURE_DEVOPS_EXT_PAT]", ErrMissingEnv)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve cwd: %w", err)
	}

	org := os.Getenv("ADO_ORG")
	project := os.Getenv("ADO_PROJECT")

	if org == "" || project == "" {
		remotes := scopeRemotes(cwd)
		if org == "" {
			org, err = uniqueField(remotes, "organization", "ADO_ORG", cwd,
				func(r adoRemote) string { return r.org })
			if err != nil {
				return nil, err
			}
		}
		if project == "" {
			project, err = uniqueField(remotes, "project", "ADO_PROJECT", cwd,
				func(r adoRemote) string { return r.project })
			if err != nil {
				return nil, err
			}
		}
	}

	return &Config{
		Org:       org,
		Project:   project,
		PAT:       pat,
		ScopeRoot: cwd,
	}, nil
}

// uniqueField extracts the single value the scope's remotes agree on.
// Zero remotes or disagreement is an ErrScopeUnresolved error that names
// the override env var to set.
func uniqueField(remotes []adoRemote, what, envName, root string, get func(adoRemote) string) (string, error) {
	var distinct []string
	seen := map[string]bool{}
	for _, r := range remotes {
		v := get(r)
		if !seen[v] {
			seen[v] = true
			distinct = append(distinct, v)
		}
	}
	switch len(distinct) {
	case 1:
		return distinct[0], nil
	case 0:
		return "", fmt.Errorf("%w: no ADO origin remote found in the repos under %s; set %s or run in a directory of ADO clones",
			ErrScopeUnresolved, root, envName)
	default:
		return "", fmt.Errorf("%w: the origin remotes under %s name more than one %s (%v); set %s",
			ErrScopeUnresolved, root, what, distinct, envName)
	}
}
