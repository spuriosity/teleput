package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	putio "github.com/putdotio/go-putio"
)

type browserModel struct {
	client        *putio.Client
	files         []putio.File
	cursor        int
	selected      map[int64]bool
	parentID      int64
	parents       []int64
	parentNames   []string
	cursorHistory map[int64]int
	loading     bool
	downloading bool
	deleting    bool
	renaming    bool
	downloadDir string
	width       int
	height      int
	spinner     spinner.Model
}

type filesLoadedMsg struct {
	files    []putio.File
	parentID int64
}

func newBrowserModel(client *putio.Client) browserModel {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(catMauve)
	return browserModel{
		client:        client,
		selected:      make(map[int64]bool),
		cursorHistory: make(map[int64]int),
		downloadDir:   ".",
		spinner:       sp,
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

func (m browserModel) selectedIDs() []int64 {
	ids := make([]int64, 0, len(m.selected))
	for id, sel := range m.selected {
		if sel {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 && len(m.files) > 0 {
		ids = append(ids, m.files[m.cursor].ID)
	}
	return ids
}

func (m browserModel) update(msg tea.Msg) (browserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case filesLoadedMsg:
		m.files = msg.files
		m.parentID = msg.parentID
		if saved, ok := m.cursorHistory[msg.parentID]; ok {
			m.cursor = saved
			if m.cursor >= len(m.files) {
				m.cursor = max(len(m.files)-1, 0)
			}
			delete(m.cursorHistory, msg.parentID)
		} else {
			m.cursor = 0
		}
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

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
		case key.Matches(msg, keys.Top):
			m.cursor = 0
		case key.Matches(msg, keys.Bottom):
			if len(m.files) > 0 {
				m.cursor = len(m.files) - 1
			}
		case key.Matches(msg, keys.Enter):
			if len(m.files) > 0 && m.files[m.cursor].IsDir() {
				m.cursorHistory[m.parentID] = m.cursor
				m.parents = append(m.parents, m.parentID)
				m.parentNames = append(m.parentNames, m.currentDirName())
				m.loading = true
				return m, tea.Batch(m.loadDir(m.files[m.cursor].ID), m.spinner.Tick)
			}
		case key.Matches(msg, keys.Back):
			if len(m.parents) > 0 {
				prev := m.parents[len(m.parents)-1]
				m.parents = m.parents[:len(m.parents)-1]
				m.parentNames = m.parentNames[:len(m.parentNames)-1]
				m.loading = true
				m.selected = make(map[int64]bool)
				return m, tea.Batch(m.loadDir(prev), m.spinner.Tick)
			}
		case key.Matches(msg, keys.Space):
			if len(m.files) > 0 {
				id := m.files[m.cursor].ID
				m.selected[id] = !m.selected[id]
				if !m.selected[id] {
					delete(m.selected, id)
				}
				if m.cursor < len(m.files)-1 {
					m.cursor++
				}
			}
		case key.Matches(msg, keys.SelectAll):
			if len(m.selected) > 0 {
				m.selected = make(map[int64]bool)
			} else {
				for _, f := range m.files {
					m.selected[f.ID] = true
				}
			}
		case key.Matches(msg, keys.Download):
			if len(m.files) > 0 {
				m.downloading = true
			}
		case key.Matches(msg, keys.Delete):
			if len(m.files) > 0 {
				m.deleting = true
			}
		case key.Matches(msg, keys.Rename):
			if len(m.files) > 0 {
				m.renaming = true
			}
		case key.Matches(msg, keys.SetDir):
			// TODO: text input for download dir
		}
	}
	return m, nil
}

func (m browserModel) currentDirName() string {
	if m.parentID == 0 {
		return "Your Files"
	}
	return fmt.Sprintf("ID:%d", m.parentID)
}

func (m browserModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	// Title bar with breadcrumbs
	b.WriteString(m.titleBar())
	b.WriteString("\n")

	if m.loading {
		spacer := strings.Repeat("\n", m.height/3)
		loading := lipgloss.NewStyle().Foreground(catSubtext0).Render(
			m.spinner.View() + " Loading...",
		)
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

	// Calculate visible area
	headerLines := 1
	footerLines := 2
	visibleHeight := m.height - headerLines - footerLines
	if visibleHeight < 1 {
		visibleHeight = 10
	}

	// Viewport scrolling
	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(m.files) {
		end = len(m.files)
		start = end - visibleHeight
		if start < 0 {
			start = 0
		}
	}

	cursorSt := lipgloss.NewStyle().Foreground(catPeach)
	selectedSt := lipgloss.NewStyle().Foreground(catPink)
	dirSt := lipgloss.NewStyle().Bold(true).Foreground(catSapphire)
	normalSt := lipgloss.NewStyle().Foreground(catText)
	sizeSt := lipgloss.NewStyle().Foreground(catOverlay1)

	nameWidth := m.width - 30
	if nameWidth < 20 {
		nameWidth = 40
	}
	contentWidth := m.width - 2

	for i := start; i < end; i++ {
		f := m.files[i]
		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(catPeach).Render("▸ ")
		}

		sel := "  "
		if m.selected[f.ID] {
			sel = lipgloss.NewStyle().Foreground(catPink).Render("● ")
		}

		icon := fileIcon(f)
		name := f.Name
		var line string
		if f.IsDir() {
			line = fmt.Sprintf("%s%s%s %s", cursor, sel, icon, dirSt.Render(name))
		} else {
			size := humanSize(f.Size)
			line = fmt.Sprintf("%s%s%s %-*s %s", cursor, sel, icon, nameWidth, normalSt.Render(name), sizeSt.Render(size))
		}

		if i == m.cursor {
			line = cursorSt.Render(line)
		} else if m.selected[f.ID] {
			line = selectedSt.Render(line)
		}

		// Scrollbar
		scrollChar := " "
		if len(m.files) > visibleHeight {
			thumbPos := int(float64(m.cursor) / float64(len(m.files)-1) * float64(visibleHeight-1))
			lineIdx := i - start
			if lineIdx == thumbPos {
				scrollChar = lipgloss.NewStyle().Foreground(catMauve).Render("┃")
			} else {
				scrollChar = lipgloss.NewStyle().Foreground(catSurface1).Render("│")
			}
		}

		lineLen := lipgloss.Width(line)
		if lineLen < contentWidth {
			line += strings.Repeat(" ", contentWidth-lineLen)
		}

		b.WriteString(line + scrollChar + "\n")
	}

	// Pad remaining lines
	rendered := end - start
	for i := rendered; i < visibleHeight; i++ {
		b.WriteString(strings.Repeat(" ", m.width) + "\n")
	}

	// Status bar
	selCount := len(m.selected)
	left := fmt.Sprintf(" %d items", len(m.files))
	if selCount > 0 {
		left += fmt.Sprintf(" │ %d selected", selCount)
	}
	right := fmt.Sprintf("↓ %s ", m.downloadDir)
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	statusContent := left + strings.Repeat(" ", gap) + right
	b.WriteString(statusBarStyle.Width(m.width).Render(statusContent))
	b.WriteString("\n")

	// Hint bar
	hints := " ↑↓ navigate │ → open │ ← back │ Space select │ d download │ x delete │ r rename │ ? help"
	b.WriteString(hintBarStyle.Width(m.width).Render(hints))

	return b.String()
}

func (m browserModel) titleBar() string {
	brand := lipgloss.NewStyle().Bold(true).Render("teleput")

	crumbs := m.breadcrumbs()
	var trail string
	if len(crumbs) > 1 {
		dimCrumb := lipgloss.NewStyle().Foreground(catCrust)
		brightCrumb := lipgloss.NewStyle().Foreground(catBase).Bold(true)
		sep := lipgloss.NewStyle().Foreground(catCrust).Render(" / ")

		parts := make([]string, len(crumbs))
		for i, c := range crumbs {
			if i == len(crumbs)-1 {
				parts[i] = brightCrumb.Render(c)
			} else {
				parts[i] = dimCrumb.Render(c)
			}
		}
		trail = " │ " + strings.Join(parts, sep)
	}

	content := brand + trail
	return titleBarStyle.Width(m.width).Render(content)
}

func (m browserModel) breadcrumbs() []string {
	parts := []string{"Your Files"}
	parts = append(parts, m.parentNames...)
	return parts
}

func fileIcon(f putio.File) string {
	if f.IsDir() {
		return "📁"
	}
	ct := f.ContentType
	switch {
	case strings.HasPrefix(ct, "video/"):
		return "🎬"
	case strings.HasPrefix(ct, "audio/"):
		return "🎵"
	case strings.HasPrefix(ct, "image/"):
		return "🖼 "
	case strings.Contains(ct, "zip") || strings.Contains(ct, "rar") || strings.Contains(ct, "tar"):
		return "📦"
	case strings.Contains(ct, "pdf"):
		return "📄"
	default:
		return "📄"
	}
}

func humanSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
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
