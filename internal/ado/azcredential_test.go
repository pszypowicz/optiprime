package ado

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	orgTenant = "11111111-2222-3333-4444-555555555555"
	nilTenant = "00000000-0000-0000-0000-000000000000"
)

type azResult struct {
	stdout, stderr string
	err            error
}

// fakeAz records az invocations and replays queued results in order.
// Running out of results fails the test.
type fakeAz struct {
	t       *testing.T
	mu      sync.Mutex
	results []azResult
	calls   [][]string
}

func (f *fakeAz) queue(r ...azResult) { f.results = append(f.results, r...) }

func (f *fakeAz) run(_ context.Context, args ...string) (string, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, append([]string(nil), args...))
	if len(f.results) == 0 {
		f.t.Errorf("unexpected az call: %v", args)
		return "", "", errors.New("unexpected az call")
	}
	r := f.results[0]
	f.results = f.results[1:]
	return r.stdout, r.stderr, r.err
}

func (f *fakeAz) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// tenantServer answers the anonymous connectionData probe the way ADO does:
// 401 with the org tenant in X-VSS-ResourceTenant. header "" sends none.
func tenantServer(t *testing.T, header string) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		assert.Equal(t, "/_apis/connectionData", r.URL.Path)
		assert.Empty(t, r.Header.Get("Authorization"), "the tenant probe is anonymous")
		if header != "" {
			w.Header().Set("X-VSS-ResourceTenant", header)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func newTestAz(t *testing.T, orgURL, tenantID string) (*AzCredential, *fakeAz) {
	t.Helper()
	fake := &fakeAz{t: t}
	c := NewAzCredential(orgURL, tenantID)
	c.run = fake.run
	c.loginCommand = func() string { return "az login" }
	return c, fake
}

func mintArgs(tenant string) []string {
	args := []string{"account", "get-access-token", "--resource", adoResourceID,
		"--query", "accessToken", "-o", "tsv"}
	if tenant != "" {
		args = append(args, "--tenant", tenant)
	}
	return args
}

func TestAzCredential_MintsForDiscoveredTenant(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	tok := fakeJWT(time.Now().Add(time.Hour).Unix(), "a")
	fake.queue(azResult{stdout: tok})

	got, err := c.Authorization(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "Bearer "+tok, got)
	require.Len(t, fake.calls, 1)
	assert.Equal(t, mintArgs(orgTenant), fake.calls[0])
}

func TestAzCredential_TenantHeaderListsSeveral(t *testing.T) {
	srv, _ := tenantServer(t, nilTenant+", not-a-guid, "+orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	fake.queue(azResult{stdout: fakeJWT(0, "a")})

	_, err := c.Authorization(context.Background())
	require.NoError(t, err)
	assert.Equal(t, mintArgs(orgTenant), fake.calls[0], "first real tenant GUID wins")
}

func TestAzCredential_NoTenantReported(t *testing.T) {
	for name, header := range map[string]string{"no header": "", "nil tenant": nilTenant} {
		t.Run(name, func(t *testing.T) {
			srv, _ := tenantServer(t, header)
			c, fake := newTestAz(t, srv.URL, "")
			fake.queue(azResult{stdout: fakeJWT(0, "a")})

			_, err := c.Authorization(context.Background())
			require.NoError(t, err)
			assert.Equal(t, mintArgs(""), fake.calls[0], "mints for the az login default tenant")
		})
	}
}

func TestAzCredential_TenantProbeFails(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	c, fake := newTestAz(t, srv.URL, "")
	fake.queue(azResult{stdout: fakeJWT(0, "a")})

	_, err := c.Authorization(context.Background())
	require.NoError(t, err)
	assert.Equal(t, mintArgs(""), fake.calls[0])
}

func TestAzCredential_FixedTenantSkipsDiscovery(t *testing.T) {
	srv, hits := tenantServer(t, "99999999-9999-9999-9999-999999999999")
	c, fake := newTestAz(t, srv.URL, orgTenant)
	fake.queue(azResult{stdout: fakeJWT(0, "a")})

	_, err := c.Authorization(context.Background())
	require.NoError(t, err)
	assert.Equal(t, mintArgs(orgTenant), fake.calls[0])
	assert.Zero(t, *hits)
}

func TestAzCredential_DiscoversTenantOnce(t *testing.T) {
	srv, hits := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	first := fakeJWT(time.Now().Add(time.Hour).Unix(), "a")
	fake.queue(azResult{stdout: first}, azResult{stdout: fakeJWT(0, "b")})

	_, err := c.Authorization(context.Background())
	require.NoError(t, err)
	require.True(t, c.Renew("Bearer "+first))
	_, err = c.Authorization(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, *hits)
	assert.Equal(t, mintArgs(orgTenant), fake.calls[1])
}

func TestAzCredential_CachesValidToken(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	tok := fakeJWT(time.Now().Add(time.Hour).Unix(), "a")
	fake.queue(azResult{stdout: tok})

	for range 3 {
		got, err := c.Authorization(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "Bearer "+tok, got)
	}
	assert.Len(t, fake.calls, 1)
}

func TestAzCredential_RemintsNearExpiry(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	now := time.Now()
	c.now = func() time.Time { return now }
	old := fakeJWT(now.Add(4*time.Minute).Unix(), "old")
	fresh := fakeJWT(now.Add(time.Hour).Unix(), "fresh")
	fake.queue(azResult{stdout: old}, azResult{stdout: fresh})

	_, err := c.Authorization(context.Background())
	require.NoError(t, err)
	got, err := c.Authorization(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "Bearer "+fresh, got, "a token within 5 minutes of expiry is replaced")
}

func TestAzCredential_ConcurrentCallersMintOnce(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	fake.queue(azResult{stdout: fakeJWT(time.Now().Add(time.Hour).Unix(), "a")})

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			_, err := c.Authorization(context.Background())
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	assert.Equal(t, 1, fake.callCount())
}

func TestAzCredential_PicksTokenLineFromNoisyOutput(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	tok := fakeJWT(0, "a")
	fake.queue(azResult{stdout: "WARNING: something\n" + tok + "\n"})

	got, err := c.Authorization(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Bearer "+tok, got)
}

func TestAzCredential_NoTokenInOutput(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	fake.queue(azResult{stdout: "\n"})

	_, err := c.Authorization(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "returned no token")
	assert.False(t, errors.Is(err, ErrAzUnavailable))
}

func TestAzCredential_RenewDropsOnlyTheRejectedToken(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	tok := fakeJWT(time.Now().Add(time.Hour).Unix(), "a")
	fake.queue(azResult{stdout: tok})

	_, err := c.Authorization(context.Background())
	require.NoError(t, err)
	assert.True(t, c.Renew("Bearer stale-from-another-request"))
	got, err := c.Authorization(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "Bearer "+tok, got, "a header that is not the cached one leaves the cache alone")
	assert.Len(t, fake.calls, 1)
}

func TestAzCredential_MintErrors(t *testing.T) {
	exit1 := errors.New("exit status 1")
	cases := []struct {
		name        string
		stderr      string
		unavailable bool
		wantSubs    []string
	}{
		{
			name:        "not logged in",
			stderr:      "ERROR: Please run 'az login' to setup account.",
			unavailable: true,
			wantSubs:    []string{"az login"},
		},
		{
			name:     "session expired",
			stderr:   "ERROR: AADSTS700082: The refresh token has expired due to inactivity. Run 'az login'.",
			wantSubs: []string{"expired", "az login"},
		},
		{
			name:     "not in tenant",
			stderr:   "ERROR: AADSTS50020: User account from identity provider does not exist in tenant.",
			wantSubs: []string{"not a member or guest", orgTenant},
		},
		{
			name:     "interaction required",
			stderr:   "ERROR: AADSTS50076: Due to a configuration change made by your administrator ... Run the command below to authenticate interactively: az login --tenant " + orgTenant,
			wantSubs: []string{"az login --tenant " + orgTenant + " --allow-no-subscriptions --scope " + adoResourceID + "/.default"},
		},
		{
			name:     "other failure",
			stderr:   "ERROR: something unexpected",
			wantSubs: []string{"az account get-access-token failed", "something unexpected"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := tenantServer(t, orgTenant)
			c, fake := newTestAz(t, srv.URL, "")
			fake.queue(azResult{stderr: tc.stderr, err: exit1})

			_, err := c.Authorization(context.Background())
			require.Error(t, err)
			assert.Equal(t, tc.unavailable, errors.Is(err, ErrAzUnavailable))
			for _, s := range tc.wantSubs {
				assert.Contains(t, err.Error(), s)
			}
		})
	}
}

func TestAzCredential_AzNotInstalled(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, _ := newTestAz(t, srv.URL, "")
	c.run = func(context.Context, ...string) (string, string, error) {
		return "", "", errAzNotFound
	}

	_, err := c.Authorization(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAzUnavailable))
}

func TestAzCredential_SuggestsDeviceCodeOverSSH(t *testing.T) {
	srv, _ := tenantServer(t, orgTenant)
	c, fake := newTestAz(t, srv.URL, "")
	c.loginCommand = func() string { return "az login --use-device-code" }
	fake.queue(azResult{stderr: "ERROR: Please run 'az login' to setup account.", err: errors.New("exit status 1")})

	_, err := c.Authorization(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "az login --use-device-code")
}

func TestLoginCommand(t *testing.T) {
	ssh := "1.2.3.4 5 6.7.8.9 22"
	cases := []struct {
		name    string
		env     map[string]string
		tmuxOut string
		tmuxOK  bool
		want    string
	}{
		{"local", nil, "", false, "az login"},
		{"ssh", map[string]string{"SSH_CONNECTION": ssh}, "", false, "az login --use-device-code"},
		{"tmux attached over ssh", map[string]string{"TMUX": "/tmp/tmux"}, "SSH_CONNECTION=" + ssh, true, "az login --use-device-code"},
		{"tmux attached locally, stale pane value", map[string]string{"TMUX": "/tmp/tmux", "SSH_CONNECTION": ssh}, "-SSH_CONNECTION", true, "az login"},
		{"tmux not on PATH", map[string]string{"TMUX": "/tmp/tmux", "SSH_CONNECTION": ssh}, "", false, "az login --use-device-code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(k string) string { return tc.env[k] }
			tmux := func() (string, bool) { return tc.tmuxOut, tc.tmuxOK }
			assert.Equal(t, tc.want, loginCommand(getenv, tmux))
		})
	}
}
