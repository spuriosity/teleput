package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	putio "github.com/putdotio/go-putio"
)

type transfersLoadedMsg struct {
	transfers []putio.Transfer
}

type transfersTickMsg struct{}

type transfersModel struct {
	client    *putio.Client
	transfers []putio.Transfer
	cursor    int
	selected  map[int64]bool
	loading   bool
	diskInfo  string
	width     int
	height    int
	spinner   spinner.Model

	// Flags for root model transitions
	cancelling    bool
	retrying      bool
	addingMagnet  bool
	cleaning      bool
	browsingFiles bool
	browseFileID  int64
}

func newTransfersModel(client *putio.Client) transfersModel {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(catMauve)
	return transfersModel{
		client:   client,
		selected: make(map[int64]bool),
		spinner:  sp,
	}
}

func (m transfersModel) loadTransfers() tea.Cmd {
	return func() tea.Msg {
		transfers, err := m.client.Transfers.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return transfersLoadedMsg{transfers: transfers}
	}
}

func transfersTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return transfersTickMsg{}
	})
}

func (m transfersModel) selectedIDs() []int64 {
	ids := make([]int64, 0, len(m.selected))
	for id, sel := range m.selected {
		if sel {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 && len(m.transfers) > 0 {
		ids = append(ids, m.transfers[m.cursor].ID)
	}
	return ids
}

func (m transfersModel) update(msg tea.Msg) (transfersModel, tea.Cmd) {
	switch msg := msg.(type) {
	case transfersLoadedMsg:
		m.transfers = msg.transfers
		m.loading = false
		if m.cursor >= len(m.transfers) {
			m.cursor = max(len(m.transfers)-1, 0)
		}
		return m, transfersTick()

	case transfersTickMsg:
		return m, m.loadTransfers()

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
			if m.cursor < len(m.transfers)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Top):
			m.cursor = 0
		case key.Matches(msg, keys.Bottom):
			if len(m.transfers) > 0 {
				m.cursor = len(m.transfers) - 1
			}
		case key.Matches(msg, keys.Space):
			if len(m.transfers) > 0 {
				id := m.transfers[m.cursor].ID
				m.selected[id] = !m.selected[id]
				if !m.selected[id] {
					delete(m.selected, id)
				}
				if m.cursor < len(m.transfers)-1 {
					m.cursor++
				}
			}
		case key.Matches(msg, keys.SelectAll):
			if len(m.selected) > 0 {
				m.selected = make(map[int64]bool)
			} else {
				for _, t := range m.transfers {
					m.selected[t.ID] = true
				}
			}
		case key.Matches(msg, keys.Delete):
			if len(m.transfers) > 0 {
				m.cancelling = true
			}
		case key.Matches(msg, keys.Retry):
			if len(m.transfers) > 0 && m.transfers[m.cursor].Status == "ERROR" {
				m.retrying = true
			}
		case key.Matches(msg, keys.Clean):
			m.cleaning = true
		case key.Matches(msg, keys.Enter):
			if len(m.transfers) > 0 && m.transfers[m.cursor].FileID != 0 {
				m.browsingFiles = true
				m.browseFileID = m.transfers[m.cursor].FileID
			}
		case key.Matches(msg, keys.AddMagnet):
			m.addingMagnet = true
		}
	}
	return m, nil
}

