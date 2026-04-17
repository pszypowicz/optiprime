package tui

import (
	"github.com/pszypowicz/optiprime-sync/internal/ado"
	"github.com/pszypowicz/optiprime-sync/internal/gitops"
)

type localScannedMsg struct {
	items []*localItem
	err   error
}

type statusMsg struct {
	name   string
	status gitops.Status
	err    error
}

type remoteListedMsg struct {
	repos []ado.Repo
	err   error
}

type prCountsMsg struct {
	counts map[string]int
	err    error
}

type ffDoneMsg struct {
	name   string
	status gitops.Status
	err    error
}

type cloneDoneMsg struct {
	name string
	path string
	err  error
}

type refreshMsg struct{}

type lazygitDoneMsg struct {
	name string
	path string
	err  error
}

type detailsMsg struct {
	name    string
	details gitops.Details
	err     error
}
