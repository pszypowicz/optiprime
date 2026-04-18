package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pszypowicz/optiprime-sync/internal/gitops"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.scroll(-3)
		case tea.MouseButtonWheelDown:
			m.scroll(3)
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				cmd := m.handleClick(msg.X, msg.Y)
				return m, cmd
			}
		}
		return m, nil

	case localScannedMsg:
		if msg.err != nil {
			m.loadingLocals = false
			m.scanErr = msg.err.Error()
			return m, nil
		}
		m.locals = msg.items
		m.loadingLocals = false
		return m, m.startFetchesIfReady()

	case statusMsg:
		if it := m.findLocal(msg.name); it != nil {
			it.Loading = false
			if msg.err != nil {
				it.Err = msg.err.Error()
			} else {
				it.Status = msg.status
				it.Err = ""
				it.Selected = msg.status.SafeToUpdate()
			}
			if m.prCounts != nil {
				it.PRCount = m.prCounts[it.Name]
			}
		}
		m.reconcileRemotes()
		return m, nil

	case prCountsMsg:
		m.loadingPRs = false
		if msg.err != nil {
			m.prErr = msg.err.Error()
			return m, nil
		}
		m.prCounts = msg.counts
		for _, it := range m.locals {
			it.PRCount = m.prCounts[it.Name]
		}
		return m, nil

	case remoteListedMsg:
		m.loadingRemotes = false
		if msg.err != nil {
			m.remoteListErr = msg.err.Error()
		} else {
			m.remotes = make([]*remoteItem, 0, len(msg.repos))
			for _, r := range msg.repos {
				m.remotes = append(m.remotes, &remoteItem{Repo: r})
			}
			m.reconcileRemotes()
		}
		return m, m.startFetchesIfReady()

	case ffDoneMsg:
		if it := m.findLocal(msg.name); it != nil {
			it.Loading = false
			if msg.err != nil {
				it.Err = msg.err.Error()
				it.Message = "failed"
			} else {
				it.Status = msg.status
				it.Err = ""
				it.Message = "updated"
				it.Selected = false
			}
		}
		return m, nil

	case cloneDoneMsg:
		if it := m.findRemote(msg.name); it != nil {
			it.Cloning = false
			if msg.err != nil {
				it.Err = msg.err.Error()
				it.Message = "clone failed"
			} else {
				it.Cloned = true
				it.Message = "cloned"
				it.Err = ""
			}
		}
		m.loadingLocals = true
		return m, scanLocalsCmd(m.cfg.ScopeRoot)

	case lazygitDoneMsg:
		// lazygit resets terminal modes on exit, including mouse tracking.
		// Re-enable it alongside any follow-up commands.
		reenable := tea.EnableMouseCellMotion
		if msg.err != nil {
			m.flash = "lazygit: " + msg.err.Error()
			return m, reenable
		}
		if it := m.findLocal(msg.name); it != nil {
			it.Loading = true
		}
		m.flash = "re-fetching " + msg.name
		return m, tea.Batch(reenable, fetchAndStatusCmd(m.sem, msg.name, msg.path))

	case detailsMsg:
		delete(m.detailsLoading, msg.name)
		if msg.err == nil {
			if m.detailsCache == nil {
				m.detailsCache = map[string]*gitops.Details{}
			}
			m.detailsCache[msg.name] = &msg.details
		}
		return m, nil

	case refreshMsg:
		m.loadingLocals = true
		m.loadingRemotes = true
		m.loadingPRs = true
		m.fetchesStarted = false
		m.scanErr = ""
		m.remoteListErr = ""
		m.prErr = ""
		return m, tea.Batch(
			scanLocalsCmd(m.cfg.ScopeRoot),
			listRemotesCmd(m.adoClient),
			fetchPRsCmd(m.adoClient),
		)

	default:
		// spinner tick and others
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
		if m.detailsOpen {
			m.detailsOpen = false
		}
		return m, nil

	case "tab":
		if m.tab == tabLocal {
			m.tab = tabRemote
		} else {
			m.tab = tabLocal
		}
		return m, nil

	case "r":
		return m, func() tea.Msg { return refreshMsg{} }

	case "j", "down":
		m.moveCursor(1)
		m.ensureCursorVisible()
		return m, m.ensureDetailsLoaded()

	case "k", "up":
		m.moveCursor(-1)
		m.ensureCursorVisible()
		return m, m.ensureDetailsLoaded()

	case "g", "home":
		m.setCursor(0)
		m.ensureCursorVisible()
		return m, m.ensureDetailsLoaded()

	case "G", "end":
		m.setCursor(m.itemCount() - 1)
		m.ensureCursorVisible()
		return m, m.ensureDetailsLoaded()

	case "pgdown", "ctrl+d":
		m.moveCursor(m.viewportHeight())
		m.ensureCursorVisible()
		return m, m.ensureDetailsLoaded()

	case "pgup", "ctrl+u":
		m.moveCursor(-m.viewportHeight())
		m.ensureCursorVisible()
		return m, m.ensureDetailsLoaded()

	case " ":
		if m.tab == tabLocal && len(m.locals) > 0 {
			it := m.locals[m.localCursor]
			it.Selected = !it.Selected
		}
		return m, nil

	case "a":
		if m.tab == tabLocal {
			for _, it := range m.locals {
				if it.Status.SafeToUpdate() {
					it.Selected = true
				}
			}
		}
		return m, nil

	case "n":
		if m.tab == tabLocal {
			for _, it := range m.locals {
				it.Selected = false
			}
		}
		return m, nil

	case "u":
		if m.tab == tabLocal {
			var cmds []tea.Cmd
			for _, it := range m.locals {
				if !it.Selected {
					continue
				}
				switch it.Status.UpdateAction() {
				case gitops.UpdateFastForward:
					it.Loading = true
					it.Message = "updating"
					cmds = append(cmds, ffCmd(m.sem, it.Name, it.Path))
				case gitops.UpdateSwitchAndFF:
					it.Loading = true
					it.Message = "switching"
					cmds = append(cmds, switchAndFFCmd(m.sem, it.Name, it.Path))
				default:
					it.Message = "skipped"
					it.Selected = false
				}
			}
			if len(cmds) > 0 {
				m.flash = fmt.Sprintf("updating %d repo(s)...", len(cmds))
				return m, tea.Batch(cmds...)
			}
			m.flash = "nothing to update"
		}
		return m, nil

	case "i":
		if m.tab == tabLocal {
			m.detailsOpen = !m.detailsOpen
			if m.detailsOpen {
				return m, m.ensureDetailsLoaded()
			}
		}
		return m, nil

	case "l":
		if m.tab == tabLocal && len(m.locals) > 0 {
			it := m.locals[m.localCursor]
			if _, err := exec.LookPath("lazygit"); err != nil {
				m.flash = "lazygit not on PATH - install via `brew install lazygit`"
				return m, nil
			}
			cmd := exec.Command("lazygit", "-p", it.Path)
			name := it.Name
			path := it.Path
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return lazygitDoneMsg{name: name, path: path, err: err}
			})
		}
		return m, nil

	case "enter":
		if m.tab == tabRemote && len(m.remotes) > 0 {
			it := m.remotes[m.remoteCursor]
			if it.Cloned || it.Cloning {
				return m, nil
			}
			if it.Repo.SSHURL == "" {
				it.Err = "no sshUrl in ADO metadata"
				return m, nil
			}
			dest := filepath.Join(m.cfg.ScopeRoot, it.Repo.Name)
			it.Cloning = true
			it.Message = "cloning"
			return m, cloneCmd(m.sem, it.Repo.Name, it.Repo.SSHURL, dest)
		}
		return m, nil
	}
	return m, nil
}

