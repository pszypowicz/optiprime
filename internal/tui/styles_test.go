package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStyles_BackgroundImpliesForeground(t *testing.T) {
	for _, isDark := range []bool{true, false} {
		s := newStyles(isDark)
		tinted := map[string]lipgloss.Style{
			"title":       s.title,
			"tabActive":   s.tabActive,
			"zebra":       s.zebra,
			"rowSelected": s.rowSelected,
			"overlay":     s.overlay,
		}
		for name, st := range tinted {
			require.NotEqual(t, lipgloss.NoColor{}, st.GetBackground(), "%s (dark=%v) has no background", name, isDark)
			assert.NotEqual(t, lipgloss.NoColor{}, st.GetForeground(), "%s (dark=%v) has no foreground", name, isDark)
		}
	}
}

func TestApplyRowBg_BackgroundResumesAfterStyledCell(t *testing.T) {
	s := newStyles(true)
	prefix := styleOpenSequence(s.zebra)
	require.NotEmpty(t, prefix)

	out := applyRowBg("a"+s.muted.Render("b")+"c", s.zebra)

	assert.True(t, strings.HasPrefix(out, prefix))
	assert.Contains(t, out, ansi.ResetStyle+prefix+"c")
}
