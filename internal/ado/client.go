package ado

import (
	"context"
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
	cred       Credential
	httpClient *http.Client
}

func NewClient(org, project string, cred Credential) *Client {
	return &Client{
		baseURL:    orgURL(org),
		project:    project,
		cred:       cred,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

// orgURL normalizes an org name or URL into the org root URL.
func orgURL(org string) string {
	base := strings.TrimRight(org, "/")
	lower := strings.ToLower(base)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		base = "https://dev.azure.com/" + url.PathEscape(base)
	}
	return base
}

// CheckAuth obtains a credential without calling the REST API, so a
// credential that cannot be had surfaces before any request goes out.
func (c *Client) CheckAuth(ctx context.Context) error {
	_, err := c.cred.Authorization(ctx)
	return err
}
