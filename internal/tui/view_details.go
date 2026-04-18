package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pszypowicz/optiprime-sync/internal/gitops"
)

func (m model) renderDetailsOverlay() string {
	if len(m.locals) == 0 || m.localCursor < 0 || m.localCursor >= len(m.locals) {
		return ""
	}
	it := m.locals[m.localCursor]

	var d *gitops.Details
	if m.detailsCache != nil {
		d = m.detailsCache[it.Name]
	}

	title := tableHeaderStyle.Render(fmt.Sprintf("Details · %s", it.Name))
	hint := mutedStyle.Render("[esc/i] close")

	lines := []string{title + "    " + hint, ""}

	if d == nil {
		lines = append(lines, mutedStyle.Render(m.spinner.View()+" loading..."))
		return overlayStyle.Render(strings.Join(lines, "\n"))
	}

	s := it.Status
	branchLine := s.Branch
	if s.Upstream != "" {
		branchLine += mutedStyle.Render(" → ") + s.Upstream
	}
	if !s.BranchIsDefault {
		branchLine += mutedStyle.Render(fmt.Sprintf("   (default: %s)", s.DefaultBranch))
	}
	lines = append(lines, field("Branch", branchLine))

	if d.LastCommitSHA != "" {
		commit := fmt.Sprintf("%s  %s  %s(%s, %s)",
			okStyle.Render(d.LastCommitSHA),
			d.LastCommitSubject,
			mutedStyle.Render(""),
			d.LastCommitAge,
			d.LastCommitAuthor,
		)
		lines = append(lines, field("Commit", commit))
	}

	if len(d.DirtyFiles) > 0 {
		first := d.DirtyFiles[0]
		lines = append(lines, field("Dirty", fmt.Sprintf("%s %s", warnStyle.Render(first.XY), first.Path)))
		more := len(d.DirtyFiles) - 1
		max := 3
		if more < max {
			max = more
		}
		for i := 1; i <= max; i++ {
			f := d.DirtyFiles[i]
			lines = append(lines, field("", fmt.Sprintf("%s %s", warnStyle.Render(f.XY), f.Path)))
		}
		if more > 3 {
			lines = append(lines, field("", mutedStyle.Render(fmt.Sprintf("+%d more", more-3))))
		}
	}

	if len(d.Stashes) > 0 {
		first := d.Stashes[0]
		line := fmt.Sprintf("%s  %s  %s", mutedStyle.Render(first.Ref), first.Subject, mutedStyle.Render("("+first.Age+")"))
		lines = append(lines, field("Stash", line))
		if extra := len(d.Stashes) - 1; extra > 0 {
			lines = append(lines, field("", mutedStyle.Render(fmt.Sprintf("+%d more", extra))))
		}
	}

	var sshURL, webURL string
	if rm := m.remoteByName(); len(rm) > 0 {
		if r, ok := rm[it.Name]; ok {
			sshURL = r.Repo.SSHURL
			webURL = r.Repo.WebURL
		}
	}
	if sshURL == "" {
		sshURL = d.RemoteURL
	}
	if sshURL != "" {
		lines = append(lines, field("SSH", mutedStyle.Render(sshURL)))
	}
	if webURL != "" {
		lines = append(lines, field("Web", mutedStyle.Render(webURL)))
	}

	return overlayStyle.Render(strings.Join(lines, "\n"))
}

// composeOverlay splices a small pre-rendered overlay into the center of
// listPart. Each row the overlay covers is replaced by left-pad + overlay
// row + right-pad (padded with plain spaces), so the overlay's own bg is
// what the user sees while the list underneath is hidden.
func composeOverlay(list, overlay string, innerWidth int) string {
	listLines := strings.Split(list, "\n")
	overlayLines := strings.Split(overlay, "\n")
	if len(overlayLines) == 0 {
		return list
	}

	overlayW := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > overlayW {
			overlayW = w
		}
	}
	if overlayW > innerWidth {
		overlayW = innerWidth
	}

	startY := (len(listLines) - len(overlayLines)) / 2
	if startY < 0 {
		startY = 0
	}
	padLeft := (innerWidth - overlayW) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padLeftStr := strings.Repeat(" ", padLeft)

	for i, ol := range overlayLines {
		y := startY + i
		if y < 0 || y >= len(listLines) {
			continue
		}
		olW := lipgloss.Width(ol)
		padRight := innerWidth - padLeft - olW
		if padRight < 0 {
			padRight = 0
		}
		listLines[y] = padLeftStr + ol + strings.Repeat(" ", padRight)
	}
	return strings.Join(listLines, "\n")
}

// field formats one labeled line of the details panel.
func field(label, value string) string {
	if label == "" {
		return "           " + value
	}
	return mutedStyle.Render(fmt.Sprintf("%-10s", label)) + " " + value
}
