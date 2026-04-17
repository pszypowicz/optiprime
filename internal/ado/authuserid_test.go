package ado

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthUserID_UsesPreviewApiVersion(t *testing.T) {
	var gotApiVersion, gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotApiVersion = r.URL.Query().Get("api-version")
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"authenticatedUser":{"id":"abc-123"}}`))
	})

	id, err := c.AuthUserID(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "abc-123", id)
	assert.Equal(t, "7.1-preview.1", gotApiVersion, "GA default must not leak into connectionData")
	assert.Equal(t, "/_apis/connectionData", gotPath)
}

func TestAuthUserID_EmptyIDError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"authenticatedUser":{"id":""}}`))
	})

	id, err := c.AuthUserID(context.Background())
	require.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "authenticatedUser.id")
}

func TestAuthUserID_TransportErrorPropagates(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.AuthUserID(context.Background())
	require.Error(t, err)
}
