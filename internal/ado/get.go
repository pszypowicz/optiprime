package ado

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pszypowicz/optiprime-sync/internal/applog"
)

// get hits BaseURL + orgPath with the given query. orgPath must start with "/"
// and be relative to the org root (e.g. "/_apis/connectionData" or
// "/<project>/_apis/git/repositories").
func (c *Client) get(ctx context.Context, orgPath string, q url.Values, out any) error {
	if q == nil {
		q = url.Values{}
	}
	// Default to the GA api-version but let specific endpoints override
	// (e.g. connectionData is only available as a preview resource).
	if q.Get("api-version") == "" {
		q.Set("api-version", apiVersion)
	}

	u := c.baseURL + orgPath + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		applog.Errorf("ado.request", u, "transport error: %v", err)
		return fmt.Errorf("ado request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	switch resp.StatusCode {
	case http.StatusOK:
		// proceed
	case http.StatusUnauthorized, http.StatusNonAuthoritativeInfo:
		applog.Errorf("ado.request", u, "auth failed HTTP %d body=%s", resp.StatusCode, string(body))
		return fmt.Errorf("ado auth failed (%d) - check AZURE_DEVOPS_EXT_PAT scope/expiry", resp.StatusCode)
	case http.StatusNotFound:
		applog.Errorf("ado.request", u, "404 body=%s", string(body))
		return fmt.Errorf("ado path %q returned 404 - check ADO_ORG/ADO_PROJECT", orgPath)
	default:
		applog.Errorf("ado.request", u, "HTTP %d body=%s", resp.StatusCode, string(body))
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return fmt.Errorf("ado HTTP %d: %s", resp.StatusCode, snippet)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}
