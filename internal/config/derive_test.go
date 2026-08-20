package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseADORemote(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want adoRemote
		ok   bool
	}{
		{
			name: "https with user info",
			raw:  "https://contoso@dev.azure.com/contoso/Proj/_git/repo",
			want: adoRemote{org: "contoso", project: "Proj"},
			ok:   true,
		},
		{
			name: "https without user info",
			raw:  "https://dev.azure.com/contoso/Proj/_git/repo",
			want: adoRemote{org: "contoso", project: "Proj"},
			ok:   true,
		},
		{
			name: "https with encoded space in project",
			raw:  "https://dev.azure.com/contoso/My%20Proj/_git/repo",
			want: adoRemote{org: "contoso", project: "My Proj"},
			ok:   true,
		},
		{
			name: "scp-like ssh",
			raw:  "git@ssh.dev.azure.com:v3/contoso/Proj/repo",
			want: adoRemote{org: "contoso", project: "Proj"},
			ok:   true,
		},
		{
			name: "scp-like ssh with encoded space",
			raw:  "git@ssh.dev.azure.com:v3/contoso/My%20Proj/repo",
			want: adoRemote{org: "contoso", project: "My Proj"},
			ok:   true,
		},
		{
			name: "ssh scheme url",
			raw:  "ssh://git@ssh.dev.azure.com/v3/contoso/Proj/repo",
			want: adoRemote{org: "contoso", project: "Proj"},
			ok:   true,
		},
		{name: "github https", raw: "https://github.com/o/r.git", ok: false},
		{name: "github ssh", raw: "git@github.com:o/r.git", ok: false},
		{name: "https wrong depth", raw: "https://dev.azure.com/contoso/repo", ok: false},
		{name: "https missing _git", raw: "https://dev.azure.com/contoso/Proj/x/repo", ok: false},
		{name: "ssh not v3", raw: "git@ssh.dev.azure.com:v2/contoso/Proj/repo", ok: false},
		{name: "local path", raw: "/srv/git/repo.git", ok: false},
		{name: "empty", raw: "", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseADORemote(tc.raw)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
