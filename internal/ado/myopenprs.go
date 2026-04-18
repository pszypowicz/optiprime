package ado

import (
	"context"
	"net/url"
)

type pullRequestsResponse struct {
	Value []struct {
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	} `json:"value"`
}

func (c *Client) MyOpenPRs(ctx context.Context, userID string) (map[string]int, error) {
	// No $top: url.Values.Encode() percent-encodes the $ to %24, which ADO's
	// ASP request-path validator misreports as a "potentially dangerous"
	// colon. Default pagination is enough for one user's open PRs.
	q := url.Values{}
	q.Set("searchCriteria.creatorId", userID)
	q.Set("searchCriteria.status", "active")

	path := "/" + url.PathEscape(c.project) + "/_apis/git/pullrequests"
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
