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
		BaseURL: srv.URL,
		Project: "proj",
		PAT:     "tok",
		HTTP:    srv.Client(),
	}
	return c, srv
}
