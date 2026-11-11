package ado

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJWT builds an unsigned JWT-shaped token whose payload carries exp.
// exp <= 0 leaves the claim out.
func fakeJWT(exp int64, tag string) string {
	payload := fmt.Sprintf(`{"tag":%q}`, tag)
	if exp > 0 {
		payload = fmt.Sprintf(`{"exp":%d,"tag":%q}`, exp, tag)
	}
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(`{"alg":"none"}`)) + "." + enc.EncodeToString([]byte(payload)) + ".sig"
}

func TestEnvCredential_PATUsesBasic(t *testing.T) {
	c := NewEnvCredential("AZURE_DEVOPS_EXT_PAT", "abc123")

	got, err := c.Authorization(context.Background())
	require.NoError(t, err)
	// base64(":abc123") = "OmFiYzEyMw=="
	assert.Equal(t, "Basic OmFiYzEyMw==", got)
}

func TestEnvCredential_JWTUsesBearer(t *testing.T) {
	tok := fakeJWT(0, "env")
	for _, name := range []string{"AZURE_DEVOPS_EXT_PAT", "ADO_TOKEN"} {
		t.Run(name, func(t *testing.T) {
			got, err := NewEnvCredential(name, tok).Authorization(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "Bearer "+tok, got)
		})
	}
}

func TestEnvCredential_NeverRenews(t *testing.T) {
	c := NewEnvCredential("AZURE_DEVOPS_EXT_PAT", "abc123")
	got, err := c.Authorization(context.Background())
	require.NoError(t, err)

	assert.False(t, c.Renew(got), "a credential the user supplied is never replaced")
}
