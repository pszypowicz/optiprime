package ado

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// Credential supplies the Authorization header for REST calls.
type Credential interface {
	// Authorization returns the value of the Authorization header.
	Authorization(ctx context.Context) (string, error)
	// Renew is called when ADO rejects the header value rejected. It reports
	// whether asking Authorization again can yield a different value.
	Renew(rejected string) bool
	// rejectedHint tells the user what to check after ADO rejects the
	// credential.
	rejectedHint() string
}

// isJWT reports whether s has the shape of an Entra access token.
func isJWT(s string) bool {
	return strings.HasPrefix(s, "eyJ") && strings.Count(s, ".") == 2
}

type envCredential struct {
	name   string
	header string
}

// NewEnvCredential wraps a credential read from the environment variable
// name. A JWT-shaped value is an Entra access token and goes out as Bearer.
// Any other value is a PAT and goes out as Basic.
func NewEnvCredential(name, value string) Credential {
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+value))
	if isJWT(value) {
		header = "Bearer " + value
	}
	return &envCredential{name: name, header: header}
}

func (c *envCredential) Authorization(context.Context) (string, error) {
	return c.header, nil
}

// Renew never replaces a credential the user supplied.
func (c *envCredential) Renew(string) bool { return false }

func (c *envCredential) rejectedHint() string {
	return fmt.Sprintf("check %s scope/expiry", c.name)
}
