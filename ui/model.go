package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	putio "github.com/putdotio/go-putio"
	"golang.org/x/oauth2"
)

type view int

const (
	viewBrowser view = iota
	viewDownload
	viewHelp
	viewConfirmDelete
	viewDelete
	viewRename
	viewTransfers
	viewAddMagnet
	viewConfirmCancelTransfer
	viewCancelTransfer
)

type Model struct {
	client         *putio.Client
	token          string
	width          int
	height         int
	view           view
	browser        browserModel
	download       downloadModel
	confirm        confirmModel
	delete         deleteModel
	rename         renameModel
	transfers      transfersModel
	magnet         magnetModel
	transferCancel transferCancelModel
	diskUsed       int64
	diskTotal      int64
	err            error
	quitting       bool
	startView      view
}

func NewModel(token string) Model {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	client := putio.NewClient(httpClient)

	return Model{
		client:    client,
		token:     token,
		view:      viewBrowser,
		startView: viewBrowser,
		browser:   newBrowserModel(client),
		transfers: newTransfersModel(client),
	}
}

func NewTransfersModel(token string) Model {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	client := putio.NewClient(httpClient)

	return Model{
		client:    client,
		token:     token,
		view:      viewTransfers,
		startView: viewTransfers,
		browser:   newBrowserModel(client),
		transfers: newTransfersModel(client),
	}
}

