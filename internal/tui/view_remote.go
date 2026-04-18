package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderRemote() string {
	if m.loadingRemotes && len(m.remotes) == 0 {
		return "  " + m.spinner.View() + " listing " + m.cfg.Org + "/" + m.cfg.Project + " via ADO REST"
	}
	if m.remoteListErr != "" {
		return "  " + errStyle.Render("ADO error: "+sanitizeInline(m.remoteListErr))
	}
	if len(m.remotes) == 0 {
		return mutedStyle.Render("  no repos returned from ADO")
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

	hdr := tableHeaderStyle.Render(
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
		state = mutedStyle.Render(m.spinner.View() + " cloning")
	case it.Cloned:
		marker = "✓"
		state = okStyle.Render("cloned")
	case it.Repo.Disabled:
		marker = "-"
		state = mutedStyle.Render("disabled upstream")
	default:
		state = warnStyle.Render("not cloned")
	}
	if it.Err != "" {
		state = errStyle.Render(it.Err)
	} else if it.Message != "" {
		state = okStyle.Render(it.Message)
	}

	row := cell(colCursorW, cursor) +
		cell(colCheckW, mutedStyle.Render(marker)) +
		cell(L.name, it.Repo.Name) +
		cell(L.branch, state) +
		cell(sshW, mutedStyle.Render(it.Repo.SSHURL))

	switch {
	case i == m.remoteCursor:
		return applyRowBg(row, rowSelected)
	case i%2 == 1:
		return applyRowBg(row, zebraStyle)
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
