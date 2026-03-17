package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const appName = "teleput"

type Config struct {
	OAuthToken string          `json:"oauth_token"`
	Organizer  OrganizerConfig `json:"organizer,omitempty"`
}

type OrganizerConfig struct {
	Enabled    bool         `json:"enabled"`
	Paths      LibraryPaths `json:"paths"`
	DeleteJunk bool         `json:"delete_junk"`
}

type LibraryPaths struct {
	Movies     string `json:"movies,omitempty"`
	TV         string `json:"tv,omitempty"`
	Music      string `json:"music,omitempty"`
	Audiobooks string `json:"audiobooks,omitempty"`
	Books      string `json:"books,omitempty"`
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
