package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	putio "github.com/putdotio/go-putio"

	"github.com/jack/teleput/config"
)

func sizedModel(t *testing.T) Model {
	t.Helper()
	m := NewModel(nil, &config.Config{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(Model)
}

func TestShouldRenderErrorBanner(t *testing.T) {
	m := sizedModel(t)

	next, _ := m.Update(errMsg{errors.New("boom")})
	m = next.(Model)

	if !strings.Contains(m.View(), "boom") {
		t.Fatalf("expected error banner containing %q in view", "boom")
	}
}

func TestShouldClearErrorBannerOnNextLoad(t *testing.T) {
	m := sizedModel(t)

	next, _ := m.Update(errMsg{errors.New("boom")})
	m = next.(Model)
	next, _ = m.Update(filesLoadedMsg{files: []putio.File{}, parentID: 0})
	m = next.(Model)

	if strings.Contains(m.View(), "boom") {
		t.Fatal("expected error banner to clear after successful load")
	}
}
