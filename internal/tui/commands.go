package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pszypowicz/optiprime-sync/internal/ado"
	"github.com/pszypowicz/optiprime-sync/internal/gitops"
	"github.com/pszypowicz/optiprime-sync/internal/scanner"
)

const maxParallel = 8

var sem = make(chan struct{}, maxParallel)

func scanLocalsCmd(root string) tea.Cmd {
	return func() tea.Msg {
		repos, err := scanner.FindRepos(root)
		if err != nil {
			return localScannedMsg{err: err}
		}
		items := make([]*localItem, 0, len(repos))
		for _, r := range repos {
			items = append(items, &localItem{
				Name:    r.Name,
				Path:    r.Path,
				Loading: true,
			})
		}
		return localScannedMsg{items: items}
	}
}

func fetchAndStatusCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		if err := gitops.Fetch(path); err != nil {
			return statusMsg{name: name, err: err}
		}
		st, err := gitops.GetStatus(path)
		if err != nil {
			return statusMsg{name: name, err: err}
		}
		return statusMsg{name: name, status: st}
	}
}

// statusOnlyCmd runs git status without fetching. Used for repos we already
// know are archived/orphan in ADO so we don't wait on a doomed fetch.
func statusOnlyCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		st, err := gitops.GetStatus(path)
		if err != nil {
			return statusMsg{name: name, err: err}
		}
		return statusMsg{name: name, status: st}
	}
}

func listRemotesCmd(c *ado.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		repos, err := c.ListRepos(ctx)
		return remoteListedMsg{repos: repos, err: err}
	}
}

func fetchPRsCmd(c *ado.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		uid, err := c.AuthUserID(ctx)
		if err != nil {
			return prCountsMsg{err: err}
		}
		counts, err := c.MyOpenPRs(ctx, uid)
		return prCountsMsg{counts: counts, err: err}
	}
}

func ffCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		if err := gitops.FastForward(path); err != nil {
			return ffDoneMsg{name: name, err: err}
		}
		st, err := gitops.GetStatus(path)
		if err != nil {
			return ffDoneMsg{name: name, err: err}
		}
		return ffDoneMsg{name: name, status: st}
	}
}

// switchAndFFCmd checks out the default branch and fast-forwards it.
// Used for repos whose feature branch work is already in origin/<default>.
func switchAndFFCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		if err := gitops.SwitchAndFF(path); err != nil {
			return ffDoneMsg{name: name, err: err}
		}
		st, err := gitops.GetStatus(path)
		if err != nil {
			return ffDoneMsg{name: name, err: err}
		}
		return ffDoneMsg{name: name, status: st}
	}
}

func fetchDetailsCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		d, err := gitops.GetDetails(path)
		return detailsMsg{name: name, details: d, err: err}
	}
}

func cloneCmd(name, sshURL, dest string) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		if err := gitops.Clone(sshURL, dest); err != nil {
			return cloneDoneMsg{name: name, err: err}
		}
		return cloneDoneMsg{name: name, path: dest}
	}
}
