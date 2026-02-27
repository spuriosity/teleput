package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	catPink      = lipgloss.Color("#f5c2e7")
	catMauve     = lipgloss.Color("#cba6f7")
	catPeach     = lipgloss.Color("#fab387")
	catSapphire  = lipgloss.Color("#74c7ec")
	catText      = lipgloss.Color("#cdd6f4")
	catSubtext1  = lipgloss.Color("#bac2de")
	catSubtext0  = lipgloss.Color("#a6adc8")
	catOverlay1  = lipgloss.Color("#7f849c")
	catSurface1  = lipgloss.Color("#45475a")
	catSurface0  = lipgloss.Color("#313244")
	catBase      = lipgloss.Color("#1e1e2e")
	catCrust     = lipgloss.Color("#11111b")
)

var (
	titleBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(catCrust).
			Background(catMauve).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(catSubtext1).
			Background(catSurface1).
			Padding(0, 1)

	hintBarStyle = lipgloss.NewStyle().
			Foreground(catOverlay1).
			Background(catSurface0).
			Padding(0, 1)
)
