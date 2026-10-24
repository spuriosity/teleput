package ui

import (
	"context"
	"testing"

	putio "github.com/putdotio/go-putio"
)

type fakeLister map[int64][]putio.File

func (f fakeLister) List(_ context.Context, parentID int64) ([]putio.File, putio.File, error) {
	return f[parentID], putio.File{}, nil
}

func dir(id int64, name string) putio.File {
	return putio.File{ID: id, Name: name, ContentType: "application/x-directory"}
}

func file(id int64, name string, size int64) putio.File {
	return putio.File{ID: id, Name: name, Size: size, ContentType: "video/x-matroska"}
}

func TestShouldFlattenNestedFolder(t *testing.T) {
	l := fakeLister{
		10: {file(1, "movie.mkv", 100), dir(20, "Extras")},
		20: {file(2, "making-of.mkv", 50)},
	}

	got, err := collectFiles(context.Background(), l, dir(10, "My Movie"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[int64]string{
		1: "My Movie/movie.mkv",
		2: "My Movie/Extras/making-of.mkv",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d files, want %d", len(got), len(want))
	}
	for _, rf := range got {
		if want[rf.ID] != rf.RelPath {
			t.Fatalf("file %d relpath = %q, want %q", rf.ID, rf.RelPath, want[rf.ID])
		}
	}
}

func TestShouldReturnSingleEntryForFile(t *testing.T) {
	got, err := collectFiles(context.Background(), fakeLister{}, file(7, "solo.epub", 42))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].RelPath != "solo.epub" || got[0].Size != 42 {
		t.Fatalf("unexpected result %+v", got)
	}
}

func TestShouldCollectTreeIncludingFolders(t *testing.T) {
	l := fakeLister{
		10: {file(1, "movie.mkv", 100), dir(20, "Extras")},
		20: {file(2, "making-of.mkv", 50)},
	}

	got, err := collectTree(context.Background(), l, dir(10, "My Movie"))
	if err != nil {
		t.Fatal(err)
	}
	want := []remoteFile{
		{ID: 10, RelPath: "My Movie", IsDir: true},
		{ID: 1, RelPath: "My Movie/movie.mkv", Size: 100},
		{ID: 20, RelPath: "My Movie/Extras", IsDir: true},
		{ID: 2, RelPath: "My Movie/Extras/making-of.mkv", Size: 50},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestShouldCollectTreeForSingleFile(t *testing.T) {
	got, err := collectTree(context.Background(), fakeLister{}, file(1, "a.mkv", 5))
	if err != nil || len(got) != 1 || got[0].IsDir || got[0].RelPath != "a.mkv" {
		t.Fatalf("expected the file alone, got %+v err=%v", got, err)
	}
}
