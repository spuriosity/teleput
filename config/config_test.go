package config

import (
	"strings"
	"testing"
)

func TestShouldDefaultDownloadDirToHomeDownloads(t *testing.T) {
	c := &Config{}
	if got := c.EffectiveDownloadDir(); !strings.HasSuffix(got, "/Downloads") {
		t.Fatalf("expected default ending in /Downloads, got %q", got)
	}
}

func TestShouldPreferConfiguredDownloadDir(t *testing.T) {
	c := &Config{DownloadDir: "/tmp/media"}
	if got := c.EffectiveDownloadDir(); got != "/tmp/media" {
		t.Fatalf("expected configured dir, got %q", got)
	}
}

func TestShouldFallBackToClientIDEnvVar(t *testing.T) {
	t.Setenv("PUTIO_CLIENT_ID", "1234")
	c := &Config{}
	if got := c.EffectiveClientID(); got != "1234" {
		t.Fatalf("expected env fallback, got %q", got)
	}
	c.ClientID = "5678"
	if got := c.EffectiveClientID(); got != "5678" {
		t.Fatalf("expected config to win over env, got %q", got)
	}
}

func TestShouldFallBackToGeminiEnvVar(t *testing.T) {
	t.Setenv("GEMINI_TOKEN", "env-token")
	c := &Config{}
	if got := c.EffectiveGeminiToken(); got != "env-token" {
		t.Fatalf("expected env fallback, got %q", got)
	}
	c.GeminiToken = "cfg-token"
	if got := c.EffectiveGeminiToken(); got != "cfg-token" {
		t.Fatalf("expected config to win over env, got %q", got)
	}
}
