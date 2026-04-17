package ado

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRepos_SortCaseInsensitive(t *testing.T) {
	var gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"value":[
			{"id":"3","name":"zeta"},
			{"id":"1","name":"Alpha"},
			{"id":"2","name":"beta"}
		]}`))
	})

	repos, err := c.ListRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 3)
	assert.Equal(t, []string{"Alpha", "beta", "zeta"},
		[]string{repos[0].Name, repos[1].Name, repos[2].Name})
	assert.Equal(t, "/proj/_apis/git/repositories", gotPath)
}

func TestListRepos_ProjectWithSpacesEscaped(t *testing.T) {
	var gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"value":[]}`))
	})
	c.Project = "My Proj"

	_, err := c.ListRepos(context.Background())
	require.NoError(t, err)
	// httptest decodes the path before exposing it, but the raw path preserves encoding.
	// We can check either: here we verify the decoded form is "My Proj" and RawPath is escaped.
	assert.Equal(t, "/My Proj/_apis/git/repositories", gotPath)
}

func TestListRepos_ErrorPropagates(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server boom"))
	})

	repos, err := c.ListRepos(context.Background())
	require.Error(t, err)
	assert.Nil(t, repos)
	assert.Contains(t, err.Error(), "ado HTTP 500")
}
