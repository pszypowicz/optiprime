package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/pszypowicz/optiprime/internal/applog"
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
	title := titleStyle.Render(" optiprime ")
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

func (m model) renderFooter() string {
	var keys string
	if m.tab == tabLocal {
		keys = "[space] toggle  [a] ff-ready  [n] none  [u] update  [i] info  [l] lazygit  [tab] remote  [r] refresh  [q] quit"
	} else {
		keys = "[enter] clone via SSH  [tab] local  [r] refresh  [q] quit"
	}
	return ansi.Truncate(helpStyle.Render(keys), m.innerWidth(), "…")
}