// handleClick maps terminal-absolute (x, y) to tab switches and cursor
// placement. Layout:
//
//	Y=0           : top border
//	Y=1           : header
//	Y=2           : tabs (click anywhere on this row toggles the tab)
//	Y=3           : blank
//	Y=4           : list header row
//	Y=5..5+vh-1   : data rows (click moves cursor; click in the checkbox
//	                cell also toggles selection on the local tab)
//
// X=0 is the left border, X=1 is the box's left padding, so the content
// begins at X=2.
func (m *model) handleClick(x, y int) tea.Cmd {
	switch {
	case y == 2:
		if m.tab == tabLocal {
			m.tab = tabRemote
		} else {
			m.tab = tabLocal
		}
		return nil
	}

	firstDataY := 5
	vh := m.viewportHeight()
	if y < firstDataY || y >= firstDataY+vh {
		return nil
	}

	idx := (y - firstDataY) + m.scrollOffset()
	if idx < 0 || idx >= m.itemCount() {
		return nil
	}
	m.setCursor(idx)

	contentX := x - 2 // strip border + left padding
	inCheck := contentX >= colCursorW && contentX < colCursorW+colCheckW
	if m.tab == tabLocal && inCheck && idx < len(m.locals) {
		m.locals[idx].Selected = !m.locals[idx].Selected
	}
	return m.ensureDetailsLoaded()
}
