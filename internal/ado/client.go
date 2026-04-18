package ado

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

// Client talks to a single ADO project.
//
// Org may be supplied as a bare name ("contoso") or a full
// URL ("https://dev.azure.com/contoso"). Both are normalized
// into baseURL (the "org root").
type Client struct {
	baseURL    string
	project    string
	pat        string
	httpClient *http.Client
}

func NewClient(org, project, pat string) *Client {
	base := strings.TrimRight(org, "/")
	lower := strings.ToLower(base)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		base = "https://dev.azure.com/" + url.PathEscape(base)
	}
	return &Client{
		baseURL:    base,
		project:    project,
		pat:        pat,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+c.pat))
}