func (m Model) Init() tea.Cmd {
	fetchAccount := func() tea.Msg {
		info, err := m.client.Account.Info(context.Background())
		if err != nil {
			return nil
		}
		return accountInfoMsg{diskUsed: info.Disk.Used, diskTotal: info.Disk.Size}
	}
	if m.startView == viewTransfers {
		m.transfers.loading = true
		return tea.Batch(m.transfers.loadTransfers(), m.transfers.spinner.Tick, fetchAccount)
	}
	return tea.Batch(m.browser.loadDir(0), m.browser.spinner.Tick, fetchAccount)
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
		m.rename.width = msg.Width
		m.rename.height = msg.Height
		m.transfers.width = msg.Width
		m.transfers.height = msg.Height
		m.magnet.width = msg.Width
		m.magnet.height = msg.Height
		m.transferCancel.width = msg.Width
		m.transferCancel.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			if m.view == viewConfirmDelete || m.view == viewDelete || m.view == viewRename ||
				m.view == viewAddMagnet || m.view == viewConfirmCancelTransfer || m.view == viewCancelTransfer {
				break
			}
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.Help):
			if m.view == viewConfirmDelete || m.view == viewDelete || m.view == viewRename ||
				m.view == viewAddMagnet || m.view == viewConfirmCancelTransfer || m.view == viewCancelTransfer {
				break
			}
			if m.view == viewHelp {
				m.view = viewBrowser
			} else {
				m.view = viewHelp
			}
			return m, nil
		case key.Matches(msg, keys.Tab):
			if m.view == viewBrowser {
				m.browser.fromTransfers = false
				m.transfers.loading = true
				m.view = viewTransfers
				return m, tea.Batch(m.transfers.loadTransfers(), m.transfers.spinner.Tick)
			}
			if m.view == viewTransfers {
				m.view = viewBrowser
				return m, nil
			}
		case key.Matches(msg, keys.Enter):
			if m.view == viewDownload && m.download.done {
				return m, m.download.browseFiles()
			}
		case key.Matches(msg, keys.Escape):
			if m.view == viewHelp {
				m.view = viewBrowser
				return m, nil
			}
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
			if m.view == viewRename && m.rename.done {
				m.view = viewBrowser
				return m, tea.Batch(m.browser.loadDir(m.browser.parentID), m.browser.spinner.Tick)
			}
			if m.view == viewTransfers {
				m.view = viewBrowser
				return m, nil
			}
			if m.view == viewAddMagnet && m.magnet.done {
				m.view = viewTransfers
				m.transfers.loading = true
				return m, tea.Batch(m.transfers.loadTransfers(), m.transfers.spinner.Tick)
			}
			if m.view == viewConfirmCancelTransfer {
				m.view = viewTransfers
				return m, nil
			}
			if m.view == viewCancelTransfer && m.transferCancel.done {
				m.view = viewTransfers
				m.transfers.selected = make(map[int64]bool)
				m.transfers.loading = true
				return m, tea.Batch(m.transfers.loadTransfers(), m.transfers.spinner.Tick)
			}
			if m.err != nil {
				return m, m.dismissError()
			}
		case key.Matches(msg, keys.Back):
			if m.err != nil {
				return m, m.dismissError()
			}
		}

	case confirmMsg:
		if m.view == viewConfirmCancelTransfer {
			ids := m.transfers.selectedIDs()
			m.transferCancel = newTransferCancelModel(m.client, ids)
			m.transferCancel.width = m.width
			m.transferCancel.height = m.height
			m.view = viewCancelTransfer
			return m, tea.Batch(m.transferCancel.start(), m.transferCancel.spinner.Tick)
		}
		ids := m.browser.selectedIDs()
		m.delete = newDeleteModel(m.client, ids)
		m.delete.width = m.width
		m.delete.height = m.height
		m.view = viewDelete
		return m, tea.Batch(m.delete.start(), m.delete.spinner.Tick)

	case cancelMsg:
		if m.view == viewAddMagnet {
			m.view = viewTransfers
			return m, nil
		}
		if m.view == viewConfirmCancelTransfer {
			m.view = viewTransfers
			return m, nil
		}
		m.view = viewBrowser
		return m, nil

	case deleteCompleteMsg:
		var cmd tea.Cmd
		m.delete, cmd = m.delete.update(msg)
		return m, cmd

	case renameCompleteMsg:
		var cmd tea.Cmd
		m.rename, cmd = m.rename.update(msg)
		return m, cmd

	case magnetCompleteMsg:
		var cmd tea.Cmd
		m.magnet, cmd = m.magnet.update(msg)
		return m, cmd

	case transferCancelCompleteMsg:
		var cmd tea.Cmd
		m.transferCancel, cmd = m.transferCancel.update(msg)
		return m, cmd

	case downloadBrowseMsg:
		m.browser.parents = nil
		m.browser.parentNames = nil
		m.browser.cursorHistory = make(map[int64]int)
		m.browser.selected = make(map[int64]bool)
		m.browser.loading = true
		m.view = viewBrowser
		return m, tea.Batch(m.browser.loadDir(msg.parentID), m.browser.spinner.Tick)

	case accountInfoMsg:
		m.diskUsed = msg.diskUsed
		m.diskTotal = msg.diskTotal
		return m, nil

	case transferRetryCompleteMsg:
		m.transfers.loading = true
		return m, tea.Batch(m.transfers.loadTransfers(), m.transfers.spinner.Tick)

	case transferCleanCompleteMsg:
		m.transfers.loading = true
		return m, tea.Batch(m.transfers.loadTransfers(), m.transfers.spinner.Tick)

	case transfersLoadedMsg:
		var cmd tea.Cmd
		m.transfers, cmd = m.transfers.update(msg)
		return m, cmd

	case transfersTickMsg:
		if m.view == viewTransfers {
			var cmd tea.Cmd
			m.transfers, cmd = m.transfers.update(msg)
			return m, cmd
		}
		return m, nil

	case errMsg:
		if m.view == viewRename {
			var cmd tea.Cmd
			m.rename, cmd = m.rename.update(msg)
			return m, cmd
		}
		if m.view == viewDelete {
			var cmd tea.Cmd
			m.delete, cmd = m.delete.update(msg)
			return m, cmd
		}
		if m.view == viewAddMagnet {
			var cmd tea.Cmd
			m.magnet, cmd = m.magnet.update(msg)
			return m, cmd
		}
		if m.view == viewCancelTransfer {
			var cmd tea.Cmd
			m.transferCancel, cmd = m.transferCancel.update(msg)
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
		case viewRename:
			m.rename, cmd = m.rename.update(msg)
		case viewTransfers:
			m.transfers, cmd = m.transfers.update(msg)
		case viewAddMagnet:
			m.magnet, cmd = m.magnet.update(msg)
		case viewCancelTransfer:
			m.transferCancel, cmd = m.transferCancel.update(msg)
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
		if m.browser.returningToTransfers {
			m.browser.returningToTransfers = false
			m.browser.fromTransfers = false
			m.transfers.loading = true
			m.view = viewTransfers
			return m, tea.Batch(m.transfers.loadTransfers(), m.transfers.spinner.Tick)
		}
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
		if m.browser.renaming {
			m.browser.renaming = false
			f := m.browser.files[m.browser.cursor]
			m.rename = newRenameModel(m.client, f.ID, f.Name)
			m.rename.width = m.width
			m.rename.height = m.height
			m.view = viewRename
			return m, m.rename.input.Focus()
		}
	case viewDownload:
		m.download, cmd = m.download.update(msg)
	case viewConfirmDelete:
		m.confirm, cmd = m.confirm.update(msg)
	case viewDelete:
		m.delete, cmd = m.delete.update(msg)
	case viewRename:
		m.rename, cmd = m.rename.update(msg)
	case viewTransfers:
		m.transfers, cmd = m.transfers.update(msg)
		if m.transfers.cancelling {
			m.transfers.cancelling = false
			ids := m.transfers.selectedIDs()
			m.confirm = newConfirmCancelModel(len(ids))
			m.confirm.width = m.width
			m.confirm.height = m.height
			m.view = viewConfirmCancelTransfer
			return m, nil
		}
		if m.transfers.addingMagnet {
			m.transfers.addingMagnet = false
			m.magnet = newMagnetModel(m.client)
			m.magnet.width = m.width
			m.magnet.height = m.height
			m.view = viewAddMagnet
			return m, m.magnet.input.Focus()
		}
		if m.transfers.retrying {
			m.transfers.retrying = false
			id := m.transfers.transfers[m.transfers.cursor].ID
			return m, retryTransfer(m.client, id)
		}
		if m.transfers.cleaning {
			m.transfers.cleaning = false
			return m, cleanTransfers(m.client)
		}
		if m.transfers.browsingFiles {
			m.transfers.browsingFiles = false
			fileID := m.transfers.browseFileID
			m.browser.parents = nil
			m.browser.parentNames = nil
			m.browser.cursorHistory = make(map[int64]int)
			m.browser.selected = make(map[int64]bool)
			m.browser.fromTransfers = true
			m.browser.loading = true
			m.view = viewBrowser
			return m, tea.Batch(m.browser.loadDir(fileID), m.browser.spinner.Tick)
		}
	case viewAddMagnet:
		m.magnet, cmd = m.magnet.update(msg)
	case viewConfirmCancelTransfer:
		m.confirm, cmd = m.confirm.update(msg)
	case viewCancelTransfer:
		m.transferCancel, cmd = m.transferCancel.update(msg)
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

	if m.diskTotal > 0 {
		m.browser.diskInfo = fmt.Sprintf("%s / %s", humanSize(m.diskUsed), humanSize(m.diskTotal))
		m.transfers.diskInfo = m.browser.diskInfo
	}

	if m.err != nil {
		return m.errorView()
	}

	switch m.view {
	case viewHelp:
		return m.helpView()
	case viewDownload:
		return m.download.view()
	case viewConfirmDelete, viewConfirmCancelTransfer:
		return m.confirm.view()
	case viewDelete:
		return m.delete.view()
	case viewRename:
		return m.rename.view()
	case viewTransfers:
		return m.transfers.view()
	case viewAddMagnet:
		return m.magnet.view()
	case viewCancelTransfer:
		return m.transferCancel.view()
	default:
		return m.browser.view()
	}
}

