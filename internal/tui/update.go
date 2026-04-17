package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
				it.Selected = msg.status.CanFF
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
		// Rescan locals to pick up the new clone.
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
		return m, tea.Batch(reenable, fetchAndStatusCmd(msg.name, msg.path))

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
		return m, nil

	case "k", "up":
		m.moveCursor(-1)
		m.ensureCursorVisible()
		return m, nil

	case "g", "home":
		m.setCursor(0)
		m.ensureCursorVisible()
		return m, nil

	case "G", "end":
		m.setCursor(m.itemCount() - 1)
		m.ensureCursorVisible()
		return m, nil

	case "pgdown", "ctrl+d":
		m.moveCursor(m.viewportHeight())
		m.ensureCursorVisible()
		return m, nil

	case "pgup", "ctrl+u":
		m.moveCursor(-m.viewportHeight())
		m.ensureCursorVisible()
		return m, nil

	case " ":
		if m.tab == tabLocal && len(m.locals) > 0 {
			it := m.locals[m.localCursor]
			it.Selected = !it.Selected
		}
		return m, nil

	case "a":
		if m.tab == tabLocal {
			for _, it := range m.locals {
				if it.Status.CanFF {
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
				if !it.Status.CanFF {
					it.Message = "skipped"
					it.Selected = false
					continue
				}
				it.Loading = true
				it.Message = "updating"
				cmds = append(cmds, ffCmd(it.Name, it.Path))
			}
			if len(cmds) > 0 {
				m.flash = fmt.Sprintf("fast-forwarding %d repo(s)...", len(cmds))
				return m, tea.Batch(cmds...)
			}
			m.flash = "nothing to update"
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
			return m, cloneCmd(it.Repo.Name, it.Repo.SSHURL, dest)
		}
		return m, nil
	}
	return m, nil
}

func (m *model) moveCursor(delta int) {
	n := m.itemCount()
	if n == 0 {
		return
	}
	c := m.cursor() + delta
	if c < 0 {
		c = 0
	}
	if c >= n {
		c = n - 1
	}
	m.setCursor(c)
}

func (m *model) cursor() int {
	if m.tab == tabLocal {
		return m.localCursor
	}
	return m.remoteCursor
}

func (m *model) setCursor(c int) {
	if c < 0 {
		c = 0
	}
	if m.tab == tabLocal {
		m.localCursor = c
	} else {
		m.remoteCursor = c
	}
}

func (m *model) itemCount() int {
	if m.tab == tabLocal {
		return len(m.locals)
	}
	return len(m.remotes)
}

func (m *model) scroll(delta int) {
	n := m.itemCount()
	vh := m.viewportHeight()
	if n <= vh {
		return
	}
	off := m.scrollOffset() + delta
	maxOff := n - vh
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	m.setScrollOffset(off)
}

func (m *model) scrollOffset() int {
	if m.tab == tabLocal {
		return m.localScroll
	}
	return m.remoteScroll
}

func (m *model) setScrollOffset(o int) {
	if m.tab == tabLocal {
		m.localScroll = o
	} else {
		m.remoteScroll = o
	}
}

func (m *model) ensureCursorVisible() {
	vh := m.viewportHeight()
	if vh <= 0 {
		return
	}
	cur := m.cursor()
	off := m.scrollOffset()
	switch {
	case cur < off:
		m.setScrollOffset(cur)
	case cur >= off+vh:
		m.setScrollOffset(cur - vh + 1)
	}
}

func (m *model) findLocal(name string) *localItem {
	for _, it := range m.locals {
		if it.Name == name {
			return it
		}
	}
	return nil
}

func (m *model) findRemote(name string) *remoteItem {
	for _, it := range m.remotes {
		if it.Repo.Name == name {
			return it
		}
	}
	return nil
}

// startFetchesIfReady dispatches per-repo refresh once both the local scan
// AND the ADO repo list are done. For repos we already know are archived
// (disabled upstream) or orphan (not in ADO), it runs status-only and
// skips the git fetch - saves ~15s per dead remote on startup.
func (m *model) startFetchesIfReady() tea.Cmd {
	if m.loadingLocals || m.loadingRemotes || m.fetchesStarted {
		return nil
	}
	m.fetchesStarted = true

	remoteMap := m.remoteByName()
	cmds := make([]tea.Cmd, 0, len(m.locals))
	for _, it := range m.locals {
		if m.canSkipFetch(it.Name, remoteMap) {
			cmds = append(cmds, statusOnlyCmd(it.Name, it.Path))
		} else {
			cmds = append(cmds, fetchAndStatusCmd(it.Name, it.Path))
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *model) canSkipFetch(name string, remoteMap map[string]*remoteItem) bool {
	// If we couldn't list remotes at all, play it safe and fetch everything.
	if m.remoteListErr != "" {
		return false
	}
	// Wikis don't appear in /_apis/git/repositories; fetch normally.
	if strings.HasSuffix(name, ".wiki") {
		return false
	}
	r, ok := remoteMap[name]
	if !ok {
		return true // orphan: local folder with no matching ADO repo
	}
	return r.Repo.Disabled // archived upstream
}

func (m *model) reconcileRemotes() {
	local := make(map[string]bool, len(m.locals))
	for _, it := range m.locals {
		local[it.Name] = true
	}
	for _, r := range m.remotes {
		r.Cloned = local[r.Repo.Name]
	}
}
