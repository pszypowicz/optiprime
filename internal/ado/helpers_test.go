package ado

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := &Client{
		baseURL:    srv.URL,
		project:    "proj",
		cred:       NewEnvCredential("AZURE_DEVOPS_EXT_PAT", "tok"),
		httpClient: srv.Client(),
	}
	return c, srv
}
