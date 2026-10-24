package ui

import (
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jack/teleput/download"
)

type proposal struct {
	file     remoteFile
	proposed string
	ticked   bool
}

type confirmDoneMsg struct{ items []download.Item }
type confirmCancelMsg struct{}

// renameOp renames one put.io item where it stands.
type renameOp struct {
	id      int64
	relPath string
	newName string
}

type confirmRenameMsg struct{ ops []renameOp }

type confirmModel struct {
	rows    []proposal
	cursor  int
	editing bool
	input   textinput.Model
	width   int
	height  int
	// inPlace confirms renames on put.io rather than download paths.
	inPlace bool
}

func newConfirmModel(rows []proposal, width, height int) confirmModel {
	ti := textinput.New()
	ti.CharLimit = 512
	return confirmModel{rows: rows, input: ti, width: width, height: height}
}

func newRenameConfirmModel(rows []proposal, width, height int) confirmModel {
	m := newConfirmModel(rows, width, height)
	m.inPlace = true
	return m
}

func (m confirmModel) update(msg tea.Msg) (confirmModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.editing {
		switch keyMsg.String() {
		case "enter":
			m.rows[m.cursor].proposed = m.input.Value()
			m.rows[m.cursor].ticked = true
			m.editing = false
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
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case " ":
		m.rows[m.cursor].ticked = !m.rows[m.cursor].ticked
	case "e":
		m.input.SetValue(m.rows[m.cursor].proposed)
		m.input.Focus()
		m.editing = true
	case "enter", "a":
		if m.inPlace {
			var ops []renameOp
			for _, r := range m.rows {
				if r.ticked && r.proposed != path.Base(r.file.RelPath) {
					ops = append(ops, renameOp{id: r.file.ID, relPath: r.file.RelPath, newName: r.proposed})
				}
			}
			return m, func() tea.Msg { return confirmRenameMsg{ops: ops} }
		}
		items := make([]download.Item, 0, len(m.rows))
		for _, r := range m.rows {
			path := r.file.RelPath
			if r.ticked {
				path = r.proposed
			}
			items = append(items, download.Item{FileID: r.file.ID, RelPath: path, Size: r.file.Size})
		}
		return m, func() tea.Msg { return confirmDoneMsg{items: items} }
	case "esc", "q":
		return m, func() tea.Msg { return confirmCancelMsg{} }
	}
	return m, nil
}

func (m confirmModel) view() string {
	var b strings.Builder
	heading, hint := "confirm renames", " unticked rows keep their original name"
	if m.inPlace {
		heading, hint = "confirm renames on put.io", " renames apply on put.io · unticked rows are left alone"
	}
	title := lipgloss.NewStyle().Bold(true).Render(heading)
	b.WriteString(titleBarStyle.Width(m.width).Render(title))
	b.WriteString("\n")

	tickedSt := lipgloss.NewStyle().Foreground(catGreen)
	untickedSt := lipgloss.NewStyle().Foreground(catOverlay1)
	origSt := lipgloss.NewStyle().Foreground(catSubtext0)

	listRows := m.height - 4
	start, end := visibleRange(m.cursor, len(m.rows), listRows)
	for i := start; i < end; i++ {
		r := m.rows[i]
		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(catPeach).Render("> ")
		}
		box, st := "[x]", tickedSt
		if !r.ticked {
			box, st = "[ ]", untickedSt
		}
		if m.editing && i == m.cursor {
			b.WriteString(cursor + box + " " + origSt.Render(r.file.RelPath) + " → " + m.input.View() + "\n")
			continue
		}
		b.WriteString(cursor + st.Render(box) + " " + origSt.Render(r.file.RelPath) + " → " + st.Render(r.proposed) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(statusBarStyle.Width(m.width).Render(hint))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("space toggle · e edit · enter approve all · esc cancel"))
	return b.String()
}
