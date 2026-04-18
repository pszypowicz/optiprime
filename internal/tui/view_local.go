package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderLocal() string {
	if m.loadingLocals && len(m.locals) == 0 {
		return "  " + m.spinner.View() + " scanning " + m.cfg.ScopeRoot
	}
	if m.scanErr != "" {
		return "  " + errStyle.Render("scan error: "+sanitizeInline(m.scanErr))
	}
	if len(m.locals) == 0 {
		return mutedStyle.Render("  no git repos found in scope root")
	}

	remoteMap := m.remoteByName()

	vh := m.viewportHeight()
	n := len(m.locals)
	start := m.localScroll
	end := start + vh
	if end > n {
		end = n
	}

	// Measure only the rows we're about to render so dynamic sizing matches
	// what's actually on screen (scrolling doesn't shift column widths).
	widths := measureLocalWidths(m.locals[start:end], remoteMap)
	L := computeLayout(m.width, widths)

	rows := []string{m.renderLocalHeader(L)}
	for i := start; i < end; i++ {
		rows = append(rows, m.renderLocalRow(i, L, remoteMap))
	}
	for padIdx := 0; len(rows) < vh+1; padIdx++ {
		virtualIdx := end + padIdx
		rows = append(rows, m.emptyStripedRow(virtualIdx))
	}
	return strings.Join(rows, "\n")
}

// emptyStripedRow returns a blank row padded to the inner content width. Odd
// rows carry the zebra bg so empty space continues the alternating pattern.
func (m model) emptyStripedRow(virtualIdx int) string {
	blank := strings.Repeat(" ", m.innerWidth())
	if virtualIdx%2 == 1 {
		return applyRowBg(blank, zebraStyle)
	}
	return blank
}

func (m model) renderLocalHeader(L layout) string {
	return tableHeaderStyle.Render(
		cell(colCursorW, "") +
			cell(colCheckW, "") +
			cell(L.name, "repo") +
			cell(L.branch, "branch") +
			cell(L.glyph, "status") +
			cell(L.state, "state"),
	)
}

func (m model) renderLocalRow(i int, L layout, remoteMap map[string]*remoteItem) string {
	it := m.locals[i]
	cursor := "  "
	if i == m.localCursor {
		cursor = "> "
	}
	check := "[ ]"
	if it.Selected {
		check = "[x]"
	}

	var branchCell, glyphCell, stateCell string
	switch {
	case it.Loading:
		branchCell = mutedStyle.Render(m.spinner.View() + " fetching")
		glyphCell = mutedStyle.Render("-")
		stateCell = mutedStyle.Render("fetching")
	case it.Err != "":
		branchCell = mutedStyle.Render("-")
		glyphCell = mutedStyle.Render("-")
		stateCell = errStyle.Render(it.Err)
	default:
		branchCell = renderBranch(it.Status)
		glyphCell = renderGlyphs(it.Status, it.PRCount)
		stateCell = renderLocalState(it, remoteMap)
	}

	row := cell(colCursorW, cursor) +
		cell(colCheckW, check) +
		cell(L.name, it.Name) +
		cell(L.branch, branchCell) +
		cell(L.glyph, glyphCell) +
		cell(L.state, stateCell)

	switch {
	case i == m.localCursor:
		return applyRowBg(row, rowSelected)
	case i%2 == 1:
		return applyRowBg(row, zebraStyle)
	default:
		return row
	}
}

func measureLocalWidths(items []*localItem, rm map[string]*remoteItem) contentWidths {
	w := contentWidths{}
	for _, it := range items {
		w.name = maxi(w.name, lipgloss.Width(it.Name))
		var bcell, gcell, scell string
		switch {
		case it.Loading:
			bcell = "spinner fetching"
			gcell = "-"
			scell = "fetching"
		case it.Err != "":
			bcell = "-"
			gcell = "-"
			scell = sanitizeInline(it.Err)
		default:
			bcell = renderBranch(it.Status)
			gcell = renderGlyphs(it.Status, it.PRCount)
			scell = renderLocalState(it, rm)
		}
		w.branch = maxi(w.branch, lipgloss.Width(bcell))
		w.glyph = maxi(w.glyph, lipgloss.Width(gcell))
		w.state = maxi(w.state, lipgloss.Width(scell))
	}
	return w
}
