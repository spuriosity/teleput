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

type deleteCompleteMsg struct{}

type deleteModel struct {
	client  *putio.Client
	fileIDs []int64
	done    bool
	err     error
	width   int
	height  int
	spinner spinner.Model
}

func newDeleteModel(client *putio.Client, fileIDs []int64) deleteModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(catPeach)

	return deleteModel{
		client:  client,
		fileIDs: fileIDs,
		spinner: sp,
	}
}

func (m deleteModel) start() tea.Cmd {
	return func() tea.Msg {
		err := m.client.Files.Delete(context.Background(), m.fileIDs...)
		if err != nil {
			return errMsg{fmt.Errorf("deleting files: %w", err)}
		}
		return deleteCompleteMsg{}
	}
}

func (m deleteModel) update(msg tea.Msg) (deleteModel, tea.Cmd) {
	switch msg := msg.(type) {
	case deleteCompleteMsg:
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

func (m deleteModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(catPeach).
		Bold(true).
		Render("Delete")
	content.WriteString(title + "\n\n")

	count := len(m.fileIDs)
	itemStr := "item"
	if count != 1 {
		itemStr += "s"
	}

	if m.err != nil {
		content.WriteString(errorTextStyle.Render("  ✗ "+m.err.Error()) + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	} else if m.done {
		content.WriteString(successStyle.Render(fmt.Sprintf("  ✓ Deleted %d %s", count, itemStr)) + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	} else {
		content.WriteString(lipgloss.NewStyle().Foreground(catText).Render(
			fmt.Sprintf("  %s Deleting %d %s...", m.spinner.View(), count, itemStr),
		) + "\n")
	}

	panel := warnPanelStyle.Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
