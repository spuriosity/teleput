package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmMsg struct{}
type cancelMsg struct{}

type confirmModel struct {
	message string
	count   int
	width   int
	height  int
}

func newConfirmModel(count int) confirmModel {
	msg := fmt.Sprintf("Delete %d item", count)
	if count != 1 {
		msg += "s"
	}
	msg += "?"
	return confirmModel{
		message: msg,
		count:   count,
	}
}

func (m confirmModel) update(msg tea.Msg) (confirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Escape):
			return m, func() tea.Msg { return cancelMsg{} }
		case msg.String() == "y":
			return m, func() tea.Msg { return confirmMsg{} }
		}
	}
	return m, nil
}

func (m confirmModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(catPeach).
		Bold(true).
		Render("Confirm Delete")
	content.WriteString(title + "\n\n")

	content.WriteString(lipgloss.NewStyle().Foreground(catText).Render(m.message))
	content.WriteString("\n\n")
	content.WriteString(dimTextStyle.Render("y to confirm, Esc to cancel"))

	panel := warnPanelStyle.Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
