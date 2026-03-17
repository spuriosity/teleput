package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	putio "github.com/putdotio/go-putio"
	"golang.org/x/oauth2"

	"github.com/jack/teleput/auth"
	"github.com/jack/teleput/config"
	"github.com/jack/teleput/ui"
)

func resolveToken(tokenFlag string) string {
	if tokenFlag != "" {
		return tokenFlag
	}

	if token := os.Getenv("PUTIO_TOKEN"); token != "" {
		return token
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if cfg.OAuthToken != "" {
		return cfg.OAuthToken
	}

	fmt.Println("No OAuth token found. Starting authentication...")
	fmt.Println("(You can also pass --token=<token> or set PUTIO_TOKEN)")
	fmt.Println()

	token, err := auth.Authenticate(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
		os.Exit(1)
	}

	if err := config.Save(&config.Config{OAuthToken: token}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save token: %v\n", err)
	} else {
		fmt.Println("Authentication successful! Token saved.")
	}

	return token
}

func main() {
	tokenFlag := flag.String("token", "", "put.io OAuth token (or set PUTIO_TOKEN env var)")
	interactive := flag.Bool("interactive", false, "launch TUI after uploading torrent")
	flag.Parse()

	token := resolveToken(*tokenFlag)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// If positional args are given, upload them to put.io
	if args := flag.Args(); len(args) > 0 {
		uploadFiles(token, cfg, args, *interactive)
		return
	}

	m := ui.NewModel(token, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func uploadFiles(token string, cfg *config.Config, paths []string, interactive bool) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	client := putio.NewClient(httpClient)

	var transferIDs []int64

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", path, err)
			os.Exit(1)
		}

		fmt.Printf("Uploading %s...", path)
		upload, err := client.Files.Upload(context.Background(), f, f.Name(), -1)
		f.Close()
		if err != nil {
			if strings.Contains(err.Error(), "TRANSFER_ALREADY_ADDED") {
				fmt.Println(" already exists, skipping")
				continue
			}
			fmt.Fprintf(os.Stderr, " failed: %v\n", err)
			os.Exit(1)
		}

		if upload.Transfer != nil {
			fmt.Printf(" transfer created: %s\n", upload.Transfer.Name)
			transferIDs = append(transferIDs, upload.Transfer.ID)
		} else {
			fmt.Println(" done")
		}
	}

	if interactive && len(transferIDs) > 0 {
		m := ui.NewTransfersModel(token, cfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

