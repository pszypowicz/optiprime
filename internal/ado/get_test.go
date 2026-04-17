package ado

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type emptyPayload struct {
	Value []any `json:"value"`
}

func TestGet_DefaultApiVersionInjected(t *testing.T) {
	var gotQuery url.Values
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"value":[]}`))
	})

	var out emptyPayload
	require.NoError(t, c.get(context.Background(), "/something", nil, &out))
	assert.Equal(t, "7.1", gotQuery.Get("api-version"))
}

func TestGet_ApiVersionOverridePreserved(t *testing.T) {
	var gotQuery url.Values
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"value":[]}`))
	})

	q := url.Values{}
	q.Set("api-version", "7.1-preview.1")

	var out emptyPayload
	require.NoError(t, c.get(context.Background(), "/something", q, &out))
	assert.Equal(t, "7.1-preview.1", gotQuery.Get("api-version"))
}

func TestGet_HeadersSent(t *testing.T) {
	var gotAuth, gotAccept string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte(`{"value":[]}`))
	})

	var out emptyPayload
	require.NoError(t, c.get(context.Background(), "/x", nil, &out))
	// base64(":tok") = "OnRvaw=="
	assert.Equal(t, "Basic OnRvaw==", gotAuth)
	assert.Equal(t, "application/json", gotAccept)
}

func TestGet_StatusBranches(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		body        string
		wantErrSubs []string
	}{
		{"200 ok", http.StatusOK, `{"value":[]}`, nil},
		{"200 malformed", http.StatusOK, `not-json`, []string{"parse response"}},
		{"401 unauth", http.StatusUnauthorized, `denied`, []string{"ado auth failed (401)"}},
		{"203 treated as unauth", http.StatusNonAuthoritativeInfo, `denied`, []string{"ado auth failed (203)"}},
		{"404 path", http.StatusNotFound, `nope`, []string{"returned 404", `/bad`}},
		{"500 short body", http.StatusInternalServerError, "oops", []string{"ado HTTP 500", "oops"}},
		{"500 whitespace body trimmed", http.StatusInternalServerError, "   spaced   ", []string{"spaced"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})

			var out emptyPayload
			err := c.get(context.Background(), "/bad", nil, &out)
			if len(tc.wantErrSubs) == 0 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, s := range tc.wantErrSubs {
				assert.Contains(t, err.Error(), s)
			}
		})
	}
}

func TestGet_500BodyTruncatedWithEllipsis(t *testing.T) {
	longBody := strings.Repeat("A", 300)
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(longBody))
	})

	var out emptyPayload
	err := c.get(context.Background(), "/x", nil, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "...", "long body should be truncated with ellipsis")
	// 200 A's + "..." must be present, 201+ A's in a row must not.
	assert.Contains(t, err.Error(), strings.Repeat("A", 200)+"...")
	assert.NotContains(t, err.Error(), strings.Repeat("A", 201))
}

func TestGet_BodyLimitDoesNotHang(t *testing.T) {
	// 10 MB body; client reads 8 MB limit. Must complete without OOM/hang.
	const tenMB = 10 << 20
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Write JSON that will be truncated; parse error is acceptable.
		_, _ = w.Write([]byte(`{"value":["`))
		chunk := make([]byte, 64<<10)
		for i := range chunk {
			chunk[i] = 'A'
		}
		written := len(`{"value":["`)
		for written < tenMB {
			_, _ = w.Write(chunk)
			written += len(chunk)
		}
	})

	var out emptyPayload
	err := c.get(context.Background(), "/big", nil, &out)
	// Either parses (extremely unlikely given truncation) or errors; both fine.
	// The assertion that matters: call returned.
	_ = err
	_ = out
}

func TestGet_TransportError(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"value":[]}`))
	})
	srv.Close() // force a transport error on the next call

	var out emptyPayload
	err := c.get(context.Background(), "/x", nil, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ado request")
}
