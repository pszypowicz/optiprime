package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/pszypowicz/optiprime-sync/internal/gitops"
)

// chromeLines: border(2) + header(1) + tabs(1) + blank(1) + list-hdr(1) + blank(1) + footer(1) + 1 for safety.
const chromeLines = 9

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	if m.tab == tabLocal {
		b.WriteString(m.renderLocal())
	} else {
		b.WriteString(m.renderRemote())
	}

	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())
	return boxStyle.Render(b.String())
}

func (m model) innerWidth() int {
	// terminal width minus box border(2) minus box padding(2)
	w := m.width - 4
	if w < 40 {
		w = 40
	}
	return w
}

func (m model) viewportHeight() int {
	h := m.height - chromeLines
	if h < 3 {
		h = 3
	}
	return h
}

func (m model) renderHeader() string {
	title := titleStyle.Render(" optiprime-sync ")
	meta := headerStyle.Render(fmt.Sprintf(" %s / %s   %s", m.cfg.Org, m.cfg.Project, m.cfg.ScopeRoot))
	head := title + meta

	extras := []string{}
	if m.flash != "" {
		extras = append(extras, mutedStyle.Render(m.flash))
	}
	if m.prErr != "" {
		extras = append(extras, warnStyle.Render("PRs: "+m.prErr))
	}
	if len(extras) > 0 {
		head += "   " + strings.Join(extras, "   ")
	}

	return ansi.Truncate(head, m.innerWidth(), "…")
}

func (m model) renderTabs() string {
	local := fmt.Sprintf("Local (%d)", len(m.locals))
	remote := fmt.Sprintf("Remote (%d)", len(m.remotes))
	if m.tab == tabLocal {
		return tabActive.Render(local) + " " + tabInactive.Render(remote)
	}
	return tabInactive.Render(local) + " " + tabActive.Render(remote)
}

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

	rows := []string{m.renderLocalHeader(L, n, start, end)}
	for i := start; i < end; i++ {
		rows = append(rows, m.renderLocalRow(i, L, remoteMap))
	}
	for len(rows) < vh+1 {
		rows = append(rows, "")
	}
	return strings.Join(rows, "\n")
}

func (m model) renderLocalHeader(L layout, total, start, end int) string {
	hdr := mutedStyle.Render(
		cell(colCursorW, "") +
			cell(colCheckW, "") +
			cell(L.name, "repo") +
			cell(L.branch, "branch") +
			cell(L.glyph, "status") +
			cell(L.state, "state"),
	)
	if total > (end - start) {
		hdr += "   " + mutedStyle.Render(fmt.Sprintf("(%d-%d of %d)", start+1, end, total))
	}
	return hdr
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

	if i == m.localCursor {
		return rowSelected.Render(row)
	}
	return row
}

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

	hdr := mutedStyle.Render(
		cell(colCursorW, "") +
			cell(colCheckW, "") +
			cell(L.name, "repo") +
			cell(L.branch, "state") +
			cell(sshW, "ssh"),
	)
	if n > (end - start) {
		hdr += "   " + mutedStyle.Render(fmt.Sprintf("(%d-%d of %d)", start+1, end, n))
	}

	rows := []string{hdr}
	for i := start; i < end; i++ {
		rows = append(rows, m.renderRemoteRow(i, L, sshW))
	}
	for len(rows) < vh+1 {
		rows = append(rows, "")
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

	if i == m.remoteCursor {
		return rowSelected.Render(row)
	}
	return row
}

func (m model) renderFooter() string {
	var keys string
	if m.tab == tabLocal {
		keys = "[space] toggle  [a] ff-ready  [n] none  [u] update  [l] lazygit  [tab] remote  [r] refresh  [q] quit"
	} else {
		keys = "[enter] clone via SSH  [tab] local  [r] refresh  [q] quit"
	}
	return ansi.Truncate(helpStyle.Render(keys), m.innerWidth(), "…")
}

type contentWidths struct {
	name, branch, glyph, state int
}

