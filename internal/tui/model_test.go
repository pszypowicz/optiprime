package tui

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
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

	state := m.renderLocalState(it, m.remoteByName())
	assert.NotContains(t, state, "not in ADO")
}

func TestNewModel_DarkPaletteUntilBackgroundKnown(t *testing.T) {
	m := gitOnlyModel()

	assert.Equal(t, newStyles(true), m.styles)
}

func TestBackgroundColorMsg_PicksPalette(t *testing.T) {
	cases := []struct {
		name   string
		bg     color.Color
		isDark bool
	}{
		{"light terminal", color.White, false},
		{"dark terminal", color.Black, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := gitOnlyModel()

			updated, _ := m.Update(tea.BackgroundColorMsg{Color: tc.bg})
			got, ok := updated.(model)
			require.True(t, ok)

			assert.Equal(t, newStyles(tc.isDark), got.styles)
		})
	}
}

func TestView_AltScreenAndMouse(t *testing.T) {
	m := gitOnlyModel()

	v := m.View()
	assert.True(t, v.AltScreen)
	assert.Equal(t, tea.MouseModeCellMotion, v.MouseMode)
}

func TestView_BoxFillsTerminal(t *testing.T) {
	m := gitOnlyModel()
	m.width = 80
	m.height = 24

	content := m.View().Content
	assert.Equal(t, 80, lipgloss.Width(content))
	assert.Equal(t, 24, lipgloss.Height(content))
}

func TestSpaceKey_TogglesSelection(t *testing.T) {
	m := gitOnlyModel()
	m.locals = []*localItem{{Name: "repo"}}

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	got, ok := updated.(model)
	require.True(t, ok)

	assert.True(t, got.locals[0].Selected)
}

func TestClickOnCheckbox_TogglesSelection(t *testing.T) {
	m := gitOnlyModel()
	m.locals = []*localItem{{Name: "repo"}}

	// Y=5 is the first data row, X=4 is the first checkbox column.
	updated, _ := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 4, Y: 5})
	got, ok := updated.(model)
	require.True(t, ok)

	assert.True(t, got.locals[0].Selected)
}
