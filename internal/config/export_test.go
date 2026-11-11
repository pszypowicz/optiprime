package config

import "testing"

// StubAz makes Load see the az CLI as installed or not, whatever the PATH
// of the machine running the tests holds.
func StubAz(t testing.TB, installed bool) {
	t.Helper()
	old := azInstalled
	azInstalled = func() bool { return installed }
	t.Cleanup(func() { azInstalled = old })
}
