package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyPress(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	if s == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	if s == "esc" {
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func testRows() []proposal {
	return []proposal{
		{file: remoteFile{ID: 1, RelPath: "orig/a.mkv", Size: 10}, proposed: "Movies/A (2001)/A (2001).mkv", ticked: true},
		{file: remoteFile{ID: 2, RelPath: "orig/b.nfo", Size: 5}, proposed: "Other/orig/b.nfo", ticked: false},
	}
}

func runConfirm(t *testing.T, m confirmModel, presses ...string) (confirmModel, tea.Msg) {
	t.Helper()
	var lastMsg tea.Msg
	for _, p := range presses {
		var cmd tea.Cmd
		m, cmd = m.update(keyPress(p))
		if cmd != nil {
			lastMsg = cmd()
		}
	}
	return m, lastMsg
}

func TestShouldToggleRowWithSpace(t *testing.T) {
	m := newConfirmModel(testRows(), 80, 24)
	m, _ = runConfirm(t, m, " ")
	if m.rows[0].ticked {
		t.Fatal("expected space to untick row 0")
	}
}

func TestShouldApproveTickedAsProposedAndUntickedAsOriginal(t *testing.T) {
	m := newConfirmModel(testRows(), 80, 24)
	_, msg := runConfirm(t, m, "enter")

	done, ok := msg.(confirmDoneMsg)
	if !ok {
		t.Fatalf("expected confirmDoneMsg, got %T", msg)
	}
	if done.items[0].RelPath != "Movies/A (2001)/A (2001).mkv" {
		t.Fatalf("ticked row should use proposed path, got %q", done.items[0].RelPath)
	}
	if done.items[1].RelPath != "orig/b.nfo" {
		t.Fatalf("unticked row should keep original path, got %q", done.items[1].RelPath)
	}
}

func TestShouldEditProposedPath(t *testing.T) {
	m := newConfirmModel(testRows(), 80, 24)
	m, _ = runConfirm(t, m, "e")
	if !m.editing {
		t.Fatal("expected e to start editing")
	}
	m.input.SetValue("Movies/Renamed/Renamed.mkv")
	m, _ = runConfirm(t, m, "enter")
	if m.rows[0].proposed != "Movies/Renamed/Renamed.mkv" {
		t.Fatalf("expected edited path, got %q", m.rows[0].proposed)
	}

	_, msg := runConfirm(t, m, "enter")
	done := msg.(confirmDoneMsg)
	if done.items[0].RelPath != "Movies/Renamed/Renamed.mkv" {
		t.Fatalf("approve should use edited path, got %q", done.items[0].RelPath)
	}
}

func TestShouldCancelWholeBatchOnEsc(t *testing.T) {
	m := newConfirmModel(testRows(), 80, 24)
	_, msg := runConfirm(t, m, "esc")
	if _, ok := msg.(confirmCancelMsg); !ok {
		t.Fatalf("expected confirmCancelMsg, got %T", msg)
	}
}
