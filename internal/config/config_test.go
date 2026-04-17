package config_test

import (
	"errors"
	"os"
	"testing"

	"github.com/pszypowicz/optiprime-sync/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Success(t *testing.T) {
	t.Setenv("ADO_ORG", "foo-org")
	t.Setenv("ADO_PROJECT", "bar-project")
	t.Setenv("AZURE_DEVOPS_EXT_PAT", "pat-xyz")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cwd, _ := os.Getwd()
	assert.Equal(t, "foo-org", cfg.Org)
	assert.Equal(t, "bar-project", cfg.Project)
	assert.Equal(t, "pat-xyz", cfg.PAT)
	assert.Equal(t, cwd, cfg.ScopeRoot)
}

func TestLoad_MissingEnv(t *testing.T) {
	cases := []struct {
		name         string
		org, proj, pat string
		wantMissing  []string
	}{
		{"all missing", "", "", "", []string{"ADO_ORG", "ADO_PROJECT", "AZURE_DEVOPS_EXT_PAT"}},
		{"org missing", "", "p", "k", []string{"ADO_ORG"}},
		{"project missing", "o", "", "k", []string{"ADO_PROJECT"}},
		{"pat missing", "o", "p", "", []string{"AZURE_DEVOPS_EXT_PAT"}},
		{"org+pat missing", "", "p", "", []string{"ADO_ORG", "AZURE_DEVOPS_EXT_PAT"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ADO_ORG", tc.org)
			t.Setenv("ADO_PROJECT", tc.proj)
			t.Setenv("AZURE_DEVOPS_EXT_PAT", tc.pat)

			cfg, err := config.Load()
			require.Error(t, err)
			assert.Nil(t, cfg)
			assert.True(t, errors.Is(err, config.ErrMissingEnv),
				"error should wrap ErrMissingEnv, got %v", err)
			for _, name := range tc.wantMissing {
				assert.Contains(t, err.Error(), name)
			}
		})
	}
}

// TestLoad_ErrorOrder documents that missing var names appear in the code's
// check order: ORG, PROJECT, PAT. Guards against someone reordering the
// checks and quietly changing the error message shape consumers might parse.
func TestLoad_ErrorOrder(t *testing.T) {
	t.Setenv("ADO_ORG", "")
	t.Setenv("ADO_PROJECT", "set")
	t.Setenv("AZURE_DEVOPS_EXT_PAT", "")

	_, err := config.Load()
	require.Error(t, err)
	msg := err.Error()
	orgIdx := indexOf(msg, "ADO_ORG")
	patIdx := indexOf(msg, "AZURE_DEVOPS_EXT_PAT")
	require.GreaterOrEqual(t, orgIdx, 0)
	require.GreaterOrEqual(t, patIdx, 0)
	assert.Less(t, orgIdx, patIdx, "ADO_ORG should appear before AZURE_DEVOPS_EXT_PAT")
	assert.NotContains(t, msg, "ADO_PROJECT")
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