func (m transfersModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	// Title bar
	bg := catMauve
	brand := lipgloss.NewStyle().Bold(true).Background(bg).Render("teleput")
	label := lipgloss.NewStyle().Foreground(catBase).Background(bg).Bold(true).Render("Transfers")
	left := brand + lipgloss.NewStyle().Background(bg).Render(" │ ") + label
	if m.diskInfo != "" {
		right := lipgloss.NewStyle().Foreground(catBase).Background(bg).Render(m.diskInfo)
		padding := 2 // titleBarStyle has Padding(0, 1)
		gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - padding
		if gap < 1 {
			gap = 1
		}
		filler := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", gap))
		b.WriteString(titleBarStyle.Width(m.width).Render(left + filler + right))
	} else {
		b.WriteString(titleBarStyle.Width(m.width).Render(left))
	}
	b.WriteString("\n")

	if m.loading {
		spacer := strings.Repeat("\n", m.height/3)
		loading := lipgloss.NewStyle().Foreground(catSubtext0).Render(
			m.spinner.View() + " Loading transfers...",
		)
		centered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, loading)
		b.WriteString(spacer)
		b.WriteString(centered)
		return b.String()
	}

	if len(m.transfers) == 0 {
		spacer := strings.Repeat("\n", m.height/3)
		empty := lipgloss.NewStyle().Foreground(catOverlay1).Render("No transfers")
		centered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, empty)
		b.WriteString(spacer)
		b.WriteString(centered)
		b.WriteString("\n\n")
		hint := lipgloss.NewStyle().Foreground(catOverlay0).Render("Press m to add a magnet link")
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, hint))
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
	if end > len(m.transfers) {
		end = len(m.transfers)
		start = end - visibleHeight
		if start < 0 {
			start = 0
		}
	}

	cursorSt := lipgloss.NewStyle().Foreground(catPeach)
	selectedSt := lipgloss.NewStyle().Foreground(catPink)
	normalSt := lipgloss.NewStyle().Foreground(catText)
	sizeSt := lipgloss.NewStyle().Foreground(catOverlay1)
	contentWidth := m.width - 2

	nameWidth := m.width - 45
	if nameWidth < 20 {
		nameWidth = 20
	}

	for i := start; i < end; i++ {
		t := m.transfers[i]
		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(catPeach).Render("\u25b8 ")
		}

		sel := "  "
		if m.selected[t.ID] {
			sel = lipgloss.NewStyle().Foreground(catPink).Render("\u25cf ")
		}

		icon := transferIcon(t.Status)
		pct := fmt.Sprintf("%3d%%", t.PercentDone)
		speed := ""
		if t.DownloadSpeed > 0 {
			speed = humanSize(int64(t.DownloadSpeed)) + "/s"
		}
		statusStr := transferStatusStr(t.Status)

		name := t.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		line := fmt.Sprintf("%s%s%s %-*s %s  %7s  %s",
			cursor, sel, icon, nameWidth, normalSt.Render(name),
			sizeSt.Render(pct), sizeSt.Render(speed), statusStr)

		if i == m.cursor {
			line = cursorSt.Render(line)
		} else if m.selected[t.ID] {
			line = selectedSt.Render(line)
		}

		// Scrollbar
		scrollChar := " "
		if len(m.transfers) > visibleHeight {
			thumbPos := int(float64(m.cursor) / float64(len(m.transfers)-1) * float64(visibleHeight-1))
			lineIdx := i - start
			if lineIdx == thumbPos {
				scrollChar = lipgloss.NewStyle().Foreground(catMauve).Render("|")
			} else {
				scrollChar = lipgloss.NewStyle().Foreground(catSurface1).Render("|")
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
	left = fmt.Sprintf(" %d transfers", len(m.transfers))
	if selCount > 0 {
		left += fmt.Sprintf(" | %d selected", selCount)
	}
	right := ""
	if len(m.transfers) > 0 && m.cursor < len(m.transfers) {
		t := m.transfers[m.cursor]
		if t.PeersConnected > 0 {
			right += fmt.Sprintf("%d peers", t.PeersConnected)
		}
		// Timing info based on status
		switch t.Status {
		case "DOWNLOADING":
			if t.CreatedAt != nil {
				elapsed := int64(time.Since(t.CreatedAt.Time).Seconds())
				if elapsed > 0 {
					if right != "" {
						right += " | "
					}
					right += "↓ " + formatDuration(elapsed)
				}
			}
			if t.EstimatedTime > 0 {
				if right != "" {
					right += " | "
				}
				right += "ETA " + formatDuration(t.EstimatedTime)
			}
		case "SEEDING":
			if t.SecondsSeeding > 0 {
				if right != "" {
					right += " | "
				}
				right += "seeding " + formatDuration(int64(t.SecondsSeeding))
			}
		case "COMPLETED":
			if t.CreatedAt != nil && t.FinishedAt != nil {
				dur := int64(t.FinishedAt.Time.Sub(t.CreatedAt.Time).Seconds())
				if dur > 0 {
					if right != "" {
						right += " | "
					}
					right += "took " + formatDuration(dur)
				}
			}
		case "IN_QUEUE", "WAITING":
			if t.CreatedAt != nil {
				waiting := int64(time.Since(t.CreatedAt.Time).Seconds())
				if waiting > 0 {
					if right != "" {
						right += " | "
					}
					right += "waiting " + formatDuration(waiting)
				}
			}
		default:
			if t.EstimatedTime > 0 {
				if right != "" {
					right += " | "
				}
				right += "ETA " + formatDuration(t.EstimatedTime)
			}
		}
		if t.ErrorMessage != "" {
			if right != "" {
				right += " | "
			}
			right += t.ErrorMessage
		}
		right += " "
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	statusContent := left + strings.Repeat(" ", gap) + right
	b.WriteString(statusBarStyle.Width(m.width).Render(statusContent))
	b.WriteString("\n")

	// Hint bar
	hints := " Tab files | → browse | Space select | m magnet | x cancel | R retry | C clean | ? help"
	b.WriteString(hintBarStyle.Width(m.width).Render(hints))

	return b.String()
}

func transferIcon(status string) string {
	switch status {
	case "DOWNLOADING":
		return lipgloss.NewStyle().Foreground(catSapphire).Render("↓")
	case "COMPLETED", "SEEDING":
		return lipgloss.NewStyle().Foreground(catGreen).Render("✓")
	case "ERROR":
		return lipgloss.NewStyle().Foreground(catRed).Render("✗")
	case "IN_QUEUE", "WAITING":
		return lipgloss.NewStyle().Foreground(catYellow).Render("⏳")
	default:
		return lipgloss.NewStyle().Foreground(catOverlay1).Render("·")
	}
}

func transferStatusStr(status string) string {
	switch status {
	case "DOWNLOADING":
		return lipgloss.NewStyle().Foreground(catSapphire).Render("DOWNLOADING")
	case "COMPLETED":
		return lipgloss.NewStyle().Foreground(catGreen).Render("COMPLETED")
	case "SEEDING":
		return lipgloss.NewStyle().Foreground(catGreen).Render("SEEDING")
	case "ERROR":
		return lipgloss.NewStyle().Foreground(catRed).Render("ERROR")
	case "IN_QUEUE":
		return lipgloss.NewStyle().Foreground(catYellow).Render("IN_QUEUE")
	case "WAITING":
		return lipgloss.NewStyle().Foreground(catYellow).Render("WAITING")
	default:
		return lipgloss.NewStyle().Foreground(catOverlay1).Render(status)
	}
}

func formatDuration(seconds int64) string {
	if seconds <= 0 {
		return ""
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
