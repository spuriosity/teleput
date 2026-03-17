package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	putio "github.com/putdotio/go-putio"

	"github.com/jack/teleput/config"
	"github.com/jack/teleput/organize"
)

type organizePhase int

const (
	phaseClassifying organizePhase = iota
	phasePlanReady
	phaseExecuting
	phaseDone
)

type organizeModel struct {
	client   *putio.Client
	cfg      config.OrganizerConfig
	folderID int64
	folder   string
	phase    organizePhase
	plan     organize.Plan
	pathCfg  *organize.PathConfig
	sub      chan organize.Result
	progress organize.Result
	err      error
	width    int
	height   int
	spinner  spinner.Model
	// Plan scroll state
	scrollOffset int
}

// Messages
type organizeClassifyDoneMsg struct {
	plan    organize.Plan
	pathCfg *organize.PathConfig
	err     error
}

type organizeProgressMsg struct {
	result organize.Result
}

type organizeCompleteMsg struct{}

func newOrganizeModel(client *putio.Client, cfg config.OrganizerConfig, folderID int64, folderName string) organizeModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(catMauve)

	return organizeModel{
		client:   client,
		cfg:      cfg,
		folderID: folderID,
		folder:   folderName,
		phase:    phaseClassifying,
		sub:      make(chan organize.Result, 100),
		spinner:  sp,
	}
}

func (m organizeModel) startClassify() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Collect files
		files, err := organize.CollectFiles(ctx, m.client, m.folderID)
		if err != nil {
			return organizeClassifyDoneMsg{err: err}
		}

		if len(files) == 0 {
			return organizeClassifyDoneMsg{err: fmt.Errorf("no files to organize")}
		}

		// Resolve library paths
		pathCfg, err := organize.ResolvePathConfig(ctx, m.client, m.cfg)
		if err != nil {
			return organizeClassifyDoneMsg{err: err}
		}

		// Classify with Claude
		classifications, err := organize.Classify(ctx, files)
		if err != nil {
			return organizeClassifyDoneMsg{err: err}
		}

		// Build plan
		plan := organize.BuildPlan(classifications, pathCfg, m.folder)

		return organizeClassifyDoneMsg{plan: plan, pathCfg: pathCfg}
	}
}

func (m organizeModel) startExecute() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		organize.Execute(ctx, m.client, m.plan, m.pathCfg, m.sub)
		return organizeCompleteMsg{}
	}
}

func waitForOrganizeProgress(sub chan organize.Result) tea.Cmd {
	return func() tea.Msg {
		return organizeProgressMsg{result: <-sub}
	}
}

func (m organizeModel) update(msg tea.Msg) (organizeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case organizeClassifyDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = phaseDone
			return m, nil
		}
		m.plan = msg.plan
		m.pathCfg = msg.pathCfg
		if len(m.plan.Actions) == 0 {
			m.err = fmt.Errorf("nothing to organize — all files classified as other/unknown")
			m.phase = phaseDone
			return m, nil
		}
		m.phase = phasePlanReady
		return m, nil

	case organizeProgressMsg:
		m.progress = msg.result
		if msg.result.Err != nil {
			m.err = msg.result.Err
			m.phase = phaseDone
			return m, nil
		}
		return m, waitForOrganizeProgress(m.sub)

	case organizeCompleteMsg:
		m.phase = phaseDone
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch m.phase {
		case phasePlanReady:
			switch {
			case key.Matches(msg, keys.Up):
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
			case key.Matches(msg, keys.Down):
				m.scrollOffset++
			case msg.String() == "y":
				m.phase = phaseExecuting
				return m, tea.Batch(m.startExecute(), waitForOrganizeProgress(m.sub), m.spinner.Tick)
			}
		}
	}
	return m, nil
}

