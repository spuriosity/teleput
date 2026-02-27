package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	catRosewater = lipgloss.Color("#f5e0dc")
	catFlamingo  = lipgloss.Color("#f2cdcd")
	catPink      = lipgloss.Color("#f5c2e7")
	catMauve     = lipgloss.Color("#cba6f7")
	catRed       = lipgloss.Color("#f38ba8")
	catMaroon    = lipgloss.Color("#eba0ac")
	catPeach     = lipgloss.Color("#fab387")
	catYellow    = lipgloss.Color("#f9e2af")
	catGreen     = lipgloss.Color("#a6e3a1")
	catTeal      = lipgloss.Color("#94e2d5")
	catSky       = lipgloss.Color("#89dceb")
	catSapphire  = lipgloss.Color("#74c7ec")
	catBlue      = lipgloss.Color("#89b4fa")
	catLavender  = lipgloss.Color("#b4befe")
	catText      = lipgloss.Color("#cdd6f4")
	catSubtext1  = lipgloss.Color("#bac2de")
	catSubtext0  = lipgloss.Color("#a6adc8")
	catOverlay2  = lipgloss.Color("#9399b2")
	catOverlay1  = lipgloss.Color("#7f849c")
	catOverlay0  = lipgloss.Color("#6c7086")
	catSurface2  = lipgloss.Color("#585b70")
	catSurface1  = lipgloss.Color("#45475a")
	catSurface0  = lipgloss.Color("#313244")
	catBase      = lipgloss.Color("#1e1e2e")
	catMantle    = lipgloss.Color("#181825")
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

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(catMauve).
			Padding(1, 2)

	dimTextStyle = lipgloss.NewStyle().
			Foreground(catOverlay1)

	errorPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(catRed).
			Padding(1, 2)

	errorTextStyle = lipgloss.NewStyle().
			Foreground(catRed).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(catGreen).
			Bold(true)

	warnPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(catPeach).
			Padding(1, 2)
)