// computeLayout picks column widths so natural content fits first, then gives
// the remainder of the line to the branch column. Branch is the flex column
// because it's the one with the widest variance in real data.
func computeLayout(termWidth int, c contentWidths) layout {
	inner := termWidth - 4 // box border(2) + padding(2)
	if inner < 40 {
		inner = 40
	}
	data := inner - colCursorW - colCheckW

	name := clamp(c.name+1, minNameW, maxNameW)
	glyph := clamp(c.glyph+1, minGlyphW, maxGlyphW)
	state := clamp(c.state+1, minStateW, maxStateW)

	branch := data - name - glyph - state

	// Branch too tight? Borrow from over-allocated name first, then glyph/state.
	for branch < minBranchW {
		switch {
		case name > minNameW:
			name--
		case state > minStateW:
			state--
		case glyph > minGlyphW:
			glyph--
		default:
			branch = minBranchW
		}
		branch = data - name - glyph - state
	}

	return layout{name: name, branch: branch, glyph: glyph, state: state}
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

func measureRemoteWidths(items []*remoteItem) contentWidths {
	w := contentWidths{}
	for _, it := range items {
		w.name = maxi(w.name, lipgloss.Width(it.Repo.Name))
		// state text is short; fixed-ish
		w.branch = maxi(w.branch, 18)
	}
	return w
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m model) remoteByName() map[string]*remoteItem {
	out := make(map[string]*remoteItem, len(m.remotes))
	for _, r := range m.remotes {
		out[r.Repo.Name] = r
	}
	return out
}

// cell sanitizes content (collapses newlines/tabs so multi-line errors
// can't break the grid), truncates ANSI-aware to width-1, then pads to exact
// visible width. The -1 guarantees at least one trailing space, so a cell
// whose content fills the column never visually touches the next one.
func cell(width int, content string) string {
	if width <= 0 {
		return ""
	}
	content = sanitizeInline(content)
	budget := width - 1
	if budget < 1 {
		budget = width
	}
	truncated := ansi.Truncate(content, budget, "…")
	return lipgloss.NewStyle().Width(width).Render(truncated)
}

func sanitizeInline(s string) string {
	// Git/ADO errors include \n and \t; replace with single spaces so one
	// cell's content stays on one visual row.
	r := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ", "\t", " ")
	return r.Replace(s)
}

func renderBranch(s gitops.Status) string {
	if s.Detached {
		return warnStyle.Render("(detached)")
	}
	if !s.BranchIsDefault {
		return warnStyle.Render(s.Branch) + mutedStyle.Render(" ["+s.DefaultBranch+"]")
	}
	return s.Branch
}

func renderGlyphs(s gitops.Status, prCount int) string {
	var parts []string
	if s.Ahead > 0 || s.Behind > 0 {
		parts = append(parts, mutedStyle.Render(fmt.Sprintf("↑%d↓%d", s.Ahead, s.Behind)))
	}
	if s.Staged > 0 {
		parts = append(parts, okStyle.Render(fmt.Sprintf("+%d", s.Staged)))
	}
	if s.Unstaged > 0 {
		parts = append(parts, warnStyle.Render(fmt.Sprintf("~%d", s.Unstaged)))
	}
	if s.Conflicts > 0 {
		parts = append(parts, errStyle.Render(fmt.Sprintf("!%d", s.Conflicts)))
	}
	if s.Untracked > 0 {
		parts = append(parts, mutedStyle.Render(fmt.Sprintf("?%d", s.Untracked)))
	}
	if s.Stashes > 0 {
		parts = append(parts, mutedStyle.Render(fmt.Sprintf("⚑%d", s.Stashes)))
	}
	if s.InProgress != gitops.OpNone {
		parts = append(parts, errStyle.Render("["+string(s.InProgress)+"]"))
	}
	base := mutedStyle.Render("clean")
	if len(parts) > 0 {
		base = strings.Join(parts, " ")
	}
	if prCount > 0 {
		base += " " + prStyle.Render(fmt.Sprintf("[%d PR]", prCount))
	}
	return base
}

func renderLocalState(it *localItem, remoteMap map[string]*remoteItem) string {
	if it.Message != "" {
		return okStyle.Render(it.Message)
	}
	if len(remoteMap) > 0 {
		if r, ok := remoteMap[it.Name]; ok {
			if r.Repo.Disabled {
				return mutedStyle.Render("archived upstream")
			}
		} else if !strings.HasSuffix(it.Name, ".wiki") {
			return errStyle.Render("not in ADO")
		}
	}
	return renderState(it.Status)
}

func renderState(s gitops.Status) string {
	if s.InProgress != gitops.OpNone {
		return errStyle.Render(strings.ToLower(string(s.InProgress)))
	}
	switch {
	case s.CanFF:
		return okStyle.Render("ff-ready")
	case s.Ahead == 0 && s.Behind == 0:
		return okStyle.Render("up-to-date")
	case s.Ahead > 0 && s.Behind > 0:
		return warnStyle.Render("diverged")
	case s.Ahead > 0:
		return warnStyle.Render("ahead")
	case s.Behind > 0 && !s.BranchIsDefault:
		return mutedStyle.Render("behind (other branch)")
	case s.Behind > 0 && s.Dirty():
		return warnStyle.Render("behind (dirty)")
	default:
		return mutedStyle.Render("-")
	}
}
