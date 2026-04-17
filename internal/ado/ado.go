package ado

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	apiVersion     = "7.1"
	requestTimeout = 20 * time.Second
)

type Repo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SSHURL    string `json:"sshUrl"`
	RemoteURL string `json:"remoteUrl"`
	WebURL    string `json:"webUrl"`
	Disabled  bool   `json:"isDisabled"`
}

// Client talks to a single ADO project.
//
// Org may be supplied as a bare name ("contoso") or a full
// URL ("https://dev.azure.com/contoso"). Both are normalized
// into BaseURL (the "org root").
type Client struct {
	BaseURL string
	Project string
	PAT     string
	HTTP    *http.Client
}

func NewClient(org, project, pat string) *Client {
	base := strings.TrimRight(org, "/")
	lower := strings.ToLower(base)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		base = "https://dev.azure.com/" + url.PathEscape(base)
	}
	return &Client{
		BaseURL: base,
		Project: project,
		PAT:     pat,
		HTTP:    &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+c.PAT))
}

// get hits BaseURL + orgPath with the given query. orgPath must start with "/"
// and be relative to the org root (e.g. "/_apis/connectionData" or
// "/<project>/_apis/git/repositories").
func (c *Client) get(ctx context.Context, orgPath string, q url.Values, out any) error {
	if q == nil {
		q = url.Values{}
	}
	q.Set("api-version", apiVersion)

	u := c.BaseURL + orgPath + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("ado request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	switch resp.StatusCode {
	case http.StatusOK:
		// proceed
	case http.StatusUnauthorized, http.StatusNonAuthoritativeInfo:
		return fmt.Errorf("ado auth failed (%d) - check AZURE_DEVOPS_EXT_PAT scope/expiry", resp.StatusCode)
	case http.StatusNotFound:
		return fmt.Errorf("ado path %q returned 404 - check ADO_ORG/ADO_PROJECT", orgPath)
	default:
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

type listReposResponse struct {
	Value []Repo `json:"value"`
}

func (c *Client) ListRepos(ctx context.Context) ([]Repo, error) {
	path := "/" + url.PathEscape(c.Project) + "/_apis/git/repositories"
	var resp listReposResponse
	if err := c.get(ctx, path, nil, &resp); err != nil {
		return nil, err
	}
	out := append([]Repo(nil), resp.Value...)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

type connectionData struct {
	AuthenticatedUser struct {
		ID string `json:"id"`
	} `json:"authenticatedUser"`
}

func (c *Client) AuthUserID(ctx context.Context) (string, error) {
	var cd connectionData
	if err := c.get(ctx, "/_apis/connectionData", nil, &cd); err != nil {
		return "", err
	}
	if cd.AuthenticatedUser.ID == "" {
		return "", fmt.Errorf("connectionData returned no authenticatedUser.id")
	}
	return cd.AuthenticatedUser.ID, nil
}

type pullRequestsResponse struct {
	Value []struct {
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	} `json:"value"`
}

func (c *Client) MyOpenPRs(ctx context.Context, userID string) (map[string]int, error) {
	q := url.Values{}
	q.Set("searchCriteria.creatorId", userID)
	q.Set("searchCriteria.status", "active")
	q.Set("$top", "1000")

	path := "/" + url.PathEscape(c.Project) + "/_apis/git/pullrequests"
	var resp pullRequestsResponse
	if err := c.get(ctx, path, q, &resp); err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(resp.Value))
	for _, p := range resp.Value {
		counts[p.Repository.Name]++
	}
	return counts, nil
}
