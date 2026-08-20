package config_test

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pszypowicz/optiprime-sync/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sshURL   = "git@ssh.dev.azure.com:v3/foo-org/bar-project/repo"
	httpsURL = "https://foo-org@dev.azure.com/foo-org/bar-project/_git/repo"
)

// initRepo creates a real git repo under root with the given origin URL,
// so Load exercises the actual scan-and-derive chain.
func initRepo(t *testing.T, root, name, remote string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := filepath.Join(root, name)
	for _, args := range [][]string{
		{"init", "--quiet", dir},
		{"-C", dir, "remote", "add", "origin", remote},
	} {
		out, err := exec.Command("git", args...).CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
}

// setEnv pins all three vars so values leaking in from the developer's or
// CI shell can't change what Load sees.
func setEnv(t *testing.T, org, project, pat string) {
	t.Helper()
	t.Setenv("ADO_ORG", org)
	t.Setenv("ADO_PROJECT", project)
	t.Setenv("AZURE_DEVOPS_EXT_PAT", pat)
}

func TestLoad_AllOverridesSet(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "pat-xyz")
	dir := t.TempDir()
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "foo-org", cfg.Org)
	assert.Equal(t, "bar-project", cfg.Project)
	assert.Equal(t, "pat-xyz", cfg.PAT)
	assert.Equal(t, dir, cfg.ScopeRoot)
}

func TestLoad_MissingPAT(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "")
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t, errors.Is(err, config.ErrMissingEnv))
	assert.Contains(t, err.Error(), "AZURE_DEVOPS_EXT_PAT")
}

func TestLoad_DerivesFromRemotes(t *testing.T) {
	setEnv(t, "", "", "pat-xyz")
	dir := t.TempDir()
	initRepo(t, dir, "repo-ssh", sshURL)
	initRepo(t, dir, "repo-https", httpsURL)
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "foo-org", cfg.Org)
	assert.Equal(t, "bar-project", cfg.Project)
}

func TestLoad_PartialOverrideWins(t *testing.T) {
	setEnv(t, "override-org", "", "pat-xyz")
	dir := t.TempDir()
	initRepo(t, dir, "repo", sshURL)
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "override-org", cfg.Org, "env override beats the derived org")
	assert.Equal(t, "bar-project", cfg.Project, "project still derives from remotes")
}

func TestLoad_NoRemotesNoOverrides(t *testing.T) {
	setEnv(t, "", "", "pat-xyz")
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t, errors.Is(err, config.ErrScopeUnresolved))
	assert.Contains(t, err.Error(), "ADO_ORG")
}

func TestLoad_ConflictingOrgs(t *testing.T) {
	setEnv(t, "", "", "pat-xyz")
	dir := t.TempDir()
	initRepo(t, dir, "repo-a", "git@ssh.dev.azure.com:v3/org-one/proj/a")
	initRepo(t, dir, "repo-b", "git@ssh.dev.azure.com:v3/org-two/proj/b")
	t.Chdir(dir)

	cfg, err := config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t, errors.Is(err, config.ErrScopeUnresolved))
	assert.Contains(t, err.Error(), "org-one")
	assert.Contains(t, err.Error(), "org-two")
}

func TestLoad_NonADORemotesIgnored(t *testing.T) {
	setEnv(t, "", "", "pat-xyz")
	dir := t.TempDir()
	initRepo(t, dir, "github-repo", "git@github.com:someone/tool.git")
	initRepo(t, dir, "ado-repo", sshURL)
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "foo-org", cfg.Org)
	assert.Equal(t, "bar-project", cfg.Project)
}
