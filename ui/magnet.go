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

type magnetCompleteMsg struct {
	name string
}

type magnetModel struct {
	client  *putio.Client
	input   textinput.Model
	running bool
	done    bool
	err     error
	name    string
	width   int
	height  int
	spinner spinner.Model
}

func newMagnetModel(client *putio.Client) magnetModel {
	ti := textinput.New()
	ti.Placeholder = "magnet:?xt=urn:btih:... or https://..."
	ti.Focus()
	ti.CharLimit = 2048
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(catMauve)
	ti.TextStyle = lipgloss.NewStyle().Foreground(catText)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(catPeach)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(catMauve)

	return magnetModel{
		client:  client,
		input:   ti,
		spinner: sp,
	}
}

func (m magnetModel) addMagnet(url string) tea.Cmd {
	return func() tea.Msg {
		transfer, err := m.client.Transfers.Add(context.Background(), url, -1, "")
		if err != nil {
			return errMsg{fmt.Errorf("adding transfer: %w", err)}
		}
		return magnetCompleteMsg{name: transfer.Name}
	}
}

func (m magnetModel) update(msg tea.Msg) (magnetModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.done || m.running {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEnter:
			url := strings.TrimSpace(m.input.Value())
			if url == "" {
				return m, nil
			}
			m.running = true
			return m, tea.Batch(m.addMagnet(url), m.spinner.Tick)
		case tea.KeyEsc:
			return m, func() tea.Msg { return cancelMsg{} }
		}

	case magnetCompleteMsg:
		m.done = true
		m.running = false
		m.name = msg.name
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

func (m magnetModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(catMauve).
		Bold(true).
		Render("Add Magnet / URL")
	content.WriteString(title + "\n\n")

	if m.err != nil {
		content.WriteString(errorTextStyle.Render("  \u2717 "+m.err.Error()) + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	} else if m.done {
		content.WriteString(successStyle.Render("  \u2713 Transfer added: "+m.name) + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	} else if m.running {
		content.WriteString(lipgloss.NewStyle().Foreground(catText).Render(
			fmt.Sprintf("  %s Adding transfer...", m.spinner.View()),
		) + "\n")
	} else {
		content.WriteString("  " + m.input.View() + "\n\n")
		content.WriteString(dimTextStyle.Render("  Enter to add, Esc to cancel"))
	}

	panel := panelStyle.Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
