package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type settingsField int

const (
	fieldPutioToken settingsField = iota
	fieldGeminiToken
	fieldDownloadDir
	fieldCount
)

var settingsLabels = [fieldCount]string{"put.io token", "gemini token", "download dir"}
var settingsSecret = [fieldCount]bool{true, true, false}

// settingsSavedMsg carries one edited field for Model to apply and persist.
type settingsSavedMsg struct {
	field settingsField
	value string
}
type settingsCancelMsg struct{}

type settingsModel struct {
	values  [fieldCount]string
	cursor  settingsField
	editing bool
	input   textinput.Model
	note    string
	width   int
	height  int
}

func newSettingsModel(putioToken, geminiToken, downloadDir string, width, height int) settingsModel {
	ti := textinput.New()
	ti.CharLimit = 512
	return settingsModel{
		values: [fieldCount]string{putioToken, geminiToken, downloadDir},
		input:  ti,
		width:  width,
		height: height,
	}
}

func (m settingsModel) update(msg tea.Msg) (settingsModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.editing {
		switch keyMsg.String() {
		case "enter":
			m.values[m.cursor] = m.input.Value()
			m.editing = false
			field, value := m.cursor, m.input.Value()
			m.note = "saved"
			return m, func() tea.Msg { return settingsSavedMsg{field: field, value: value} }
		case "esc":
			m.editing = false
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		if m.cursor < fieldCount-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter", "e":
		m.input.SetValue(m.values[m.cursor])
		m.input.Focus()
		m.editing = true
		m.note = ""
	case "esc", "q", "s":
		return m, func() tea.Msg { return settingsCancelMsg{} }
	}
	return m, nil
}

func mask(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("•", len(s))
	}
	return strings.Repeat("•", 8) + s[len(s)-4:]
}

func (m settingsModel) view() string {
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Render("settings")
	b.WriteString(titleBarStyle.Width(m.width).Render(title))
	b.WriteString("\n\n")

	labelSt := lipgloss.NewStyle().Foreground(catSapphire).Width(14)
	valueSt := lipgloss.NewStyle().Foreground(catText)

	for f := settingsField(0); f < fieldCount; f++ {
		cursor := "  "
		if f == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(catPeach).Render("> ")
		}
		if m.editing && f == m.cursor {
			b.WriteString(cursor + labelSt.Render(settingsLabels[f]) + m.input.View() + "\n")
			continue
		}
		val := m.values[f]
		if settingsSecret[f] && val != "" {
			val = mask(val)
		}
		if val == "" {
			val = "(not set)"
		}
		b.WriteString(cursor + labelSt.Render(settingsLabels[f]) + valueSt.Render(val) + "\n")
	}

	b.WriteString("\n")
	status := " changes save immediately · put.io token applies on next launch"
	if m.note != "" {
		status = " " + m.note + " ·" + status
	}
	b.WriteString(statusBarStyle.Width(m.width).Render(status))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k move · enter edit/save · esc back"))
	return b.String()
}
