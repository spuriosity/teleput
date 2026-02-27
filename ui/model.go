package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	putio "github.com/putdotio/go-putio"
	"golang.org/x/oauth2"
)

type view int

const (
	viewBrowser view = iota
	viewDownload
	viewConfirmDelete
	viewDelete
)

type Model struct {
	client   *putio.Client
	token    string
	width    int
	height   int
	view     view
	browser  browserModel
	download downloadModel
	confirm  confirmModel
	delete   deleteModel
	err      error
	quitting bool
}

func NewModel(token string) Model {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	client := putio.NewClient(httpClient)

	return Model{
		client:  client,
		token:   token,
		view:    viewBrowser,
		browser: newBrowserModel(client),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.browser.loadDir(0), m.browser.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browser.width = msg.Width
		m.browser.height = msg.Height
		m.download.width = msg.Width
		m.download.height = msg.Height
		m.confirm.width = msg.Width
		m.confirm.height = msg.Height
		m.delete.width = msg.Width
		m.delete.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			if m.view == viewConfirmDelete || m.view == viewDelete {
				break
			}
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.Escape):
			if m.view == viewDownload && m.download.done {
				m.view = viewBrowser
				return m, nil
			}
			if m.view == viewConfirmDelete {
				m.view = viewBrowser
				return m, nil
			}
			if m.view == viewDelete && m.delete.done {
				m.view = viewBrowser
				m.browser.selected = make(map[int64]bool)
				return m, tea.Batch(m.browser.loadDir(m.browser.parentID), m.browser.spinner.Tick)
			}
			if m.err != nil {
				m.err = nil
				m.view = viewBrowser
				return m, nil
			}
		}

	case confirmMsg:
		ids := m.browser.selectedIDs()
		m.delete = newDeleteModel(m.client, ids)
		m.delete.width = m.width
		m.delete.height = m.height
		m.view = viewDelete
		return m, tea.Batch(m.delete.start(), m.delete.spinner.Tick)

	case cancelMsg:
		m.view = viewBrowser
		return m, nil

	case deleteCompleteMsg:
		var cmd tea.Cmd
		m.delete, cmd = m.delete.update(msg)
		return m, cmd

	case errMsg:
		if m.view == viewDelete {
			var cmd tea.Cmd
			m.delete, cmd = m.delete.update(msg)
			return m, cmd
		}
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		switch m.view {
		case viewBrowser:
			m.browser, cmd = m.browser.update(msg)
		case viewDownload:
			m.download, cmd = m.download.update(msg)
		case viewDelete:
			m.delete, cmd = m.delete.update(msg)
		}
		return m, cmd

	case progress.FrameMsg:
		if m.view == viewDownload {
			var cmd tea.Cmd
			m.download, cmd = m.download.update(msg)
			return m, cmd
		}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.view {
	case viewBrowser:
		m.browser, cmd = m.browser.update(msg)
		if m.browser.downloading {
			m.browser.downloading = false
			m.view = viewDownload
			m.download = newDownloadModel(m.client, m.token, m.browser.selectedIDs(), m.browser.downloadDir)
			m.download.width = m.width
			m.download.height = m.height
			return m, tea.Batch(m.download.start(), m.download.spinner.Tick)
		}
		if m.browser.deleting {
			m.browser.deleting = false
			ids := m.browser.selectedIDs()
			m.confirm = newConfirmModel(len(ids))
			m.confirm.width = m.width
			m.confirm.height = m.height
			m.view = viewConfirmDelete
			return m, nil
		}
	case viewDownload:
		m.download, cmd = m.download.update(msg)
	case viewConfirmDelete:
		m.confirm, cmd = m.confirm.update(msg)
	case viewDelete:
		m.delete, cmd = m.delete.update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return ""
	}

	switch m.view {
	case viewDownload:
		return m.download.view()
	case viewConfirmDelete:
		return m.confirm.view()
	case viewDelete:
		return m.delete.view()
	default:
		return m.browser.view()
	}
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type keyMap struct {
	Up, Down, Enter, Back key.Binding
	Space, SelectAll      key.Binding
	Download, SetDir      key.Binding
	Delete                key.Binding
	Top, Bottom           key.Binding
	Quit, Escape          key.Binding
}

var keys = keyMap{
	Up:        key.NewBinding(key.WithKeys("up", "k")),
	Down:      key.NewBinding(key.WithKeys("down", "j")),
	Enter:     key.NewBinding(key.WithKeys("enter", "l", "right")),
	Back:      key.NewBinding(key.WithKeys("backspace", "h", "left")),
	Space:     key.NewBinding(key.WithKeys(" ")),
	SelectAll: key.NewBinding(key.WithKeys("a")),
	Download:  key.NewBinding(key.WithKeys("d")),
	SetDir:    key.NewBinding(key.WithKeys("D")),
	Delete:    key.NewBinding(key.WithKeys("x")),
	Top:       key.NewBinding(key.WithKeys("g")),
	Bottom:    key.NewBinding(key.WithKeys("G")),
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Escape:    key.NewBinding(key.WithKeys("esc")),
}
