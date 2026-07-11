package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	putio "github.com/putdotio/go-putio"

	"github.com/jack/teleput/config"
	"github.com/jack/teleput/download"
)

func TestShouldTrackDownloadGlyphsFromEvents(t *testing.T) {
	m := sizedModel(t)

	m = m.applyDownloadEvent(download.Event{Kind: download.Started, FileID: 1})
	if m.glyphs[1] != glyphDownloading {
		t.Fatalf("expected downloading glyph, got %q", m.glyphs[1])
	}
	m = m.applyDownloadEvent(download.Event{Kind: download.Completed, FileID: 1})
	if m.glyphs[1] != glyphDone {
		t.Fatalf("expected done glyph, got %q", m.glyphs[1])
	}
	m = m.applyDownloadEvent(download.Event{Kind: download.Failed, FileID: 2})
	if m.glyphs[2] != glyphFailed {
		t.Fatalf("expected failed glyph, got %q", m.glyphs[2])
	}
}

type fakeDeleter struct{ deleted []int64 }

func (f *fakeDeleter) Delete(_ context.Context, ids ...int64) error {
	f.deleted = append(f.deleted, ids...)
	return nil
}

func modelWithFile(t *testing.T) (Model, *fakeDeleter) {
	t.Helper()
	m := NewModel(nil, &config.Config{}, nil, nil)
	m.saveCfg = func(*config.Config) error { return nil }
	fd := &fakeDeleter{}
	m.deleter = fd
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	next, _ = m.Update(filesLoadedMsg{files: []putio.File{{ID: 42, Name: "victim.mkv"}}, parentID: 0})
	m = next.(Model)
	return m, fd
}

func TestShouldDeleteRemoteFileAfterConfirm(t *testing.T) {
	m, fd := modelWithFile(t)

	next, _ := m.Update(keyPress("D"))
	m = next.(Model)
	if m.deleteFor == nil || m.deleteFor.ID != 42 {
		t.Fatal("expected pending delete confirmation for file 42")
	}
	if !strings.Contains(m.View(), `Delete "victim.mkv"`) {
		t.Fatal("expected confirmation prompt in view")
	}

	next, cmd := m.Update(keyPress("y"))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("expected delete command")
	}
	if msg, ok := cmd().(deleteDoneMsg); !ok || msg.err != nil {
		t.Fatalf("expected successful deleteDoneMsg, got %#v", msg)
	}
	if len(fd.deleted) != 1 || fd.deleted[0] != 42 {
		t.Fatalf("expected file 42 deleted, got %v", fd.deleted)
	}
}

func TestShouldNotDeleteWhenDeclined(t *testing.T) {
	m, fd := modelWithFile(t)

	next, _ := m.Update(keyPress("D"))
	m = next.(Model)
	next, _ = m.Update(keyPress("n"))
	m = next.(Model)

	if m.deleteFor != nil {
		t.Fatal("expected pending delete cleared")
	}
	if len(fd.deleted) != 0 {
		t.Fatalf("expected no deletion, got %v", fd.deleted)
	}
}

func TestShouldSaveSettingsFieldEdits(t *testing.T) {
	m := sizedModel(t)
	saved := 0
	m.saveCfg = func(*config.Config) error { saved++; return nil }

	next, _ := m.Update(keyPress("s"))
	m = next.(Model)
	if m.screen != screenSettings {
		t.Fatal("expected settings screen")
	}

	// move to gemini token field, edit it
	next, _ = m.Update(keyPress("j"))
	m = next.(Model)
	next, _ = m.Update(keyPress("enter"))
	m = next.(Model)
	m.settings.input.SetValue("new-gemini-token")
	next, cmd := m.Update(keyPress("enter"))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("expected save command")
	}
	next, _ = m.Update(cmd())
	m = next.(Model)

	if m.cfg.GeminiToken != "new-gemini-token" {
		t.Fatalf("expected config updated, got %q", m.cfg.GeminiToken)
	}
	if saved != 1 {
		t.Fatalf("expected exactly one save, got %d", saved)
	}
	if !strings.Contains(m.settings.view(), "••••") {
		t.Fatal("expected token masked in settings view")
	}
}

func TestShouldEnqueueRawWhenGeminiUnavailable(t *testing.T) {
	m := sizedModel(t)
	dl := download.New(func(context.Context, int64) (string, error) { return "", context.Canceled }, t.TempDir())
	m.dl = dl

	rows := rawProposals([]remoteFile{{ID: 7, RelPath: "raw.mkv", Size: 1}})
	next, _ := m.Update(renamePlanMsg{rows: rows, geminiErr: context.DeadlineExceeded})
	m = next.(Model)

	if m.screen != screenBrowse {
		t.Fatal("expected to stay on browse (no dialog) on full gemini failure")
	}
	if m.glyphs[7] != glyphQueued {
		t.Fatal("expected item queued with raw name")
	}
	if !strings.Contains(m.note, "raw names") {
		t.Fatalf("expected raw-names note, got %q", m.note)
	}
}
