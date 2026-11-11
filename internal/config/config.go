package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/pszypowicz/optiprime/internal/ado"
)

type Config struct {
	Org     string
	Project string

	// Credential authenticates the ADO REST calls. Nil means git-only mode.
	Credential ado.Credential
	// AuthSource names where Credential comes from, for the log.
	AuthSource string
	// RemoteOff says why Credential is nil.
	RemoteOff string

	ScopeRoot string
}

// RemoteEnabled reports whether the ADO REST features (repo list, PR
// counts) are available. They need a credential; everything else is plain
// git.
func (c *Config) RemoteEnabled() bool {
	return c.Credential != nil
}

// ErrScopeUnresolved: org or project is neither set in the environment nor
// derivable from the origin remotes of the repos in the scope.
var ErrScopeUnresolved = errors.New("cannot resolve ADO scope")

var azInstalled = func() bool {
	_, err := exec.LookPath("az")
	return err == nil
}

// Load resolves the runtime configuration. No env var is required.
//
// The ADO REST features take the first credential available:
// AZURE_DEVOPS_EXT_PAT, then ADO_TOKEN, then a token minted from the az
// login session. Without any of them the tool runs in git-only mode.
//
// ADO_ORG and ADO_PROJECT act as overrides. When either is unset it is
// derived from the ADO origin remotes of the git repos directly under the
// working directory. Derivation fails when no ADO remote exists or when the
// remotes disagree. That failure is fatal with an env credential, and with
// az login it means git-only mode.
func Load() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve cwd: %w", err)
	}
	cfg := &Config{
		Org:       os.Getenv("ADO_ORG"),
		Project:   os.Getenv("ADO_PROJECT"),
		ScopeRoot: cwd,
	}

	var envVar, envValue string
	for _, name := range []string{"AZURE_DEVOPS_EXT_PAT", "ADO_TOKEN"} {
		if v := os.Getenv(name); v != "" {
			envVar, envValue = name, v
			break
		}
	}

	switch {
	case envVar != "":
		if err := cfg.resolveScope(); err != nil {
			return nil, err
		}
		cfg.Credential = ado.NewEnvCredential(envVar, envValue)
		cfg.AuthSource = envVar

	case azInstalled():
		tenant := os.Getenv("ADO_TENANT_ID")
		if tenant != "" && !ado.IsTenantID(tenant) {
			return nil, fmt.Errorf("ADO_TENANT_ID is not a tenant GUID: %s", tenant)
		}
		org, project := cfg.Org, cfg.Project
		if err := cfg.resolveScope(); err != nil {
			cfg.Org, cfg.Project = org, project
			cfg.RemoteOff = "no single ADO org/project in the remotes, set ADO_ORG and ADO_PROJECT"
			return cfg, nil
		}
		cfg.Credential = ado.NewAzCredential(cfg.Org, tenant)
		cfg.AuthSource = "az login"

	default:
		cfg.RemoteOff = "set AZURE_DEVOPS_EXT_PAT or run az login"
	}
	return cfg, nil
}

// resolveScope fills an unset org or project from the origin remotes of the
// repos in the scope.
func (c *Config) resolveScope() error {
	if c.Org != "" && c.Project != "" {
		return nil
	}
	remotes := scopeRemotes(c.ScopeRoot)
	var err error
	if c.Org == "" {
		c.Org, err = uniqueField(remotes, "organization", "ADO_ORG", c.ScopeRoot,
			func(r adoRemote) string { return r.org })
		if err != nil {
			return err
		}
	}
	if c.Project == "" {
		c.Project, err = uniqueField(remotes, "project", "ADO_PROJECT", c.ScopeRoot,
			func(r adoRemote) string { return r.project })
		if err != nil {
			return err
		}
	}
	return nil
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
