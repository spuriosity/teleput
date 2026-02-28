package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	putio "github.com/putdotio/go-putio"
)

type transferCancelCompleteMsg struct{}

type transferCancelModel struct {
	client  *putio.Client
	ids     []int64
	done    bool
	err     error
	width   int
	height  int
	spinner spinner.Model
}

func newTransferCancelModel(client *putio.Client, ids []int64) transferCancelModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(catPeach)

	return transferCancelModel{
		client:  client,
		ids:     ids,
		spinner: sp,
	}
}

func (m transferCancelModel) start() tea.Cmd {
	return func() tea.Msg {
		err := m.client.Transfers.Cancel(context.Background(), m.ids...)
		if err != nil {
			return errMsg{fmt.Errorf("cancelling transfers: %w", err)}
		}
		return transferCancelCompleteMsg{}
	}
}

func (m transferCancelModel) update(msg tea.Msg) (transferCancelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case transferCancelCompleteMsg:
		m.done = true
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case errMsg:
		m.err = msg.err
		m.done = true
		return m, nil
	}
	return m, nil
}

func (m transferCancelModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(catPeach).
		Bold(true).
		Render("Cancel Transfers")
	content.WriteString(title + "\n\n")

	count := len(m.ids)
	itemStr := "transfer"
	if count != 1 {
		itemStr += "s"
	}

	if m.err != nil {
		content.WriteString(errorTextStyle.Render("  \u2717 "+m.err.Error()) + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	} else if m.done {
		content.WriteString(successStyle.Render(fmt.Sprintf("  \u2713 Cancelled %d %s", count, itemStr)) + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	} else {
		content.WriteString(lipgloss.NewStyle().Foreground(catText).Render(
			fmt.Sprintf("  %s Cancelling %d %s...", m.spinner.View(), count, itemStr),
		) + "\n")
	}

	panel := warnPanelStyle.Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
