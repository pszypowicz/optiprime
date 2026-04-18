package ado

import (
	"context"
	"net/url"
	"sort"
	"strings"
)

type Repo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SSHURL    string `json:"sshUrl"`
	RemoteURL string `json:"remoteUrl"`
	WebURL    string `json:"webUrl"`
	Disabled  bool   `json:"isDisabled"`
}

type listReposResponse struct {
	Value []Repo `json:"value"`
}

func (c *Client) ListRepos(ctx context.Context) ([]Repo, error) {
	path := "/" + url.PathEscape(c.project) + "/_apis/git/repositories"
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
