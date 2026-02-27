package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	putio "github.com/putdotio/go-putio"
	"golang.org/x/oauth2"
)

type Model struct {
	client   *putio.Client
	width    int
	height   int
	browser  browserModel
	err      error
	quitting bool
}

func NewModel(token string) Model {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	client := putio.NewClient(httpClient)

	return Model{
		client:  client,
		browser: newBrowserModel(client),
	}
}

func (m Model) Init() tea.Cmd {
	return m.browser.loadDir(0)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browser.width = msg.Width
		m.browser.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, keys.Quit) {
			m.quitting = true
			return m, tea.Quit
		}

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	m.browser, cmd = m.browser.update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return ""
	}
	return m.browser.view()
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
