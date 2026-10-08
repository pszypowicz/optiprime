package tui

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/pszypowicz/optiprime/internal/ado"
	"github.com/pszypowicz/optiprime/internal/config"
	"github.com/pszypowicz/optiprime/internal/gitops"
)

type tab int

const (
	tabLocal tab = iota
	tabRemote
)

const maxParallel = 8

type localItem struct {
	Name     string
	Path     string
	Selected bool
	Loading  bool
	Status   gitops.Status
	Err      string
	Message  string // transient post-action message (e.g. "updated")
	PRCount  int    // open PRs authored by the current user
}

type remoteItem struct {
	Repo    ado.Repo
	Cloned  bool
	Cloning bool
	Err     string
	Message string
}

type model struct {
	cfg            *config.Config
	adoClient      *ado.Client
	locals         []*localItem
	remotes        []*remoteItem
	tab            tab
	localCursor    int
	remoteCursor   int
	localScroll    int
	remoteScroll   int
	width, height  int
	spinner        spinner.Model
	styles         styles
	scanErr        string
	remoteListErr  string
	prErr          string
	prCounts       map[string]int
	loadingLocals  bool
	loadingRemotes bool
	loadingPRs     bool
	fetchesStarted bool
	flash          string // top-of-screen status line

	// sem caps concurrent git/clone goroutines launched from background
	// commands. Owned by the model so two concurrent models (e.g. in tests)
	// don't share a global pool.
	sem chan struct{}

	// Details panel (toggled with `i` on the Local tab).
	detailsOpen    bool
	detailsCache   map[string]*gitops.Details
	detailsLoading map[string]bool
}

func newModel(cfg *config.Config) model {
	m := model{
		cfg:           cfg,
		spinner:       spinner.New(spinner.WithSpinner(spinner.Dot)),
		loadingLocals: true,
		sem:           make(chan struct{}, maxParallel),
		// Dark until the terminal answers the background query in Init. A
		// terminal that never answers keeps the dark palette.
		styles: newStyles(true),
	}
	// Without a PAT the REST features (Remote tab, PR counts) are off and
	// their loading flags must stay false - nothing will ever clear them.
	if cfg.RemoteEnabled() {
		m.adoClient = ado.NewClient(cfg.Org, cfg.Project, cfg.PAT)
		m.loadingRemotes = true
		m.loadingPRs = true
	}
	return m
}

func (m model) remoteEnabled() bool {
	return m.adoClient != nil
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.RequestBackgroundColor,
		m.spinner.Tick,
		scanLocalsCmd(m.cfg.ScopeRoot),
	}
	if m.remoteEnabled() {
		cmds = append(cmds, listRemotesCmd(m.adoClient), fetchPRsCmd(m.adoClient))
	}
	return tea.Batch(cmds...)
}
