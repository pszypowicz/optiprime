package ado

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pszypowicz/optiprime/internal/applog"
)

// adoResourceID is the public Azure DevOps resource id, the same in every
// tenant.
const adoResourceID = "499b84ac-1321-427f-aa17-267ca6975798"

// nilTenantID in X-VSS-ResourceTenant marks an org that no Entra tenant
// backs.
const nilTenantID = "00000000-0000-0000-0000-000000000000"

// refreshMargin: a cached token that expires sooner than this is minted
// again before use.
const refreshMargin = 5 * time.Minute

// ErrAzUnavailable: the az CLI is not installed or holds no login, so az
// cannot stand in for a PAT.
var ErrAzUnavailable = errors.New("az login is not available")

var errAzNotFound = errors.New("az CLI not found on PATH")

var tenantIDRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsTenantID reports whether s has the shape of an Entra tenant GUID.
func IsTenantID(s string) bool { return tenantIDRE.MatchString(s) }

type azRunner func(ctx context.Context, args ...string) (stdout, stderr string, err error)

// AzCredential mints Azure DevOps access tokens from the az login session.
// Tokens are minted for the Entra tenant that backs the org, which can
// differ from the tenant of the user's Azure subscriptions.
type AzCredential struct {
	orgURL       string
	run          azRunner
	probe        *http.Client
	now          func() time.Time
	loginCommand func() string

	mu             sync.Mutex
	tenantID       string
	tenantResolved bool
	token          string
	exp            time.Time // zero when the token carries no readable exp
}

// NewAzCredential returns a credential for org. A non-empty tenantID is
// used as the org's tenant and skips the lookup.
func NewAzCredential(org, tenantID string) *AzCredential {
	return &AzCredential{
		orgURL: orgURL(org),
		run:    execAz,
		probe: &http.Client{
			Timeout: requestTimeout,
			// The tenant header sits on the first response, which for an
			// anonymous caller can be a redirect to sign-in.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		now:            time.Now,
		loginCommand:   func() string { return loginCommand(os.Getenv, tmuxSSHConnection) },
		tenantID:       tenantID,
		tenantResolved: tenantID != "",
	}
}

func (c *AzCredential) Authorization(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" || (!c.exp.IsZero() && c.now().Add(refreshMargin).After(c.exp)) {
		if err := c.mint(ctx); err != nil {
			return "", err
		}
	}
	return "Bearer " + c.token, nil
}

// Renew drops the cached token when it is the rejected one. A token another
// request already replaced stays cached.
func (c *AzCredential) Renew(rejected string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && rejected == "Bearer "+c.token {
		c.token = ""
	}
	return true
}

func (c *AzCredential) rejectedHint() string {
	return fmt.Sprintf("the az login account may lack access to %s", c.orgURL)
}

func (c *AzCredential) mint(ctx context.Context) error {
	if !c.tenantResolved {
		c.tenantID, c.tenantResolved = discoverTenant(ctx, c.probe, c.orgURL)
	}
	args := []string{"account", "get-access-token", "--resource", adoResourceID,
		"--query", "accessToken", "-o", "tsv"}
	if c.tenantID != "" {
		args = append(args, "--tenant", c.tenantID)
	}
	stdout, stderr, err := c.run(ctx, args...)
	if err != nil {
		mintErr := c.explain(err, strings.TrimSpace(stdout+"\n"+stderr))
		if !errors.Is(mintErr, ErrAzUnavailable) {
			applog.Errorf("ado.az", c.orgURL, "%v", mintErr)
		}
		return mintErr
	}
	// On success the token is the only JWT-shaped line, whatever warnings
	// az prints around it.
	token := ""
	for line := range strings.Lines(stdout) {
		if line = strings.TrimSpace(line); isJWT(line) {
			token = line
		}
	}
	if token == "" {
		return errors.New("az account get-access-token returned no token")
	}
	c.token = token
	c.exp = jwtExpiry(token)
	return nil
}

