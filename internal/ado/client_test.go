package ado

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient_OrgNormalization(t *testing.T) {
	cases := []struct {
		name    string
		org     string
		wantURL string
	}{
		{"bare name", "contoso", "https://dev.azure.com/contoso"},
		{"name with spaces", "My Team Org", "https://dev.azure.com/My%20Team%20Org"},
		{"full https URL preserved", "https://dev.azure.com/foo", "https://dev.azure.com/foo"},
		{"trailing slash stripped", "https://dev.azure.com/foo/", "https://dev.azure.com/foo"},
		{"http preserved", "http://on-prem.corp/tfs", "http://on-prem.corp/tfs"},
		{"case-insensitive scheme detection", "HTTPS://FOO/bar", "HTTPS://FOO/bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(tc.org, "proj", "pat")
			assert.Equal(t, tc.wantURL, c.baseURL)
			assert.Equal(t, "proj", c.project)
			assert.Equal(t, "pat", c.pat)
			assert.NotNil(t, c.httpClient)
		})
	}
}

func TestAuthHeader(t *testing.T) {
	// base64(":abc123") = "OmFiYzEyMw=="
	c := &Client{pat: "abc123"}
	assert.Equal(t, "Basic OmFiYzEyMw==", c.authHeader())

	// base64(":") = "Og=="
	cEmpty := &Client{pat: ""}
	assert.Equal(t, "Basic Og==", cEmpty.authHeader())
}
