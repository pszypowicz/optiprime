package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pszypowicz/optiprime-sync/internal/config"
)

func Run(cfg *config.Config) error {
	p := tea.NewProgram(newModel(cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
