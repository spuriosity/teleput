package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jack/teleput/auth"
	"github.com/jack/teleput/config"
	"github.com/jack/teleput/ui"
)

func main() {
	tokenFlag := flag.String("token", "", "put.io OAuth token (or set PUTIO_TOKEN env var)")
	flag.Parse()

	token := *tokenFlag
	if token == "" {
		token = os.Getenv("PUTIO_TOKEN")
	}

	if token == "" {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		token = cfg.OAuthToken
	}

	if token == "" {
		fmt.Println("No OAuth token found. Starting authentication...")
		fmt.Println("(You can also pass --token=<token> or set PUTIO_TOKEN)")
		fmt.Println()

		var err error
		token, err = auth.Authenticate(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
			os.Exit(1)
		}

		if err := config.Save(&config.Config{OAuthToken: token}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save token: %v\n", err)
		} else {
			fmt.Println("Authentication successful! Token saved.")
		}
	}

	m := ui.NewModel(token)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
