package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const appName = "teleput"

type Config struct {
	ClientID    string `json:"client_id,omitempty"`
	OAuthToken  string `json:"oauth_token"`
	GeminiToken string `json:"gemini_token,omitempty"`
	DownloadDir string `json:"download_dir,omitempty"`
}

func (c *Config) EffectiveDownloadDir() string {
	if c.DownloadDir != "" {
		return c.DownloadDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "Downloads"
	}
	return filepath.Join(home, "Downloads")
}

// EffectiveClientID is the put.io OAuth app ID used to sign in: the config
// value, or PUTIO_CLIENT_ID if that's empty.
func (c *Config) EffectiveClientID() string {
	if c.ClientID != "" {
		return c.ClientID
	}
	return os.Getenv("PUTIO_CLIENT_ID")
}

func (c *Config) EffectiveGeminiToken() string {
	if c.GeminiToken != "" {
		return c.GeminiToken
	}
	return os.Getenv("GEMINI_TOKEN")
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", appName)
	return dir, os.MkdirAll(dir, 0700)
}

func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0600)
}
