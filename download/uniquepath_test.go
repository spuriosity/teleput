package download

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldReturnPathUnchangedWhenNoCollision(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.mkv")
	if got := uniquePath(p); got != p {
		t.Fatalf("expected %q, got %q", p, got)
	}
}

func TestShouldSuffixNameWhenFileExists(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.mkv")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := uniquePath(p)
	if got != filepath.Join(dir, "a (1).mkv") {
		t.Fatalf("expected suffixed name, got %q", got)
	}

	if err := os.WriteFile(got, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := uniquePath(p); got != filepath.Join(dir, "a (2).mkv") {
		t.Fatalf("expected second suffix, got %q", got)
	}
}
