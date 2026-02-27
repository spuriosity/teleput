# teleput

TUI file browser for [put.io](https://put.io). Browse your files, navigate folders, see sizes — all from the terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) (Catppuccin Mocha theme).

## Install

```bash
go install github.com/jack/teleput@latest
```

## Usage

```bash
teleput                          # uses saved token or starts OAuth flow
teleput --token=your_token       # pass token directly
PUTIO_TOKEN=xxx teleput          # or via env var
```

On first run with no token, it opens a browser for put.io OAuth approval and saves the token to `~/.config/teleput/config.json`.

## Keys

| Key | Action |
|-----|--------|
| `j`/`↓` | Move down |
| `k`/`↑` | Move up |
| `l`/`→`/`Enter` | Open folder |
| `h`/`←`/`Backspace` | Go back |
| `q` | Quit |
