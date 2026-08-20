package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/pszypowicz/optiprime/internal/gitops"
)

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
