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
			cred := NewEnvCredential("AZURE_DEVOPS_EXT_PAT", "pat")
			c := NewClient(tc.org, "proj", cred)
			assert.Equal(t, tc.wantURL, c.baseURL)
			assert.Equal(t, "proj", c.project)
			assert.Equal(t, cred, c.cred)
			assert.NotNil(t, c.httpClient)
		})
	}
}
