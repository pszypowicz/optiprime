package config_test

import (
	"context"
	"encoding/base64"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pszypowicz/optiprime/internal/ado"
	"github.com/pszypowicz/optiprime/internal/config"
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

// setEnv pins every var Load reads, and hides the az CLI, so values leaking
// in from the developer's or CI machine can't change what Load sees.
func setEnv(t *testing.T, org, project, pat string) {
	t.Helper()
	t.Setenv("ADO_ORG", org)
	t.Setenv("ADO_PROJECT", project)
	t.Setenv("AZURE_DEVOPS_EXT_PAT", pat)
	t.Setenv("ADO_TOKEN", "")
	t.Setenv("ADO_TENANT_ID", "")
	config.StubAz(t, false)
}

// authHeader returns the Authorization header the loaded credential sends.
func authHeader(t *testing.T, cfg *config.Config) string {
	t.Helper()
	require.NotNil(t, cfg.Credential)
	h, err := cfg.Credential.Authorization(context.Background())
	require.NoError(t, err)
	return h
}

const jwt = "eyJhbGciOiJub25lIn0.eyJleHAiOjF9.sig"

func basic(pat string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+pat))
}

func TestLoad_AllOverridesSet(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "pat-xyz")
	dir := t.TempDir()
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "foo-org", cfg.Org)
	assert.Equal(t, "bar-project", cfg.Project)
	assert.Equal(t, basic("pat-xyz"), authHeader(t, cfg))
	assert.Equal(t, "AZURE_DEVOPS_EXT_PAT", cfg.AuthSource)
	assert.Equal(t, dir, cfg.ScopeRoot)
}

func TestLoad_NoCredentials_RemoteFeaturesOff(t *testing.T) {
	setEnv(t, "", "", "")
	dir := t.TempDir()
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.False(t, cfg.RemoteEnabled())
	assert.Empty(t, cfg.Org)
	assert.Empty(t, cfg.Project)
	assert.Equal(t, dir, cfg.ScopeRoot)
	assert.Contains(t, cfg.RemoteOff, "AZURE_DEVOPS_EXT_PAT")
	assert.Contains(t, cfg.RemoteOff, "az login")
}

// Without a PAT the org and the project are never used, so Load must not
// fail on remotes that disagree - derivation is skipped entirely.
func TestLoad_NoPAT_SkipsDerivation(t *testing.T) {
	setEnv(t, "", "", "")
	dir := t.TempDir()
	initRepo(t, dir, "repo-a", "git@ssh.dev.azure.com:v3/org-one/proj/a")
	initRepo(t, dir, "repo-b", "git@ssh.dev.azure.com:v3/org-two/proj/b")
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.False(t, cfg.RemoteEnabled())
}

func TestLoad_WithPAT_RemoteFeaturesOn(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "pat-xyz")
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.True(t, cfg.RemoteEnabled())
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

func TestLoad_ADOTokenUsesBearer(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "")
	t.Setenv("ADO_TOKEN", jwt)
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "Bearer "+jwt, authHeader(t, cfg))
	assert.Equal(t, "ADO_TOKEN", cfg.AuthSource)
}

func TestLoad_PATBeatsADOToken(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "pat-xyz")
	t.Setenv("ADO_TOKEN", jwt)
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, basic("pat-xyz"), authHeader(t, cfg))
}

func TestLoad_EnvCredentialBeatsAzLogin(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "pat-xyz")
	config.StubAz(t, true)
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, basic("pat-xyz"), authHeader(t, cfg))
}

func TestLoad_AzLoginWhenNoEnvCredential(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "")
	config.StubAz(t, true)
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.True(t, cfg.RemoteEnabled())
	assert.IsType(t, &ado.AzCredential{}, cfg.Credential)
	assert.Equal(t, "az login", cfg.AuthSource)
	assert.Empty(t, cfg.RemoteOff)
}

func TestLoad_AzLoginDerivesFromRemotes(t *testing.T) {
	setEnv(t, "", "", "")
	config.StubAz(t, true)
	dir := t.TempDir()
	initRepo(t, dir, "repo", sshURL)
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.True(t, cfg.RemoteEnabled())
	assert.Equal(t, "foo-org", cfg.Org)
	assert.Equal(t, "bar-project", cfg.Project)
}

// az login is a fallback, not something the user asked for, so a scope it
// can't resolve means git-only mode rather than a startup failure.
func TestLoad_AzLoginUnresolvedScopeFallsBackToGitOnly(t *testing.T) {
	cases := map[string][]string{
		"no ADO remotes":      nil,
		"conflicting orgs":    {"git@ssh.dev.azure.com:v3/org-one/proj/a", "git@ssh.dev.azure.com:v3/org-two/proj/b"},
		"non-ADO remote only": {"git@github.com:someone/tool.git"},
	}
	for name, remotes := range cases {
		t.Run(name, func(t *testing.T) {
			setEnv(t, "", "", "")
			config.StubAz(t, true)
			dir := t.TempDir()
			for i, r := range remotes {
				initRepo(t, dir, "repo-"+string(rune('a'+i)), r)
			}
			t.Chdir(dir)

			cfg, err := config.Load()
			require.NoError(t, err)

			assert.False(t, cfg.RemoteEnabled())
			assert.Nil(t, cfg.Credential)
			assert.Contains(t, cfg.RemoteOff, "ADO_ORG")
		})
	}
}

func TestLoad_AzLoginRejectsMalformedTenantID(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "")
	config.StubAz(t, true)
	t.Setenv("ADO_TENANT_ID", "not-a-guid")
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "ADO_TENANT_ID")
}

func TestLoad_AzLoginAcceptsTenantID(t *testing.T) {
	setEnv(t, "foo-org", "bar-project", "")
	config.StubAz(t, true)
	t.Setenv("ADO_TENANT_ID", "11111111-2222-3333-4444-555555555555")
	t.Chdir(t.TempDir())

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.IsType(t, &ado.AzCredential{}, cfg.Credential)
}

func TestLoad_AzLoginGitOnlyFallbackKeepsNoDerivedScope(t *testing.T) {
	setEnv(t, "", "", "")
	config.StubAz(t, true)
	dir := t.TempDir()
	initRepo(t, dir, "repo-a", "git@ssh.dev.azure.com:v3/foo-org/proj-one/a")
	initRepo(t, dir, "repo-b", "git@ssh.dev.azure.com:v3/foo-org/proj-two/b")
	t.Chdir(dir)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.False(t, cfg.RemoteEnabled())
	assert.Empty(t, cfg.Org, "a half-derived scope is not shown in git-only mode")
	assert.Empty(t, cfg.Project)
}
