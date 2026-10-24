# teleput

TUI sync client for [put.io](https://put.io). Browse your files, download them
with resume support, and let Gemini normalise release names into a
media-server layout (`Movies/Title (Year)/…`, `TV/Show/Season 01/…`,
`Books/Author/…`) — with every rename shown for approval first.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss) (Catppuccin Mocha theme).

## Install

```bash
go build -o teleput .        # then move it onto your PATH
```

## Usage

```bash
teleput                          # uses saved token or starts OAuth flow
teleput login                    # force re-authentication
teleput logout                   # remove the saved token
teleput --token=your_token       # pass token directly
PUTIO_TOKEN=xxx teleput          # or via env var
```

On first run with no token, it opens a browser for put.io OAuth approval and
saves the token to `~/.config/teleput/config.json`.

## Keys

| Key | Action |
|-----|--------|
| `j`/`↓`, `k`/`↑` | Move |
| `l`/`→`/`Enter` | Open folder |
| `h`/`←`/`Backspace` | Go back |
| `space` | Mark/unmark for multi-download (marks survive navigation) |
| `d` | Download marked items, or the item under the cursor (opens rename confirmation) |
| `x` | Cancel download under cursor |
| `D` | Delete from put.io (asks y/N) |
| `s` | Settings (tokens, download dir) |
| `q` | Quit |

In the rename confirmation: `space` keeps a file's original name, `e` edits
the proposed path, `enter` approves the batch, `esc` cancels.

If Gemini is overloaded, naming retries up to three times; if it still fails
you're asked to retry (`r`), use original names (`o`) or
cancel (`esc`).

Downloads never replace existing local files: folders merge, and a file whose
destination already exists is saved alongside it as `name (1).ext`.

## Config

`~/.config/teleput/config.json`:

```json
{
  "oauth_token": "…",
  "gemini_token": "…",
  "download_dir": "/path/to/media"
}
```

`gemini_token` (or `GEMINI_TOKEN` env) enables name normalisation — without
it, downloads keep their original names. `download_dir` defaults to
`~/Downloads`. Downloads stream to `<name>.part` and resume automatically if
interrupted.
