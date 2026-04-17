package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/pszypowicz/optiprime-sync/internal/applog"
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

	var listPart string
	if m.tab == tabLocal {
		listPart = m.renderLocal()
		if m.detailsOpen {
			listPart = composeOverlay(listPart, m.renderDetailsOverlay(), m.innerWidth())
		}
	} else {
		listPart = m.renderRemote()
	}
	b.WriteString(listPart)

	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())
	// Explicit width/height so the border hugs the terminal, not the widest
	// line. -2 accounts for the left+right / top+bottom border glyphs.
	return boxStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(b.String())
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
	if m.prErr != "" || m.remoteListErr != "" || m.scanErr != "" {
		extras = append(extras, mutedStyle.Render("log: "+applog.Path()))
	}
	if len(extras) > 0 {
		head += "   " + strings.Join(extras, "   ")
	}

	return ansi.Truncate(head, m.innerWidth(), "…")
}

func (m model) renderTabs() string {
	local := fmt.Sprintf("Local (%d)", len(m.locals))
	remote := fmt.Sprintf("Remote (%d)", len(m.remotes))
	var tabs string
	if m.tab == tabLocal {
		tabs = tabActive.Render(local) + " " + tabInactive.Render(remote)
	} else {
		tabs = tabInactive.Render(local) + " " + tabActive.Render(remote)
	}
	if r := m.scrollRangeText(); r != "" {
		tabs += "   " + mutedStyle.Render(r)
	}
	return tabs
}

// scrollRangeText returns "(start-end of total)" when the viewport can't show
// the whole list, otherwise "".
func (m model) scrollRangeText() string {
	var n, start int
	if m.tab == tabLocal {
		n = len(m.locals)
		start = m.localScroll
	} else {
		n = len(m.remotes)
		start = m.remoteScroll
	}
	vh := m.viewportHeight()
	if n == 0 || n <= vh {
		return ""
	}
	end := start + vh
	if end > n {
		end = n
	}
	return fmt.Sprintf("(%d-%d of %d)", start+1, end, n)
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

	rows := []string{m.renderLocalHeader(L)}
	for i := start; i < end; i++ {
		rows = append(rows, m.renderLocalRow(i, L, remoteMap))
	}
	// Pad any trailing empty viewport rows with full-width zebra stripes so
	// the striping pattern stays intact below the last real row.
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

func (m model) renderFooter() string {
	var keys string
	if m.tab == tabLocal {
		keys = "[space] toggle  [a] ff-ready  [n] none  [u] update  [i] info  [l] lazygit  [tab] remote  [r] refresh  [q] quit"
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

// applyRowBg wraps a row in the given style's ANSI codes and re-applies them
// after every inner "\x1b[0m" reset so the full row width carries the
// background. lipgloss's default cell rendering emits a full reset after each
// styled substring, which otherwise breaks an outer background partway across
// the row.
func applyRowBg(row string, style lipgloss.Style) string {
	prefix := styleOpenSequence(style)
	if prefix == "" {
		return row
	}
	// Re-open the style immediately after every inner reset so the bg (and
	// any bold) stays active across the whole row.
	row = strings.ReplaceAll(row, "\x1b[0m", "\x1b[0m"+prefix)
	return prefix + row + "\x1b[0m"
}

// styleOpenSequence renders a probe character through the given style and
// extracts the opening ANSI escape sequence (everything before the probe).
// Runs the real renderer so adaptive colors resolve to the terminal's actual
// palette codes.
func styleOpenSequence(style lipgloss.Style) string {
	const probe = "\x00"
	out := style.Render(probe)
	if idx := strings.Index(out, probe); idx > 0 {
		return out[:idx]
	}
	return ""
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
	// Only truncate when the content is actually wider than the cell.
	// Otherwise (including the exact-fit case, which is normal for narrow
	// cells like the 2-wide cursor column) keep the content verbatim and let
	// lipgloss pad to width with plain spaces.
	if lipgloss.Width(content) > width {
		budget := width - 1
		if budget < 1 {
			budget = width
		}
		content = ansi.Truncate(content, budget, "…")
	}
	return lipgloss.NewStyle().Width(width).Render(content)
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
	case !s.BranchIsDefault && s.MergedInDefault && !s.Dirty():
		return okStyle.Render("merged → switch & ff")
	case !s.BranchIsDefault && s.MergedInDefault:
		return warnStyle.Render("merged (dirty)")
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
