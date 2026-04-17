package tui

import "github.com/charmbracelet/lipgloss"

// All colors are AdaptiveColor pairs so the UI picks the right shade for the
// terminal's background. Lipgloss auto-detects dark vs. light at startup;
// falls back to Dark if detection fails.
var (
	colorAccent  = lipgloss.AdaptiveColor{Light: "#5A3FA0", Dark: "#7D56F4"}
	colorSubtle  = lipgloss.AdaptiveColor{Light: "240", Dark: "241"}
	colorOK      = lipgloss.AdaptiveColor{Light: "28", Dark: "42"}
	colorWarn    = lipgloss.AdaptiveColor{Light: "130", Dark: "214"}
	colorErr     = lipgloss.AdaptiveColor{Light: "160", Dark: "196"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "245", Dark: "244"}
	colorHeading = lipgloss.AdaptiveColor{Light: "230", Dark: "229"}
	colorPR      = lipgloss.AdaptiveColor{Light: "26", Dark: "39"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHeading).
			Background(colorAccent).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	tabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHeading).
			Background(colorAccent).
			Padding(0, 2)

	tabInactive = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 2)

	// Zebra stripes for alt rows - subtle enough that colored foregrounds stay readable.
	zebraStyle = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "254", Dark: "236"})

	// Cursor row uses a stronger tint so it stands out even on a zebra row.
	rowSelected = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.AdaptiveColor{Light: "252", Dark: "238"})

	okStyle    = lipgloss.NewStyle().Foreground(colorOK)
	warnStyle  = lipgloss.NewStyle().Foreground(colorWarn)
	errStyle   = lipgloss.NewStyle().Foreground(colorErr)
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)
	prStyle    = lipgloss.NewStyle().Foreground(colorPR).Bold(true)
	helpStyle  = lipgloss.NewStyle().Foreground(colorSubtle)

	// Bold + underlined header. Foreground is chosen for contrast against the
	// terminal background, not against the accent color (which is what
	// colorHeading is tuned for) - on a light terminal colorHeading is
	// near-white and becomes invisible.
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Underline(true).
				Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "254"})

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Background(lipgloss.AdaptiveColor{Light: "255", Dark: "234"}).
			Padding(1, 2)
)

// Column width accounting. cursor+check are fixed; the other columns are
// sized dynamically from actual content (see computeLayout in view.go).
const (
	colCursorW = 2
	colCheckW  = 4

	minNameW   = 20
	maxNameW   = 50
	minBranchW = 16
	minGlyphW  = 14
	maxGlyphW  = 40
	minStateW  = 12
	maxStateW  = 40
)

type layout struct {
	name, branch, glyph, state int
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
