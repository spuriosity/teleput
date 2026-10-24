package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	putio "github.com/putdotio/go-putio"

	"github.com/jack/teleput/config"
	"github.com/jack/teleput/download"
	"github.com/jack/teleput/naming"
)

type screen int

const (
	screenBrowse screen = iota
	screenConfirm
	screenSettings
)

type deleter interface {
	Delete(ctx context.Context, ids ...int64) error
}

type Model struct {
	client   *putio.Client
	cfg      *config.Config
	dl       *download.Manager
	namer    *naming.Client
	lister   lister
	deleter  deleter
	saveCfg  func(*config.Config) error
	width    int
	height   int
	screen   screen
	browser  browserModel
	confirm  confirmModel
	settings settingsModel
	err      error
	quitting bool

	glyphs    map[int64]string
	activeDl  *download.Event
	queued    int
	note      string
	deleteFor *putio.File
}

func NewModel(client *putio.Client, cfg *config.Config, dl *download.Manager, namer *naming.Client) Model {
	m := Model{
		client:  client,
		cfg:     cfg,
		dl:      dl,
		namer:   namer,
		saveCfg: config.Save,
		browser: newBrowserModel(client),
		glyphs:  make(map[int64]string),
	}
	if client != nil {
		m.lister = client.Files
		m.deleter = client.Files
	}
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.browser.loadDir(0)}
	if m.dl != nil {
		cmds = append(cmds, listenDownloads(m.dl.Events()))
	}
	return tea.Batch(cmds...)
}

type downloadEventMsg struct{ ev download.Event }

func listenDownloads(ch <-chan download.Event) tea.Cmd {
	return func() tea.Msg { return downloadEventMsg{ev: <-ch} }
}

// renamePlanMsg is the async result of walk + Gemini normalisation.
type renamePlanMsg struct {
	rows      []proposal
	geminiErr error
}

type deleteDoneMsg struct {
	parentID int64
	err      error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browser.width = msg.Width
		m.browser.height = msg.Height
		m.confirm.width = msg.Width
		m.confirm.height = msg.Height
		m.settings.width = msg.Width
		m.settings.height = msg.Height
		return m, nil

	case downloadEventMsg:
		m = m.applyDownloadEvent(msg.ev)
		return m, listenDownloads(m.dl.Events())

	case renamePlanMsg:
		return m.handleRenamePlan(msg)

	case confirmDoneMsg:
		m.screen = screenBrowse
		m.enqueue(msg.items)
		return m, nil

	case confirmCancelMsg:
		m.screen = screenBrowse
		m.note = "download cancelled"
		return m, nil

	case settingsSavedMsg:
		switch msg.field {
		case fieldPutioToken:
			m.cfg.OAuthToken = msg.value
		case fieldGeminiToken:
			m.cfg.GeminiToken = msg.value
			if m.namer != nil {
				m.namer.Token = msg.value
			} else if msg.value != "" {
				m.namer = naming.NewClient(msg.value)
			}
		case fieldDownloadDir:
			m.cfg.DownloadDir = msg.value
			if m.dl != nil {
				m.dl.SetDestDir(m.cfg.EffectiveDownloadDir())
			}
		}
		if err := m.saveCfg(m.cfg); err != nil {
			m.err = err
		}
		return m, nil

	case settingsCancelMsg:
		m.screen = screenBrowse
		return m, nil

	case deleteDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.note = "deleted from put.io"
		m.browser.loading = true
		return m, m.browser.loadDir(msg.parentID)

	case errMsg:
		m.err = msg.err
		return m, nil

	case filesLoadedMsg:
		m.err = nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.routeToScreen(msg)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	editing := (m.screen == screenConfirm && m.confirm.editing) ||
		(m.screen == screenSettings && m.settings.editing)
	if !editing && key.Matches(msg, keys.Quit) && m.screen == screenBrowse && m.deleteFor == nil {
		m.quitting = true
		return m, tea.Quit
	}

	if m.screen == screenBrowse {
		if m.deleteFor != nil {
			return m.handleDeleteConfirmKey(msg)
		}
		switch msg.String() {
		case "d":
			return m.startDownload()
		case "D":
			if f, ok := m.browser.selected(); ok {
				m.deleteFor = &f
			}
			return m, nil
		case "x":
			if f, ok := m.browser.selected(); ok && m.dl != nil {
				m.dl.Cancel(f.ID)
			}
			return m, nil
		case "s":
			m.settings = newSettingsModel(m.cfg.OAuthToken, m.cfg.GeminiToken, m.cfg.EffectiveDownloadDir(), m.width, m.height)
			m.screen = screenSettings
			return m, nil
		}
	}

	return m.routeToScreen(msg)
}

func (m Model) routeToScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.screen {
	case screenBrowse:
		m.browser, cmd = m.browser.update(msg)
	case screenConfirm:
		m.confirm, cmd = m.confirm.update(msg)
	case screenSettings:
		m.settings, cmd = m.settings.update(msg)
	}
	return m, cmd
}

func (m Model) handleDeleteConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := *m.deleteFor
	m.deleteFor = nil
	if msg.String() != "y" {
		m.note = "delete cancelled"
		return m, nil
	}
	parentID := m.browser.parentID
	del := m.deleter
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return deleteDoneMsg{parentID: parentID, err: del.Delete(ctx, f.ID)}
	}
}

