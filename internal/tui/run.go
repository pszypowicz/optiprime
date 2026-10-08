package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/pszypowicz/optiprime/internal/config"
)

func Run(cfg *config.Config) error {
	p := tea.NewProgram(newModel(cfg))
	_, err := p.Run()
	return err
}