func (c *AzCredential) explain(err error, out string) error {
	if errors.Is(err, errAzNotFound) {
		return fmt.Errorf("%w: %v", ErrAzUnavailable, err)
	}
	login := c.loginCommand()
	tenant := c.tenantID
	if tenant == "" {
		tenant = "<unknown>"
	}
	switch {
	case strings.Contains(out, "AADSTS700082"):
		return fmt.Errorf("the az login session expired after going unused (AADSTS700082) - sign in again with: %s", login)
	case containsAny(out, "AADSTS50020", "AADSTS90072"):
		return fmt.Errorf("your az account is not a member or guest of tenant %s, which backs %s - ask an org admin for access, or use a PAT",
			tenant, c.orgURL)
	case containsAny(out, "AADSTS50076", "AADSTS50079", "AADSTS50158", "AADSTS65001", "interaction_required"):
		tenantFlag := ""
		if c.tenantID != "" {
			tenantFlag = "--tenant " + c.tenantID + " "
		}
		return fmt.Errorf("the tenant behind %s requires an interactive sign-in - run: %s %s--allow-no-subscriptions --scope %s/.default",
			c.orgURL, login, tenantFlag, adoResourceID)
	case strings.Contains(out, "az login"):
		return fmt.Errorf("%w - run '%s' or set AZURE_DEVOPS_EXT_PAT", ErrAzUnavailable, login)
	default:
		return fmt.Errorf("az account get-access-token failed (%v): %s", err, out)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// discoverTenant reads the Entra tenant that backs orgURL from the
// X-VSS-ResourceTenant header, which ADO sends even to anonymous callers.
// tenant is "" when the org reports none. resolved is false when no HTTP
// response arrived, so a later mint can ask again.
func discoverTenant(ctx context.Context, hc *http.Client, orgURL string) (tenant string, resolved bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, orgURL+"/_apis/connectionData", nil)
	if err != nil {
		return "", false
	}
	resp, err := hc.Do(req)
	if err != nil {
		applog.Warnf("ado.tenant", orgURL, "tenant lookup failed, minting for the az login default tenant: %v", err)
		return "", false
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()

	// The value can list several tenants.
	for v := range strings.SplitSeq(strings.Join(resp.Header.Values("X-VSS-ResourceTenant"), ","), ",") {
		v = strings.TrimSpace(v)
		if IsTenantID(v) && v != nilTenantID {
			return v, true
		}
	}
	applog.Infof("ado.tenant", orgURL, "org reports no Entra tenant, minting for the az login default tenant")
	return "", true
}

// jwtExpiry returns the exp claim of tok, or the zero time when it cannot
// be read. The signature is not checked, the claim only schedules renewal.
func jwtExpiry(tok string) time.Time {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return time.Time{}
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(raw, &claims) != nil || claims.Exp <= 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}

func execAz(ctx context.Context, args ...string) (string, string, error) {
	path, err := exec.LookPath("az")
	if err != nil {
		return "", "", errAzNotFound
	}
	cmd := exec.CommandContext(ctx, path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

// loginCommand returns the az login command to suggest. Over SSH it is the
// device code flow, because the browser that plain az login opens would
// appear on the remote machine's screen. Inside tmux the session
// environment decides, because tmux refreshes SSH_CONNECTION there on every
// attach while a long-lived pane keeps the value it started with.
func loginCommand(getenv func(string) string, tmuxEnv func() (string, bool)) string {
	overSSH := getenv("SSH_CONNECTION") != ""
	if getenv("TMUX") != "" {
		if out, ok := tmuxEnv(); ok {
			overSSH = strings.HasPrefix(out, "SSH_CONNECTION=")
		}
	}
	if overSSH {
		return "az login --use-device-code"
	}
	return "az login"
}

// tmuxSSHConnection returns the SSH_CONNECTION entry of the tmux session
// environment. ok is false when tmux is not on PATH.
func tmuxSSHConnection() (string, bool) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return "", false
	}
	out, _ := exec.Command(path, "show-environment", "SSH_CONNECTION").Output()
	return strings.TrimSpace(string(out)), true
}