func (m organizeModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	panelWidth := m.width - 10
	if panelWidth > 80 {
		panelWidth = 80
	}
	if panelWidth < 40 {
		panelWidth = 40
	}

	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(catMauve).
		Bold(true).
		Render("Organize")
	content.WriteString(title)
	if m.folder != "" {
		content.WriteString(dimTextStyle.Render(fmt.Sprintf("  %s", m.folder)))
	}
	content.WriteString("\n\n")

	switch m.phase {
	case phaseClassifying:
		content.WriteString(m.spinner.View() + " Analyzing files with Claude...\n")
		content.WriteString(dimTextStyle.Render("This may take a moment"))

	case phasePlanReady:
		content.WriteString(m.planView(panelWidth - 6))

	case phaseExecuting:
		pct := ""
		if m.progress.Total > 0 {
			pct = fmt.Sprintf(" (%d/%d)", m.progress.Completed, m.progress.Total)
		}
		content.WriteString(m.spinner.View() + " Executing..." + pct + "\n")
		if m.progress.Current != "" {
			content.WriteString(dimTextStyle.Render("  " + m.progress.Current))
		}

	case phaseDone:
		if m.err != nil {
			content.WriteString(errorTextStyle.Render("Error") + "\n\n")
			content.WriteString(lipgloss.NewStyle().Foreground(catText).Render(m.err.Error()) + "\n")
		} else {
			content.WriteString(successStyle.Render("  ✓ Organization complete") + "\n\n")
			content.WriteString(dimTextStyle.Render(fmt.Sprintf("  %d renames, %d moves, %d deletes",
				m.plan.RenameCount, m.plan.MoveCount, m.plan.DeleteCount)))
		}
		content.WriteString("\n\n")
		content.WriteString(dimTextStyle.Render("Esc to return"))
	}

	panel := panelStyle.Width(panelWidth).Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func (m organizeModel) planView(innerWidth int) string {
	var b strings.Builder

	// Summary line
	summary := fmt.Sprintf("Summary: %d renames, %d moves, %d deletes, %d new folders",
		m.plan.RenameCount, m.plan.MoveCount, m.plan.DeleteCount, m.plan.CreateCount)
	b.WriteString(lipgloss.NewStyle().Foreground(catSubtext1).Render(summary) + "\n\n")

	createStyle := lipgloss.NewStyle().Foreground(catGreen)
	renameStyle := lipgloss.NewStyle().Foreground(catYellow)
	moveStyle := lipgloss.NewStyle().Foreground(catSapphire)
	deleteStyle := lipgloss.NewStyle().Foreground(catRed)

	// Calculate visible lines
	maxVisible := m.height - 14
	if maxVisible < 5 {
		maxVisible = 5
	}

	// Clamp scroll offset
	if m.scrollOffset > len(m.plan.Actions)-maxVisible {
		m.scrollOffset = len(m.plan.Actions) - maxVisible
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	end := m.scrollOffset + maxVisible
	if end > len(m.plan.Actions) {
		end = len(m.plan.Actions)
	}

	for i := m.scrollOffset; i < end; i++ {
		a := m.plan.Actions[i]
		var line string
		switch a.Type {
		case organize.ActionCreateFolder:
			line = createStyle.Render("  + " + a.Description())
		case organize.ActionRename:
			line = renameStyle.Render("  ~ " + a.Description())
		case organize.ActionMove:
			line = moveStyle.Render("  → " + a.Description())
		case organize.ActionDelete:
			line = deleteStyle.Render("  ✗ " + a.Description())
		}

		// Truncate if too long
		if lipgloss.Width(line) > innerWidth {
			line = line[:innerWidth-1] + "…"
		}
		b.WriteString(line + "\n")
	}

	if len(m.plan.Actions) > maxVisible {
		scrollInfo := fmt.Sprintf("  (%d more — ↑↓ to scroll)", len(m.plan.Actions)-maxVisible)
		b.WriteString(dimTextStyle.Render(scrollInfo) + "\n")
	}

	b.WriteString("\n")
	confirm := lipgloss.NewStyle().Foreground(catGreen).Bold(true).Render("y")
	cancel := lipgloss.NewStyle().Foreground(catOverlay1).Render("Esc")
	b.WriteString("  " + confirm + " confirm │ " + cancel + " cancel")

	return b.String()
}
