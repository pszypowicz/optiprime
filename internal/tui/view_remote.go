package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m model) renderRemote() string {
	if m.loadingRemotes && len(m.remotes) == 0 {
		return "  " + m.spinner.View() + " listing " + m.cfg.Org + "/" + m.cfg.Project + " via ADO REST"
	}
	if m.remoteListErr != "" {
		return "  " + m.styles.err.Render("ADO error: "+sanitizeInline(m.remoteListErr))
	}
	if len(m.remotes) == 0 {
		return m.styles.muted.Render("  no repos returned from ADO")
	}

	vh := m.viewportHeight()
	n := len(m.remotes)
	start := m.remoteScroll
	end := start + vh
	if end > n {
		end = n
	}

	widths := measureRemoteWidths(m.remotes[start:end])
	L := computeLayout(m.width, widths)

	sshW := m.innerWidth() - colCursorW - colCheckW - L.name - L.branch
	if sshW < 20 {
		sshW = 20
	}

	hdr := m.styles.tableHeader.Render(
		cell(colCursorW, "") +
			cell(colCheckW, "") +
			cell(L.name, "repo") +
			cell(L.branch, "state") +
			cell(sshW, "ssh"),
	)

	rows := []string{hdr}
	for i := start; i < end; i++ {
		rows = append(rows, m.renderRemoteRow(i, L, sshW))
	}
	for padIdx := 0; len(rows) < vh+1; padIdx++ {
		virtualIdx := end + padIdx
		rows = append(rows, m.emptyStripedRow(virtualIdx))
	}
	return strings.Join(rows, "\n")
}

func (m model) renderRemoteRow(i int, L layout, sshW int) string {
	it := m.remotes[i]
	cursor := "  "
	if i == m.remoteCursor {
		cursor = "> "
	}
	marker := "+"
	var state string
	switch {
	case it.Cloning:
		marker = "*"
		state = m.styles.muted.Render(m.spinner.View() + " cloning")
	case it.Cloned:
		marker = "✓"
		state = m.styles.ok.Render("cloned")
	case it.Repo.Disabled:
		marker = "-"
		state = m.styles.muted.Render("disabled upstream")
	default:
		state = m.styles.warn.Render("not cloned")
	}
	if it.Err != "" {
		state = m.styles.err.Render(it.Err)
	} else if it.Message != "" {
		state = m.styles.ok.Render(it.Message)
	}

	row := cell(colCursorW, cursor) +
		cell(colCheckW, m.styles.muted.Render(marker)) +
		cell(L.name, it.Repo.Name) +
		cell(L.branch, state) +
		cell(sshW, m.styles.muted.Render(it.Repo.SSHURL))

	switch {
	case i == m.remoteCursor:
		return applyRowBg(row, m.styles.rowSelected)
	case i%2 == 1:
		return applyRowBg(row, m.styles.zebra)
	default:
		return row
	}
}

func measureRemoteWidths(items []*remoteItem) contentWidths {
	w := contentWidths{}
	for _, it := range items {
		w.name = maxi(w.name, lipgloss.Width(it.Repo.Name))
		w.branch = maxi(w.branch, 18)
	}
	return w
}
