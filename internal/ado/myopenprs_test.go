package ado

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMyOpenPRs_QueryShape(t *testing.T) {
	var gotQuery url.Values
	var gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"value":[
			{"repository":{"name":"repoA"}},
			{"repository":{"name":"repoA"}},
			{"repository":{"name":"repoB"}}
		]}`))
	})

	counts, err := c.MyOpenPRs(context.Background(), "user-1")
	require.NoError(t, err)

	assert.Equal(t, "user-1", gotQuery.Get("searchCriteria.creatorId"))
	assert.Equal(t, "active", gotQuery.Get("searchCriteria.status"))
	assert.Equal(t, "7.1", gotQuery.Get("api-version"))
	// Guard against someone re-adding $top - url.Values.Encode() percent-encodes
	// the $, which ADO's ASP request-path validator rejects.
	assert.False(t, gotQuery.Has("$top"), "$top must not be set")
	assert.Equal(t, "/proj/_apis/git/pullrequests", gotPath)

	assert.Equal(t, map[string]int{"repoA": 2, "repoB": 1}, counts)
}

func TestMyOpenPRs_EmptyValue(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"value":[]}`))
	})

	counts, err := c.MyOpenPRs(context.Background(), "user-1")
	require.NoError(t, err)
	assert.NotNil(t, counts)
	assert.Empty(t, counts)
}