func (m Model) helpView() string {
	keyStyle := lipgloss.NewStyle().Foreground(catMauve).Width(14).Align(lipgloss.Right)
	descStyle := lipgloss.NewStyle().Foreground(catText)
	sectionStyle := lipgloss.NewStyle().Foreground(catPeach).Bold(true)

	row := func(k, desc string) string {
		return keyStyle.Render(k) + "  " + descStyle.Render(desc)
	}

	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(catMauve).
		Bold(true).
		Render("put.io TUI — Keybindings")
	content.WriteString(title + "\n\n")

	content.WriteString(sectionStyle.Render("Navigation") + "\n")
	content.WriteString(row("↑ / k", "Move up") + "\n")
	content.WriteString(row("↓ / j", "Move down") + "\n")
	content.WriteString(row("→ / Enter / l", "Open folder") + "\n")
	content.WriteString(row("← / Bksp / h", "Go back") + "\n")
	content.WriteString(row("g", "Go to top") + "\n")
	content.WriteString(row("G", "Go to bottom") + "\n")

	content.WriteString("\n" + sectionStyle.Render("Selection") + "\n")
	content.WriteString(row("Space", "Toggle select") + "\n")
	content.WriteString(row("a", "Select / deselect all") + "\n")

	content.WriteString("\n" + sectionStyle.Render("Actions") + "\n")
	content.WriteString(row("d", "Download selected") + "\n")
	content.WriteString(row("x", "Delete / cancel selected") + "\n")
	content.WriteString(row("r", "Rename item") + "\n")
	content.WriteString(row("D", "Set download directory") + "\n")
	content.WriteString(row("Tab", "Toggle transfers view") + "\n")
	content.WriteString(row("?", "Toggle this help") + "\n")
	content.WriteString(row("q / Ctrl+c", "Quit") + "\n")
	content.WriteString(row("Esc", "Close overlay / go back") + "\n")

	content.WriteString("\n" + sectionStyle.Render("Transfers") + "\n")
	content.WriteString(row("→ / Enter / l", "Browse transfer files") + "\n")
	content.WriteString(row("m", "Add magnet / URL") + "\n")
	content.WriteString(row("R", "Retry failed transfer") + "\n")
	content.WriteString(row("C", "Clean completed") + "\n")

	content.WriteString("\n" + dimTextStyle.Render("Press ? or Esc to close"))

	panel := panelStyle.Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func (m Model) errorView() string {
	var content strings.Builder

	title := errorTextStyle.Render("Error")
	content.WriteString(title + "\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(catText).Render(m.err.Error()))
	content.WriteString("\n\n")
	content.WriteString(dimTextStyle.Render("← or Esc to dismiss, q to quit"))

	panel := errorPanelStyle.Render(content.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func (m *Model) dismissError() tea.Cmd {
	m.err = nil
	m.browser.loading = false
	m.transfers.loading = false
	if m.view == viewTransfers || m.view == viewAddMagnet || m.view == viewCancelTransfer {
		m.view = viewTransfers
	} else {
		m.view = viewBrowser
	}
	return nil
}

// Messages
type accountInfoMsg struct {
	diskUsed  int64
	diskTotal int64
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type transferRetryCompleteMsg struct{}
type transferCleanCompleteMsg struct{}

func retryTransfer(client *putio.Client, id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Transfers.Retry(context.Background(), id)
		if err != nil {
			var apiErr *putio.ErrorResponse
			if errors.As(err, &apiErr) && apiErr.Response.StatusCode == 404 {
				return errMsg{fmt.Errorf("transfer no longer exists on put.io — use C to clean up")}
			}
			return errMsg{err}
		}
		return transferRetryCompleteMsg{}
	}
}

func cleanTransfers(client *putio.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.Transfers.Clean(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return transferCleanCompleteMsg{}
	}
}

func newConfirmCancelModel(count int) confirmModel {
	msg := fmt.Sprintf("Cancel %d transfer", count)
	if count != 1 {
		msg += "s"
	}
	msg += "?"
	return confirmModel{
		title:   "Confirm Cancel",
		message: msg,
		count:   count,
	}
}

type keyMap struct {
	Up, Down, Enter, Back key.Binding
	Space, SelectAll      key.Binding
	Download, SetDir      key.Binding
	Delete, Rename        key.Binding
	Help, Quit, Escape    key.Binding
	Top, Bottom           key.Binding
	Tab                   key.Binding
	AddMagnet             key.Binding
	Retry                 key.Binding
	Clean                 key.Binding
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
	Rename:    key.NewBinding(key.WithKeys("r")),
	Help:      key.NewBinding(key.WithKeys("?")),
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Escape:    key.NewBinding(key.WithKeys("esc")),
	Top:       key.NewBinding(key.WithKeys("g")),
	Bottom:    key.NewBinding(key.WithKeys("G")),
	Tab:       key.NewBinding(key.WithKeys("tab")),
	AddMagnet: key.NewBinding(key.WithKeys("m")),
	Retry:     key.NewBinding(key.WithKeys("R")),
	Clean:     key.NewBinding(key.WithKeys("C")),
}
