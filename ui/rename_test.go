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

	"github.com/jack/teleput/naming"
)

type fakeRenamer struct {
	renamed map[int64]string
	failFor int64
}

func (f *fakeRenamer) Rename(_ context.Context, id int64, newname string) error {
	if id == f.failFor {
		return errors.New("put.io said no")
	}
	f.renamed[id] = newname
	return nil
}

const seasonDir = "IWTV S01 (1080p x265 EAC3 5.1 Silence)"

// renameModel shows one season folder holding an episode and a sample, with
// a fake Gemini that classifies all three.
func renameModel(t *testing.T) (Model, *fakeRenamer) {
	t.Helper()
	inner, _ := json.Marshal([]naming.Parsed{
		{Original: seasonDir, Type: "tv", Title: "Interview with the Vampire", Year: 2022, Season: 1},
		{Original: seasonDir + "/iwtv.s01e05.mkv", Type: "tv", Title: "Interview with the Vampire", Season: 1, Episode: 5, EpisodeTitle: "A Vile Hunger"},
		{Original: seasonDir + "/sample.mkv", Type: "other"},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"candidates": []map[string]any{{"content": map[string]any{"parts": []map[string]string{{"text": string(inner)}}}}}})
	}))
	t.Cleanup(srv.Close)

	m := sizedModel(t)
	m.namer = naming.NewClient("test-token")
	m.namer.BaseURL = srv.URL
	m.lister = fakeLister{10: {file(1, "iwtv.s01e05.mkv", 100), file(2, "sample.mkv", 5)}}
	fr := &fakeRenamer{renamed: map[int64]string{}}
	m.renamer = fr
	next, _ := m.Update(filesLoadedMsg{files: []putio.File{dir(10, seasonDir)}, parentID: 0})
	return next.(Model), fr
}

// run feeds a command's message back into the model.
func run(t *testing.T, m Model, cmd tea.Cmd) (Model, tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command")
	}
	next, out := m.Update(cmd())
	return next.(Model), out
}

func TestShouldRenameFolderAndContentsInPlaceAfterApproval(t *testing.T) {
	m, fr := renameModel(t)

	next, cmd := m.Update(keyPress("R"))
	m = next.(Model)
	m, _ = run(t, m, cmd)

	if m.screen != screenConfirm || !m.confirm.inPlace {
		t.Fatal("expected put.io rename confirmation")
	}
	if len(fr.renamed) != 0 {
		t.Fatal("expected nothing renamed before approval")
	}
	got := map[int64]string{}
	for _, r := range m.confirm.rows {
		got[r.file.ID] = r.proposed
	}
	want := map[int64]string{
		10: "Interview with the Vampire - Season 01",
		1:  "Interview with the Vampire - S01E05 - A Vile Hunger.mkv",
	}
	if len(got) != len(want) || got[10] != want[10] || got[1] != want[1] {
		t.Fatalf("expected folder and episode proposals only (sample left alone), got %v", got)
	}
	view := strings.Join(strings.Fields(m.View()), " ")
	if !strings.Contains(view, "confirm renames on put.io") || !strings.Contains(view, "→ Interview with the Vampire - Season 01") {
		t.Fatalf("expected old → new names in the confirm view, got %q", view)
	}

	next, cmd = m.Update(keyPress("enter"))
	m = next.(Model)
	m, cmd = run(t, m, cmd) // confirmRenameMsg → applyRenames
	m, _ = run(t, m, cmd)   // renameDoneMsg

	if len(fr.renamed) != 2 || fr.renamed[10] != want[10] || fr.renamed[1] != want[1] {
		t.Fatalf("expected both renames applied on put.io, got %v", fr.renamed)
	}
	if !strings.Contains(m.note, "renamed 2 item(s)") {
		t.Fatalf("expected summary note, got %q", m.note)
	}
}

func TestShouldOnlyRenameTickedRows(t *testing.T) {
	m, fr := renameModel(t)
	next, cmd := m.Update(keyPress("R"))
	m, _ = run(t, next.(Model), cmd)

	next, _ = m.Update(keyPress(" ")) // untick the folder (first row)
	m = next.(Model)
	next, cmd = m.Update(keyPress("enter"))
	m, cmd = run(t, next.(Model), cmd)
	run(t, m, cmd)

	if _, ok := fr.renamed[10]; ok || len(fr.renamed) != 1 {
		t.Fatalf("expected only the episode renamed, got %v", fr.renamed)
	}
}

func TestShouldCancelRenameWithoutTouchingPutio(t *testing.T) {
	m, fr := renameModel(t)
	next, cmd := m.Update(keyPress("R"))
	m, _ = run(t, next.(Model), cmd)

	next, cmd = m.Update(keyPress("esc"))
	m, _ = run(t, next.(Model), cmd)

	if m.screen != screenBrowse || len(fr.renamed) != 0 || m.note != "rename cancelled" {
		t.Fatalf("expected cancel with no renames, got screen=%v renamed=%v note=%q", m.screen, fr.renamed, m.note)
	}
}

func TestShouldSkipClashingAndReportFailedRenames(t *testing.T) {
	fr := &fakeRenamer{renamed: map[int64]string{}, failFor: 3}
	ops := []renameOp{
		{id: 1, relPath: "dir/a.mkv", newName: "Same.mkv"},
		{id: 2, relPath: "dir/b.mkv", newName: "Same.mkv"},
		{id: 3, relPath: "dir/c.mkv", newName: "C.mkv"},
		{id: 4, relPath: "dir/d.mkv", newName: "bad/name.mkv"},
		{id: 5, relPath: "other/a.mkv", newName: "Same.mkv"},
	}

	done := applyRenames(fr, ops, 0)().(renameDoneMsg)

	if done.renamed != 2 || done.skipped != 1 || len(done.failed) != 2 {
		t.Fatalf("expected 2 renamed, 1 clash skipped, 2 failed; got %+v", done)
	}
	if fr.renamed[1] != "Same.mkv" || fr.renamed[5] != "Same.mkv" {
		t.Fatalf("expected same name allowed in different folders, got %v", fr.renamed)
	}
}

func TestShouldOfferRetryButNotOriginalNamesWhenRenameNamingFails(t *testing.T) {
	m, fr := renameModel(t)
	items := []remoteFile{{ID: 10, RelPath: seasonDir, IsDir: true}}
	next, _ := m.Update(renamePlanMsg{files: items, inPlace: true, geminiErr: errors.New("gemini status 503: high demand")})
	m = next.(Model)

	view := strings.Join(strings.Fields(m.View()), " ")
	if m.namingFailed == nil || !strings.Contains(view, "r retry · esc cancel") || strings.Contains(view, "original names") {
		t.Fatalf("expected retry/cancel prompt without original-names choice, got %q", view)
	}
	next, cmd := m.Update(keyPress("o"))
	m = next.(Model)
	if cmd != nil || m.namingFailed == nil {
		t.Fatal("expected o to do nothing for renames")
	}

	next, cmd = m.Update(keyPress("r"))
	m, _ = run(t, next.(Model), cmd)
	if m.screen != screenConfirm || !m.confirm.inPlace || len(fr.renamed) != 0 {
		t.Fatal("expected retry to reach the rename confirmation")
	}
}

func TestShouldNeedGeminiTokenToRename(t *testing.T) {
	m, _ := renameModel(t)
	m.namer = nil

	next, cmd := m.Update(keyPress("R"))
	m = next.(Model)

	if cmd != nil || !strings.Contains(m.note, "gemini token") {
		t.Fatalf("expected a token hint and no work, got note=%q", m.note)
	}
}
