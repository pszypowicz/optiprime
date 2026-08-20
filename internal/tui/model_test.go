package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pszypowicz/optiprime/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitOnlyModel() model {
	return newModel(&config.Config{ScopeRoot: "/scope"})
}

func remoteModel() model {
	return newModel(&config.Config{
		Org: "foo-org", Project: "bar-project", PAT: "pat-xyz", ScopeRoot: "/scope",
	})
}

func TestNewModel_NoPAT_RemoteFeaturesOff(t *testing.T) {
	m := gitOnlyModel()

	assert.False(t, m.remoteEnabled())
	assert.False(t, m.loadingRemotes, "nothing will ever load the remote list")
	assert.False(t, m.loadingPRs, "nothing will ever load PR counts")
}

func TestNewModel_WithPAT_RemoteFeaturesOn(t *testing.T) {
	m := remoteModel()

	assert.True(t, m.remoteEnabled())
	assert.True(t, m.loadingRemotes)
	assert.True(t, m.loadingPRs)
}

// An empty remote list in git-only mode means "unknown", not "orphan" -
// every repo must still get a real fetch.
func TestCanSkipFetch_RemoteFeaturesOff(t *testing.T) {
	m := gitOnlyModel()

	assert.False(t, m.canSkipFetch("any-repo", m.remoteByName()))
}

func TestRefresh_RemoteFeaturesOff_DoesNotWaitOnRemotes(t *testing.T) {
	m := gitOnlyModel()

	updated, _ := m.Update(refreshMsg{})
	got, ok := updated.(model)
	require.True(t, ok)

	assert.True(t, got.loadingLocals)
	assert.False(t, got.loadingRemotes)
	assert.False(t, got.loadingPRs)
}

func TestTabKey_BlockedWhenRemoteOff(t *testing.T) {
	m := gitOnlyModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got, ok := updated.(model)
	require.True(t, ok)

	assert.Equal(t, tabLocal, got.tab)
	assert.Contains(t, got.flash, "AZURE_DEVOPS_EXT_PAT")
}

func TestTabClick_BlockedWhenRemoteOff(t *testing.T) {
	m := gitOnlyModel()

	m.handleClick(5, 2)

	assert.Equal(t, tabLocal, m.tab)
	assert.Contains(t, m.flash, "AZURE_DEVOPS_EXT_PAT")
}

func TestRenderTabs_OffLabelWhenRemoteOff(t *testing.T) {
	m := gitOnlyModel()

	assert.Contains(t, m.renderTabs(), "Remote (off)")
}

func TestRenderHeader_RemoteOff(t *testing.T) {
	m := gitOnlyModel()
	m.width = 200
	m.height = 50

	head := m.renderHeader()
	assert.Contains(t, head, "AZURE_DEVOPS_EXT_PAT not set")
	assert.Contains(t, head, "/scope")
	assert.NotContains(t, head, " / ", "no empty org/project placeholder")
}

func TestRenderHeader_RemoteOn(t *testing.T) {
	m := remoteModel()
	m.width = 200
	m.height = 50

	head := m.renderHeader()
	assert.Contains(t, head, "foo-org / bar-project")
	assert.NotContains(t, head, "AZURE_DEVOPS_EXT_PAT")
}

func TestRenderFooter_NoRemoteKeyWhenRemoteOff(t *testing.T) {
	m := gitOnlyModel()
	m.width = 200
	m.height = 50

	assert.NotContains(t, m.renderFooter(), "[tab] remote")
}

func TestRenderLocal_NoOrphanStateWhenRemoteOff(t *testing.T) {
	m := gitOnlyModel()
	it := &localItem{Name: "some-repo"}

	state := renderLocalState(it, m.remoteByName())
	assert.NotContains(t, state, "not in ADO")
}
