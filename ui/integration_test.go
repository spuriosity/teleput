package ui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	putio "github.com/putdotio/go-putio"

	"github.com/jack/teleput/config"
	"github.com/jack/teleput/download"
	"github.com/jack/teleput/naming"
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

func modelWithDownloads(t *testing.T) Model {
	t.Helper()
	m := sizedModel(t)
	m.dl = download.New(func(context.Context, int64) (string, error) { return "", context.Canceled }, t.TempDir())
	next, _ := m.Update(filesLoadedMsg{files: []putio.File{{ID: 7, Name: "raw.mkv"}}, parentID: 0})
	return next.(Model)
}

func failedPlan() renamePlanMsg {
	files := []remoteFile{{ID: 7, RelPath: "raw.mkv", Size: 1}}
	return renamePlanMsg{files: files, rows: rawProposals(files), geminiErr: errors.New("gemini status 503: high demand")}
}

func TestShouldPromptInsteadOfQueueingWhenGeminiFails(t *testing.T) {
	m := modelWithDownloads(t)

	next, _ := m.Update(failedPlan())
	m = next.(Model)

	if m.namingFailed == nil {
		t.Fatal("expected naming-failed prompt")
	}
	if _, queued := m.glyphs[7]; queued {
		t.Fatal("expected nothing queued before the user chooses")
	}
	// the status bar wraps, so compare with whitespace collapsed
	view := strings.Join(strings.Fields(m.View()), " ")
	if !strings.Contains(view, "r retry · o original names") || !strings.Contains(view, "high demand") {
		t.Fatalf("expected prompt with choices and the gemini error in view, got %q", view)
	}
	// browse keys are blocked while the prompt is up
	next, cmd := m.Update(keyPress("q"))
	m = next.(Model)
	if m.quitting || cmd != nil || m.namingFailed == nil {
		t.Fatal("expected q to be ignored while prompting")
	}
}

func TestShouldQueueOriginalNamesWhenChosen(t *testing.T) {
	m := modelWithDownloads(t)
	next, _ := m.Update(failedPlan())
	m = next.(Model)

	next, _ = m.Update(keyPress("o"))
	m = next.(Model)

	if m.namingFailed != nil || m.glyphs[7] != glyphQueued {
		t.Fatal("expected prompt cleared and item queued with original name")
	}
}

func TestShouldCancelFailedNaming(t *testing.T) {
	m := modelWithDownloads(t)
	next, _ := m.Update(failedPlan())
	m = next.(Model)

	next, _ = m.Update(keyPress("n"))
	m = next.(Model)

	if m.namingFailed != nil || m.glyphs[7] != "" {
		t.Fatal("expected prompt cleared and nothing queued")
	}
}

func TestShouldRetryNamingAndOpenConfirm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inner := `[{"original":"raw.mkv","type":"movie","title":"Raw","year":2020}]`
		json.NewEncoder(w).Encode(map[string]any{"candidates": []map[string]any{{"content": map[string]any{"parts": []map[string]string{{"text": inner}}}}}})
	}))
	defer srv.Close()
	m := modelWithDownloads(t)
	m.namer = naming.NewClient("test-token")
	m.namer.BaseURL = srv.URL
	next, _ := m.Update(failedPlan())
	m = next.(Model)

	next, cmd := m.Update(keyPress("r"))
	m = next.(Model)
	if cmd == nil || m.namingFailed != nil {
		t.Fatal("expected retry command and prompt cleared")
	}
	next, _ = m.Update(cmd())
	m = next.(Model)

	if m.screen != screenConfirm {
		t.Fatal("expected rename confirmation after successful retry")
	}
	if got := m.confirm.rows[0].proposed; !strings.Contains(got, "Raw (2020)") {
		t.Fatalf("expected Gemini-proposed path, got %q", got)
	}
}

func TestShouldEnqueueRawWithoutGeminiToken(t *testing.T) {
	m := modelWithDownloads(t)

	next, _ := m.Update(planNames(nil, []remoteFile{{ID: 7, RelPath: "raw.mkv", Size: 1}})())
	m = next.(Model)

	if m.namingFailed != nil || m.glyphs[7] != glyphQueued {
		t.Fatal("expected raw names queued without a prompt when no token is configured")
	}
	if !strings.Contains(m.note, "raw names") {
		t.Fatalf("expected raw-names note, got %q", m.note)
	}
}
