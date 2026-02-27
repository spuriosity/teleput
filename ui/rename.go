package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	putio "github.com/putdotio/go-putio"
)

type renameCompleteMsg struct{}

type renameModel struct {
	client  *putio.Client
	fileID  int64
	input   textinput.Model
	running bool
	done    bool
	err     error
	width   int
	height  int
	spinner spinner.Model
}

func newRenameModel(client *putio.Client, fileID int64, currentName string) renameModel {
	ti := textinput.New()
	ti.SetValue(currentName)
	ti.Focus()
	ti.CharLimit = 255
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(catMauve)
	ti.TextStyle = lipgloss.NewStyle().Foreground(catText)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(catPeach)
	ti.CursorEnd()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(catMauve)

	return renameModel{
		client:  client,
		fileID:  fileID,
		input:   ti,
		spinner: sp,
	}
}

func (m renameModel) doRename(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.Files.Rename(context.Background(), m.fileID, name)
		if err != nil {
			return errMsg{fmt.Errorf("renaming file: %w", err)}
		}
		return renameCompleteMsg{}
	}
}

func (m renameModel) update(msg tea.Msg) (renameModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.done || m.running {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEnter:
			name := strings.TrimSpace(m.input.Value())
			if name == "" {
				return m, nil
			}
			m.running = true
			return m, tea.Batch(m.doRename(name), m.spinner.Tick)
		case tea.KeyEsc:
			return m, func() tea.Msg { return cancelMsg{} }
		}

	case renameCompleteMsg:
		m.done = true
		m.running = false
		return m, nil

	case errMsg:
		m.err = msg.err
		m.done = true
		m.running = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if !m.running && !m.done {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m renameModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(catMauve).
		Bold(true).
		Render("Rename")
	content.WriteString(title + "\n\n")

	if m.err != nil {
		content.WriteString(errorTextStyle.Render("  ✗ "+m.err.Error()) + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	} else if m.done {
		content.WriteString(successStyle.Render("  ✓ Renamed") + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	} else if m.running {
		content.WriteString(lipgloss.NewStyle().Foreground(catText).Render(
			fmt.Sprintf("  %s Renaming...", m.spinner.View()),
		) + "\n")
	} else {
		content.WriteString("  " + m.input.View() + "\n\n")
		content.WriteString(dimTextStyle.Render("  Enter to confirm, Esc to cancel"))
	}

	panel := panelStyle.Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
