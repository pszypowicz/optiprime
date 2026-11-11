package ado

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pszypowicz/optiprime/internal/applog"
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
	auth, err := c.cred.Authorization(ctx)
	if err != nil {
		return err
	}
	status, body, err := c.send(ctx, u, auth)
	if err != nil {
		return err
	}
	// A rejected token that can be minted again gets one retry. az can hand
	// back the same token, and resending that would only fail again.
	if isAuthRejection(status) && c.cred.Renew(auth) {
		fresh, err := c.cred.Authorization(ctx)
		if err != nil {
			return err
		}
		if fresh != auth {
			applog.Infof("ado.request", u, "HTTP %d, retrying with a fresh token", status)
			if status, body, err = c.send(ctx, u, fresh); err != nil {
				return err
			}
		}
	}

	switch {
	case status == http.StatusOK:
		// proceed
	case isAuthRejection(status):
		applog.Errorf("ado.request", u, "auth failed HTTP %d body=%s", status, string(body))
		return fmt.Errorf("ado auth failed (%d) - %s", status, c.cred.rejectedHint())
	case status == http.StatusForbidden && isUnknownIdentity(body):
		applog.Errorf("ado.request", u, "403 body=%s", string(body))
		return errors.New("ado does not know this identity (403, has not been materialized) - " +
			"sign in to the org once in a browser, or set ADO_TENANT_ID to the org's Entra tenant")
	case status == http.StatusNotFound:
		applog.Errorf("ado.request", u, "404 body=%s", string(body))
		return fmt.Errorf("ado path %q returned 404 - check ADO_ORG/ADO_PROJECT", orgPath)
	default:
		applog.Errorf("ado.request", u, "HTTP %d body=%s", status, string(body))
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return fmt.Errorf("ado HTTP %d: %s", status, snippet)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func (c *Client) send(ctx context.Context, u, auth string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		applog.Errorf("ado.request", u, "transport error: %v", err)
		return 0, nil, fmt.Errorf("ado request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return resp.StatusCode, body, nil
}

// isAuthRejection: ADO answers a refused credential with 401, or with 203
// and a sign-in page.
func isAuthRejection(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusNonAuthoritativeInfo
}

// isUnknownIdentity: the org does not know the identity in the token,
// usually because another tenant issued it.
func isUnknownIdentity(body []byte) bool {
	return bytes.Contains(body, []byte("has not been materialized")) ||
		bytes.Contains(body, []byte("AadUserStateException"))
}
