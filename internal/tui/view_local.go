package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m model) renderLocal() string {
	if m.loadingLocals && len(m.locals) == 0 {
		return "  " + m.spinner.View() + " scanning " + m.cfg.ScopeRoot
	}
	if m.scanErr != "" {
		return "  " + m.styles.err.Render("scan error: "+sanitizeInline(m.scanErr))
	}
	if len(m.locals) == 0 {
		return m.styles.muted.Render("  no git repos found in scope root")
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
	widths := m.measureLocalWidths(m.locals[start:end], remoteMap)
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
		return applyRowBg(blank, m.styles.zebra)
	}
	return blank
}

func (m model) renderLocalHeader(L layout) string {
	return m.styles.tableHeader.Render(
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
		branchCell = m.styles.muted.Render(m.spinner.View() + " fetching")
		glyphCell = m.styles.muted.Render("-")
		stateCell = m.styles.muted.Render("fetching")
	case it.Err != "":
		branchCell = m.styles.muted.Render("-")
		glyphCell = m.styles.muted.Render("-")
		stateCell = m.styles.err.Render(it.Err)
	default:
		branchCell = m.renderBranch(it.Status)
		glyphCell = m.renderGlyphs(it.Status, it.PRCount)
		stateCell = m.renderLocalState(it, remoteMap)
	}

	row := cell(colCursorW, cursor) +
		cell(colCheckW, check) +
		cell(L.name, it.Name) +
		cell(L.branch, branchCell) +
		cell(L.glyph, glyphCell) +
		cell(L.state, stateCell)

	switch {
	case i == m.localCursor:
		return applyRowBg(row, m.styles.rowSelected)
	case i%2 == 1:
		return applyRowBg(row, m.styles.zebra)
	default:
		return row
	}
}

func (m model) measureLocalWidths(items []*localItem, rm map[string]*remoteItem) contentWidths {
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
			bcell = m.renderBranch(it.Status)
			gcell = m.renderGlyphs(it.Status, it.PRCount)
			scell = m.renderLocalState(it, rm)
		}
		w.branch = maxi(w.branch, lipgloss.Width(bcell))
		w.glyph = maxi(w.glyph, lipgloss.Width(gcell))
		w.state = maxi(w.state, lipgloss.Width(scell))
	}
	return w
}
