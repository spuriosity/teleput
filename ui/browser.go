package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	putio "github.com/putdotio/go-putio"
)

type browserModel struct {
	client   *putio.Client
	files    []putio.File
	cursor   int
	parentID int64
	parents  []int64
	loading  bool
	width    int
	height   int
}

type filesLoadedMsg struct {
	files    []putio.File
	parentID int64
}

func newBrowserModel(client *putio.Client) browserModel {
	return browserModel{
		client: client,
	}
}

func (m browserModel) loadDir(parentID int64) tea.Cmd {
	return func() tea.Msg {
		children, _, err := m.client.Files.List(context.Background(), parentID)
		if err != nil {
			return errMsg{err}
		}
		sort.Slice(children, func(i, j int) bool {
			iDir := children[i].IsDir()
			jDir := children[j].IsDir()
			if iDir != jDir {
				return iDir
			}
			return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
		})
		return filesLoadedMsg{files: children, parentID: parentID}
	}
}

func (m browserModel) update(msg tea.Msg) (browserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case filesLoadedMsg:
		m.files = msg.files
		m.parentID = msg.parentID
		m.cursor = 0
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if len(m.files) > 0 && m.files[m.cursor].IsDir() {
				m.parents = append(m.parents, m.parentID)
				m.loading = true
				return m, m.loadDir(m.files[m.cursor].ID)
			}
		case key.Matches(msg, keys.Back):
			if len(m.parents) > 0 {
				prev := m.parents[len(m.parents)-1]
				m.parents = m.parents[:len(m.parents)-1]
				m.loading = true
				return m, m.loadDir(prev)
			}
		}
	}
	return m, nil
}

func (m browserModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	// Title bar
	title := lipgloss.NewStyle().Bold(true).Render("teleput")
	b.WriteString(titleBarStyle.Width(m.width).Render(title))
	b.WriteString("\n")

	if m.loading {
		spacer := strings.Repeat("\n", m.height/3)
		loading := lipgloss.NewStyle().Foreground(catSubtext0).Render("Loading...")
		centered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, loading)
		b.WriteString(spacer)
		b.WriteString(centered)
		return b.String()
	}

	if len(m.files) == 0 {
		spacer := strings.Repeat("\n", m.height/3)
		empty := lipgloss.NewStyle().Foreground(catOverlay1).Render("Empty folder")
		centered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, empty)
		b.WriteString(spacer)
		b.WriteString(centered)
		return b.String()
	}

	dirSt := lipgloss.NewStyle().Bold(true).Foreground(catSapphire)
	normalSt := lipgloss.NewStyle().Foreground(catText)
	sizeSt := lipgloss.NewStyle().Foreground(catOverlay1)

	listRows := m.height - 4 // title, blank, status bar, help footer
	start, end := visibleRange(m.cursor, len(m.files), listRows)
	for i := start; i < end; i++ {
		f := m.files[i]
		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(catPeach).Render("> ")
		}

		name := f.Name
		var line string
		if f.IsDir() {
			line = fmt.Sprintf("%s%s %s", cursor, "📁", dirSt.Render(name))
		} else {
			size := humanSize(f.Size)
			line = fmt.Sprintf("%s%s %s  %s", cursor, "📄", normalSt.Render(name), sizeSt.Render(size))
		}

		b.WriteString(line + "\n")
	}

	// Status bar
	status := fmt.Sprintf(" %d/%d items", m.cursor+1, len(m.files))
	b.WriteString("\n")
	b.WriteString(statusBarStyle.Width(m.width).Render(status))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k move · enter open · h back · q quit"))

	return b.String()
}

// visibleRange returns the [start,end) window of rows to render so the
// cursor stays on screen; once past the first page the cursor rides the
// bottom edge.
func visibleRange(cursor, total, rows int) (int, int) {
	if total <= 0 || rows <= 0 {
		return 0, 0
	}
	start := cursor - rows + 1
	if start < 0 {
		start = 0
	}
	end := start + rows
	if end > total {
		end = total
	}
	return start, end
}

func humanSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