func (m Model) startDownload() (tea.Model, tea.Cmd) {
	roots := m.browser.markedRoots()
	if len(roots) == 0 {
		f, ok := m.browser.selected()
		if !ok {
			return m, nil
		}
		roots = []putio.File{f}
	}
	m.browser.marked = make(map[int64]putio.File)
	l, namer := m.lister, m.namer
	m.note = "planning download…"
	return m, func() tea.Msg {
		collectCtx, cancelCollect := context.WithTimeout(context.Background(), 60*time.Second)
		var files []remoteFile
		seen := make(map[int64]bool)
		for _, root := range roots {
			collected, err := collectFiles(collectCtx, l, root)
			if err != nil {
				cancelCollect()
				return errMsg{err}
			}
			for _, rf := range collected {
				if seen[rf.ID] {
					continue
				}
				seen[rf.ID] = true
				files = append(files, rf)
			}
		}
		cancelCollect()

		if namer == nil {
			return renamePlanMsg{rows: rawProposals(files), geminiErr: fmt.Errorf("no gemini token configured (press s)")}
		}
		names := make([]string, len(files))
		for i, rf := range files {
			names[i] = rf.RelPath
		}
		namingCtx, cancelNaming := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancelNaming()
		parsed, err := namer.Normalise(namingCtx, names)
		if err != nil {
			return renamePlanMsg{rows: rawProposals(files), geminiErr: err}
		}

		rows := make([]proposal, len(files))
		for i, rf := range files {
			proposed := naming.BuildPath(parsed[i])
			rows[i] = proposal{file: rf, proposed: proposed, ticked: proposed != "Other/"+rf.RelPath}
		}
		return renamePlanMsg{rows: rows}
	}
}

func rawProposals(files []remoteFile) []proposal {
	rows := make([]proposal, len(files))
	for i, rf := range files {
		rows[i] = proposal{file: rf, proposed: "Other/" + rf.RelPath, ticked: false}
	}
	return rows
}

func (m Model) handleRenamePlan(msg renamePlanMsg) (tea.Model, tea.Cmd) {
	if len(msg.rows) == 0 {
		m.note = "nothing to download"
		return m, nil
	}
	if msg.geminiErr != nil {
		// nothing normalised — skip the dialog, keep original names
		items := make([]download.Item, len(msg.rows))
		for i, r := range msg.rows {
			items[i] = download.Item{FileID: r.file.ID, RelPath: r.file.RelPath, Size: r.file.Size}
		}
		m.enqueue(items)
		m.note = fmt.Sprintf("raw names (gemini: %v)", msg.geminiErr)
		return m, nil
	}
	m.confirm = newConfirmModel(msg.rows, m.width, m.height)
	m.screen = screenConfirm
	m.note = ""
	return m, nil
}

func (m *Model) enqueue(items []download.Item) {
	if m.dl == nil || len(items) == 0 {
		return
	}
	for _, it := range items {
		m.glyphs[it.FileID] = glyphQueued
	}
	m.queued += len(items)
	m.dl.Enqueue(items)
	m.note = fmt.Sprintf("queued %d item(s)", len(items))
	m.browser.glyphs = m.glyphs
}

const (
	glyphQueued      = "·"
	glyphDownloading = "↓"
	glyphDone        = "✓"
	glyphFailed      = "✗"
)

func (m Model) applyDownloadEvent(ev download.Event) Model {
	switch ev.Kind {
	case download.Started:
		m.glyphs[ev.FileID] = glyphDownloading
		m.activeDl = &ev
	case download.Progress:
		m.activeDl = &ev
	case download.Completed:
		m.glyphs[ev.FileID] = glyphDone
		m.activeDl = nil
		m.queued--
	case download.Failed:
		m.glyphs[ev.FileID] = glyphFailed
		m.activeDl = nil
		m.queued--
		if ev.Err != nil {
			m.note = fmt.Sprintf("failed: %s (%v)", ev.RelPath, ev.Err)
		}
	}
	if m.queued < 0 {
		m.queued = 0
	}
	m.browser.glyphs = m.glyphs
	return m
}

func (m Model) statusExtra() string {
	if m.deleteFor != nil {
		return fmt.Sprintf("Delete %q from put.io? (y/N)", m.deleteFor.Name)
	}
	var parts []string
	if m.activeDl != nil {
		ev := m.activeDl
		pct := 0
		if ev.Total > 0 {
			pct = int(ev.Done * 100 / ev.Total)
		}
		seg := fmt.Sprintf("↓ %s %d%%", ev.RelPath, pct)
		if ev.Speed > 0 {
			seg += fmt.Sprintf(" %.1f MB/s", ev.Speed/1e6)
		}
		if q := m.queued - 1; q > 0 {
			seg += fmt.Sprintf(" (+%d queued)", q)
		}
		parts = append(parts, seg)
	}
	if m.note != "" {
		parts = append(parts, m.note)
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " · "
		}
		out += p
	}
	return out
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return ""
	}
	var view string
	switch m.screen {
	case screenConfirm:
		view = m.confirm.view()
	case screenSettings:
		view = m.settings.view()
	default:
		m.browser.statusExtra = m.statusExtra()
		view = m.browser.view()
	}
	if m.err != nil {
		banner := errorBannerStyle.Width(m.width).Render("error: " + m.err.Error())
		view += "\n" + banner
	}
	return view
}

type errMsg struct{ err error }

type keyMap struct {
	Up, Down, Enter, Back key.Binding
	Quit                  key.Binding
}

var keys = keyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k")),
	Down:  key.NewBinding(key.WithKeys("down", "j")),
	Enter: key.NewBinding(key.WithKeys("enter", "l", "right")),
	Back:  key.NewBinding(key.WithKeys("backspace", "h", "left")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
}
