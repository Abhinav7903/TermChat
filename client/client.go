package client

import (
	tea "github.com/charmbracelet/bubbletea"
)

// RunClient starts the Bubble Tea TUI and connects to the TermChat server.
func RunClient(host, port string) error {
	p := tea.NewProgram(
		InitialModel(host, port),
		tea.WithAltScreen(),       // use the alternate terminal screen
		tea.WithMouseCellMotion(), // optional: mouse support for scrolling
	)
	_, err := p.Run()
	return err
}
