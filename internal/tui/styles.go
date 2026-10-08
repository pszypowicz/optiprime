package tui

import "charm.land/lipgloss/v2"

// styles holds every style the view uses. The model owns one set and
// rebuilds it when the terminal reports its background color.
type styles struct {
	title, header          lipgloss.Style
	tabActive, tabInactive lipgloss.Style
	zebra, rowSelected     lipgloss.Style
	ok, warn, err, muted   lipgloss.Style
	pr, help               lipgloss.Style
	tableHeader, box       lipgloss.Style
	overlay                lipgloss.Style
}

// newStyles builds the palette for a dark or a light terminal background.
// Every style that sets a background also sets a foreground, so the text
// color never depends on the terminal default.
func newStyles(isDark bool) styles {
	ld := lipgloss.LightDark(isDark)
	pick := func(light, dark string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(ld(lipgloss.Color(light), lipgloss.Color(dark)))
	}

	accent := ld(lipgloss.Color("#5A3FA0"), lipgloss.Color("#7D56F4"))
	heading := ld(lipgloss.Color("230"), lipgloss.Color("229"))
	text := ld(lipgloss.Color("235"), lipgloss.Color("252"))

	return styles{
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(heading).
			Background(accent).
			Padding(0, 1),

		header: pick("240", "241"),

		tabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(heading).
			Background(accent).
			Padding(0, 2),

		tabInactive: pick("245", "244").Padding(0, 2),

		// Zebra stripes for alt rows - subtle enough that colored foregrounds stay readable.
		zebra: lipgloss.NewStyle().
			Foreground(text).
			Background(ld(lipgloss.Color("254"), lipgloss.Color("236"))),

		// Cursor row uses a stronger tint so it stands out even on a zebra row.
		rowSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(text).
			Background(ld(lipgloss.Color("252"), lipgloss.Color("238"))),

		ok:    pick("28", "42"),
		warn:  pick("130", "214"),
		err:   pick("160", "196"),
		muted: pick("245", "244"),
		pr:    pick("26", "39").Bold(true),
		help:  pick("240", "241"),

		// Bold + underlined header. Foreground is chosen for contrast against
		// the terminal background, not against the accent color (which is what
		// heading is tuned for) - on a light terminal heading is near-white and
		// becomes invisible.
		tableHeader: pick("235", "254").Bold(true).Underline(true),

		box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1),

		overlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Foreground(text).
			Background(ld(lipgloss.Color("255"), lipgloss.Color("234"))).
			Padding(1, 2),
	}
}

// Column width accounting. cursor+check are fixed; the other columns are
// sized dynamically from actual content (see computeLayout).
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
