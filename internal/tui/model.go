package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pszypowicz/optiprime-sync/internal/ado"
	"github.com/pszypowicz/optiprime-sync/internal/config"
	"github.com/pszypowicz/optiprime-sync/internal/gitops"
)



type tab int

const (
	tabLocal tab = iota
	tabRemote
)

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
	scanErr        string
	remoteListErr  string
	prErr          string
	prCounts       map[string]int
	loadingLocals  bool
	loadingRemotes bool
	loadingPRs     bool
	fetchesStarted bool
	flash          string // top-of-screen status line

	// Details panel (toggled with `i` on the Local tab).
	detailsOpen    bool
	detailsCache   map[string]*gitops.Details
	detailsLoading map[string]bool
}

func newModel(cfg *config.Config) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return model{
		cfg:            cfg,
		adoClient:      ado.NewClient(cfg.Org, cfg.Project, cfg.PAT),
		spinner:        sp,
		loadingLocals:  true,
		loadingRemotes: true,
		loadingPRs:     true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		scanLocalsCmd(m.cfg.ScopeRoot),
		listRemotesCmd(m.adoClient),
		fetchPRsCmd(m.adoClient),
	)
}
