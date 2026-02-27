package ui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	putio "github.com/putdotio/go-putio"
)

type downloadModel struct {
	client       *putio.Client
	token        string
	fileIDs      []int64
	dir          string
	status       string
	filename     string
	totalBytes   int64
	writtenBytes int64
	speed        float64
	done         bool
	width        int
	height       int
	sub          chan tea.Msg
	progress     progress.Model
	spinner      spinner.Model
}

type downloadCompleteMsg struct{ status string }
type downloadProgressMsg struct {
	written int64
	total   int64
	speed   float64
}
type downloadStartedMsg struct {
	filename string
	url      string
	total    int64
}

func newDownloadModel(client *putio.Client, token string, fileIDs []int64, dir string) downloadModel {
	p := progress.New(
		progress.WithGradient(string(catMauve), string(catPink)),
		progress.WithoutPercentage(),
	)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(catMauve)

	return downloadModel{
		client:   client,
		token:    token,
		fileIDs:  fileIDs,
		dir:      dir,
		status:   "Preparing...",
		sub:      make(chan tea.Msg, 100),
		progress: p,
		spinner:  sp,
	}
}

func waitForProgress(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func (m downloadModel) start() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		if len(m.fileIDs) == 1 {
			url, err := m.client.Files.URL(ctx, m.fileIDs[0], false)
			if err != nil {
				return errMsg{fmt.Errorf("getting download URL: %w", err)}
			}
			file, err := m.client.Files.Get(ctx, m.fileIDs[0])
			if err != nil {
				return errMsg{fmt.Errorf("getting file info: %w", err)}
			}
			return downloadStartedMsg{filename: file.Name, url: url, total: file.Size}
		}

		zipID, err := m.client.Zips.Create(ctx, m.fileIDs...)
		if err != nil {
			return errMsg{fmt.Errorf("creating zip: %w", err)}
		}

		for i := 0; i < 300; i++ {
			time.Sleep(2 * time.Second)
			zip, err := m.client.Zips.Get(ctx, zipID)
			if err != nil {
				continue
			}
			if zip.URL != "" {
				return downloadStartedMsg{
					filename: fmt.Sprintf("putio-%d.zip", zipID),
					url:      zip.URL,
					total:    zip.Size,
				}
			}
		}
		return errMsg{fmt.Errorf("zip creation timed out")}
	}
}

func (m downloadModel) update(msg tea.Msg) (downloadModel, tea.Cmd) {
	switch msg := msg.(type) {
	case downloadStartedMsg:
		m.filename = msg.filename
		m.totalBytes = msg.total
		m.status = "Downloading..."
		go doDownload(msg.url, m.dir, msg.filename, m.sub)
		return m, waitForProgress(m.sub)

	case downloadProgressMsg:
		m.writtenBytes = msg.written
		m.totalBytes = msg.total
		m.speed = msg.speed
		var pct float64
		if m.totalBytes > 0 {
			pct = float64(m.writtenBytes) / float64(m.totalBytes)
		}
		cmd := m.progress.SetPercent(pct)
		return m, tea.Batch(cmd, waitForProgress(m.sub))

	case downloadCompleteMsg:
		m.done = true
		m.status = msg.status
		cmd := m.progress.SetPercent(1.0)
		return m, cmd

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m downloadModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	labelStyle := lipgloss.NewStyle().Foreground(catOverlay1).Width(12).Align(lipgloss.Right)
	valueStyle := lipgloss.NewStyle().Foreground(catText)
	pctStyle := lipgloss.NewStyle().Foreground(catMauve).Bold(true)

	panelWidth := m.width - 10
	if panelWidth > 70 {
		panelWidth = 70
	}
	if panelWidth < 40 {
		panelWidth = 40
	}
	innerWidth := panelWidth - 6

	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(catMauve).
		Bold(true).
		Render("Download")
	content.WriteString(title + "\n\n")

	row := func(label, value string) string {
		return labelStyle.Render(label+" ") + valueStyle.Render(value)
	}
	content.WriteString(row("Files", fmt.Sprintf("%d", len(m.fileIDs))) + "\n")
	content.WriteString(row("Directory", m.dir) + "\n")
	if m.filename != "" {
		content.WriteString(row("Filename", m.filename) + "\n")
	}

	if !m.done && m.totalBytes == 0 {
		content.WriteString(row("Status", m.spinner.View()+" "+m.status) + "\n")
	} else if m.done {
		content.WriteString(row("Status", successStyle.Render("Complete")) + "\n")
	} else {
		content.WriteString(row("Status", m.status) + "\n")
	}

	if m.totalBytes > 0 {
		content.WriteString("\n")
		m.progress.Width = innerWidth
		content.WriteString(m.progress.View() + "\n")

		pct := float64(m.writtenBytes) / float64(m.totalBytes) * 100
		stats := fmt.Sprintf("%s / %s", humanSize(m.writtenBytes), humanSize(m.totalBytes))
		if m.speed > 0 {
			stats += fmt.Sprintf("  %s/s", humanSize(int64(m.speed)))
		}
		statsLine := dimTextStyle.Render(stats) + "  " + pctStyle.Render(fmt.Sprintf("%.1f%%", pct))
		content.WriteString(statsLine + "\n")
	} else if m.writtenBytes > 0 {
		content.WriteString("\n")
		stats := fmt.Sprintf("Downloaded: %s", humanSize(m.writtenBytes))
		if m.speed > 0 {
			stats += fmt.Sprintf("  (%s/s)", humanSize(int64(m.speed)))
		}
		content.WriteString(dimTextStyle.Render(stats) + "\n")
	}

	if m.done {
		content.WriteString("\n")
		content.WriteString(successStyle.Render("  ✓ Download complete") + "\n\n")
		content.WriteString(dimTextStyle.Render("  Press Esc to return"))
	}

	panel := panelStyle.Width(panelWidth).Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func doDownload(url, dir, filename string, ch chan<- tea.Msg) {
	resp, err := http.Get(url)
	if err != nil {
		ch <- errMsg{fmt.Errorf("downloading: %w", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- errMsg{fmt.Errorf("download failed: HTTP %d", resp.StatusCode)}
		return
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		ch <- errMsg{fmt.Errorf("creating directory: %w", err)}
		return
	}

	destPath := filepath.Join(dir, filename)
	out, err := os.Create(destPath)
	if err != nil {
		ch <- errMsg{fmt.Errorf("creating file: %w", err)}
		return
	}
	defer out.Close()

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 64*1024)
	start := time.Now()
	lastReport := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				ch <- errMsg{fmt.Errorf("writing file: %w", writeErr)}
				return
			}
			written += int64(n)

			if time.Since(lastReport) > 200*time.Millisecond {
				elapsed := time.Since(start).Seconds()
				speed := 0.0
				if elapsed > 0 {
					speed = float64(written) / elapsed
				}
				ch <- downloadProgressMsg{
					written: written,
					total:   total,
					speed:   speed,
				}
				lastReport = time.Now()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			ch <- errMsg{fmt.Errorf("reading response: %w", readErr)}
			return
		}
	}

	ch <- downloadCompleteMsg{status: fmt.Sprintf("Saved to %s", destPath)}
}
