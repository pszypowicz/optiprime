package ado

import (
	"context"
	"fmt"
	"net/url"
)

type connectionData struct {
	AuthenticatedUser struct {
		ID string `json:"id"`
	} `json:"authenticatedUser"`
}

func (c *Client) AuthUserID(ctx context.Context) (string, error) {
	// connectionData is a preview-only resource; GA api-version gets rejected.
	q := url.Values{}
	q.Set("api-version", "7.1-preview.1")

	var cd connectionData
	if err := c.get(ctx, "/_apis/connectionData", q, &cd); err != nil {
		return "", err
	}
	if cd.AuthenticatedUser.ID == "" {
		return "", fmt.Errorf("connectionData returned no authenticatedUser.id")
	}
	return cd.AuthenticatedUser.ID, nil
}
