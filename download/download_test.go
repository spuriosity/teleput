package download

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixedResolver(url string) URLResolver {
	return func(ctx context.Context, fileID int64) (string, error) { return url, nil }
}

func awaitTerminal(t *testing.T, m *Manager, want EventKind) Event {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case ev := <-m.Events():
			if ev.Kind == Completed || ev.Kind == Failed {
				if ev.Kind != want {
					t.Fatalf("terminal event = %v (err=%v), want %v", ev.Kind, ev.Err, want)
				}
				return ev
			}
		case <-deadline:
			t.Fatal("timed out waiting for terminal event")
		}
	}
}

func TestShouldDownloadFileToRelPath(t *testing.T) {
	data := bytes.Repeat([]byte("teleput!"), 4096)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "f", time.Time{}, bytes.NewReader(data))
	}))
	defer srv.Close()

	dest := t.TempDir()
	m := New(fixedResolver(srv.URL), dest)
	m.Enqueue([]Item{{FileID: 1, RelPath: "Movies/X (2001)/X (2001).mkv", Size: int64(len(data))}})

	awaitTerminal(t, m, Completed)

	got, err := os.ReadFile(filepath.Join(dest, "Movies/X (2001)/X (2001).mkv"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("downloaded content mismatch")
	}
	if _, err := os.Stat(filepath.Join(dest, "Movies/X (2001)/X (2001).mkv.part")); !os.IsNotExist(err) {
		t.Fatal("expected .part to be renamed away")
	}
}

func TestShouldResumeFromPartFile(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789abcdef"), 4096)
	var sawRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRange = r.Header.Get("Range")
		http.ServeContent(w, r, "f", time.Time{}, bytes.NewReader(data))
	}))
	defer srv.Close()

	dest := t.TempDir()
	half := len(data) / 2
	partPath := filepath.Join(dest, "big.bin.part")
	if err := os.WriteFile(partPath, data[:half], 0o644); err != nil {
		t.Fatal(err)
	}

	m := New(fixedResolver(srv.URL), dest)
	m.Enqueue([]Item{{FileID: 2, RelPath: "big.bin", Size: int64(len(data))}})
	awaitTerminal(t, m, Completed)

	if want := fmt.Sprintf("bytes=%d-", half); sawRange != want {
		t.Fatalf("expected Range %q, server saw %q", want, sawRange)
	}
	got, _ := os.ReadFile(filepath.Join(dest, "big.bin"))
	if !bytes.Equal(got, data) {
		t.Fatal("resumed content mismatch")
	}
}

func TestShouldRestartWhenServerIgnoresRange(t *testing.T) {
	data := bytes.Repeat([]byte("fresh-data!"), 2048)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // ignores Range entirely
		w.Write(data)
	}))
	defer srv.Close()

	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(dest, "f.bin.part"), []byte("stale-garbage"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New(fixedResolver(srv.URL), dest)
	m.Enqueue([]Item{{FileID: 3, RelPath: "f.bin", Size: int64(len(data))}})
	awaitTerminal(t, m, Completed)

	got, _ := os.ReadFile(filepath.Join(dest, "f.bin"))
	if !bytes.Equal(got, data) {
		t.Fatal("expected stale .part to be truncated and replaced")
	}
}

func TestShouldContinueQueueAfterFailure(t *testing.T) {
	data := []byte("second-file-ok")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "f", time.Time{}, bytes.NewReader(data))
	}))
	defer srv.Close()

	resolver := func(ctx context.Context, fileID int64) (string, error) {
		if fileID == 1 {
			return "", errors.New("resolver boom")
		}
		return srv.URL, nil
	}

	m := New(resolver, t.TempDir())
	m.Enqueue([]Item{
		{FileID: 1, RelPath: "fails.bin", Size: 10},
		{FileID: 2, RelPath: "works.bin", Size: int64(len(data))},
	})

	first := awaitTerminal(t, m, Failed)
	if first.FileID != 1 {
		t.Fatalf("expected file 1 to fail, got %d", first.FileID)
	}
	second := awaitTerminal(t, m, Completed)
	if second.FileID != 2 {
		t.Fatalf("expected file 2 to complete, got %d", second.FileID)
	}
}

func TestShouldCancelInFlightDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		for i := 0; i < 10_000; i++ {
			if _, err := w.Write(bytes.Repeat([]byte("x"), 1024)); err != nil {
				return
			}
			if fl != nil {
				fl.Flush()
			}
			time.Sleep(2 * time.Millisecond)
		}
	}))
	defer srv.Close()

	m := New(fixedResolver(srv.URL), t.TempDir())
	m.Enqueue([]Item{{FileID: 9, RelPath: "slow.bin", Size: 10_000 * 1024}})

	// wait for Started, then cancel
	deadline := time.After(5 * time.Second)
	for started := false; !started; {
		select {
		case ev := <-m.Events():
			if ev.Kind == Started {
				started = true
			}
		case <-deadline:
			t.Fatal("never saw Started")
		}
	}
	m.Cancel(9)

	ev := awaitTerminal(t, m, Failed)
	if !errors.Is(ev.Err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", ev.Err)
	}
}
