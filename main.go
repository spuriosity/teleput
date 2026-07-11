package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	putio "github.com/putdotio/go-putio"
	"golang.org/x/oauth2"

	"github.com/jack/teleput/auth"
	"github.com/jack/teleput/config"
	"github.com/jack/teleput/ui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "login":
			runLogin()
			return
		case "logout":
			runLogout()
			return
		}
	}

	tokenFlag := flag.String("token", "", "put.io OAuth token (or set PUTIO_TOKEN env var)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	token := *tokenFlag
	if token == "" {
		token = os.Getenv("PUTIO_TOKEN")
	}
	if token == "" {
		token = cfg.OAuthToken
	}

	if token == "" {
		fmt.Println("No OAuth token found. Starting authentication...")
		fmt.Println("(You can also pass --token=<token> or set PUTIO_TOKEN)")
		fmt.Println()

		token, err = auth.Authenticate(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
			os.Exit(1)
		}

		cfg.OAuthToken = token
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save token: %v\n", err)
		} else {
			fmt.Println("Authentication successful! Token saved.")
		}
	}

	client := newPutioClient(token)
	m := ui.NewModel(client, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newPutioClient(token string) *putio.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return putio.NewClient(oauth2.NewClient(context.Background(), ts))
}

func runLogin() {
	token, err := auth.Authenticate(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
		os.Exit(1)
	}
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.OAuthToken = token
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Could not save token: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Logged in. Token saved.")
}

func runLogout() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if cfg.OAuthToken == "" {
		fmt.Println("Already logged out.")
		return
	}
	cfg.OAuthToken = ""
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Could not update config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Logged out. (Token removed locally; revoke it in put.io settings if needed.)")
}
