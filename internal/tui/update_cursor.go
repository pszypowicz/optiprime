package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
			cmds = append(cmds, statusOnlyCmd(m.sem, it.Name, it.Path))
		} else {
			cmds = append(cmds, fetchAndStatusCmd(m.sem, it.Name, it.Path))
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *model) canSkipFetch(name string, remoteMap map[string]*remoteItem) bool {
	if m.remoteListErr != "" {
		return false
	}
	if strings.HasSuffix(name, ".wiki") {
		return false
	}
	r, ok := remoteMap[name]
	if !ok {
		return true
	}
	return r.Repo.Disabled
}

// ensureDetailsLoaded kicks off a fetch of git Details for the cursor repo
// when the panel is open and we don't have them cached yet. Returns nil when
// the panel is closed or the repo's details are already loaded / in flight.
func (m *model) ensureDetailsLoaded() tea.Cmd {
	if !m.detailsOpen || m.tab != tabLocal || len(m.locals) == 0 {
		return nil
	}
	if m.localCursor < 0 || m.localCursor >= len(m.locals) {
		return nil
	}
	it := m.locals[m.localCursor]
	if m.detailsCache != nil {
		if _, ok := m.detailsCache[it.Name]; ok {
			return nil
		}
	}
	if m.detailsLoading == nil {
		m.detailsLoading = map[string]bool{}
	}
	if m.detailsLoading[it.Name] {
		return nil
	}
	m.detailsLoading[it.Name] = true
	return fetchDetailsCmd(m.sem, it.Name, it.Path)
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
