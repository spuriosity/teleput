package ui

import (
	"strings"
	"testing"

	putio "github.com/putdotio/go-putio"
)

func sizedBrowser(files ...putio.File) browserModel {
	m := newBrowserModel(nil)
	m.width = 80
	m.height = 24
	m.files = files
	return m
}

func TestShouldShowFolderSizeInListing(t *testing.T) {
	d := dir(1, "Show")
	d.Size = 1610612736
	m := sizedBrowser(d)

	if !strings.Contains(m.view(), "1.5 GB") {
		t.Fatal("expected folder size in listing")
	}
}

func TestShouldToggleMarkWithSpace(t *testing.T) {
	m := sizedBrowser(file(1, "a.mkv", 10), file(2, "b.mkv", 20))

	m, _ = m.update(keyPress(" "))
	if _, on := m.marked[1]; !on {
		t.Fatal("expected file 1 marked after space")
	}
	if m.cursor != 1 {
		t.Fatalf("expected cursor to advance, got %d", m.cursor)
	}
	if !strings.Contains(m.view(), "1 marked") {
		t.Fatal("expected marked count in status bar")
	}

	m.cursor = 0
	m, _ = m.update(keyPress(" "))
	if _, on := m.marked[1]; on {
		t.Fatal("expected file 1 unmarked after second space")
	}
}

func TestShouldPlanDownloadForAllMarkedItems(t *testing.T) {
	m := sizedModel(t)
	m.lister = fakeLister{}
	next, _ := m.Update(filesLoadedMsg{files: []putio.File{file(1, "a.mkv", 10), file(2, "b.mkv", 20)}, parentID: 0})
	m = next.(Model)

	next, _ = m.Update(keyPress(" "))
	m = next.(Model)
	next, _ = m.Update(keyPress(" "))
	m = next.(Model)
	next, cmd := m.Update(keyPress("d"))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("expected plan command")
	}
	if len(m.browser.marked) != 0 {
		t.Fatal("expected marks cleared once planning starts")
	}

	plan, ok := cmd().(renamePlanMsg)
	if !ok {
		t.Fatalf("expected renamePlanMsg, got %#v", cmd())
	}
	if len(plan.rows) != 2 {
		t.Fatalf("expected 2 planned files, got %d", len(plan.rows))
	}
}

func TestShouldDedupeFileMarkedInsideMarkedFolder(t *testing.T) {
	inner := file(1, "movie.mkv", 100)
	folder := dir(10, "My Movie")
	m := sizedModel(t)
	m.lister = fakeLister{10: {inner}}
	next, _ := m.Update(filesLoadedMsg{files: []putio.File{folder, inner}, parentID: 0})
	m = next.(Model)

	next, _ = m.Update(keyPress(" ")) // marks folder
	m = next.(Model)
	next, _ = m.Update(keyPress(" ")) // marks the file it contains
	m = next.(Model)
	_, cmd := m.Update(keyPress("d"))
	if cmd == nil {
		t.Fatal("expected plan command")
	}

	plan := cmd().(renamePlanMsg)
	if len(plan.rows) != 1 {
		t.Fatalf("expected duplicate file planned once, got %d rows", len(plan.rows))
	}
}

func TestShouldDownloadCursorItemWhenNothingMarked(t *testing.T) {
	m := sizedModel(t)
	m.lister = fakeLister{}
	next, _ := m.Update(filesLoadedMsg{files: []putio.File{file(1, "a.mkv", 10)}, parentID: 0})
	m = next.(Model)

	_, cmd := m.Update(keyPress("d"))
	if cmd == nil {
		t.Fatal("expected plan command")
	}
	plan := cmd().(renamePlanMsg)
	if len(plan.rows) != 1 || plan.rows[0].file.ID != 1 {
		t.Fatalf("expected cursor item planned, got %+v", plan.rows)
	}
}

func TestShouldComputeVisibleRange(t *testing.T) {
	tests := []struct {
		name                string
		cursor, total, rows int
		wantStart, wantEnd  int
	}{
		{"fits entirely", 0, 5, 20, 0, 5},
		{"top of long list", 0, 100, 20, 0, 20},
		{"cursor mid-list is inside window", 50, 100, 20, 31, 51},
		{"cursor at end", 99, 100, 20, 80, 100},
		{"single row viewport", 7, 100, 1, 7, 8},
		{"empty list", 0, 0, 20, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleRange(tt.cursor, tt.total, tt.rows)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("visibleRange(%d,%d,%d) = (%d,%d), want (%d,%d)",
					tt.cursor, tt.total, tt.rows, start, end, tt.wantStart, tt.wantEnd)
			}
			if tt.total > 0 && (tt.cursor < start || tt.cursor >= end) {
				t.Fatalf("cursor %d outside visible window [%d,%d)", tt.cursor, start, end)
			}
		})
	}
}
