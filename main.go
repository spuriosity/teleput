package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

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

	// If a positional arg is given, treat it as a torrent file to upload
	if args := flag.Args(); len(args) > 0 {
		uploadTorrent(token, args[0], *interactive)
		return
	}

	m := ui.NewModel(token)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func uploadTorrent(token, path string, interactive bool) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	client := putio.NewClient(httpClient)

	fmt.Printf("Uploading %s...\n", path)
	upload, err := client.Files.Upload(context.Background(), f, f.Name(), -1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Upload failed: %v\n", err)
		os.Exit(1)
	}

	if upload.Transfer == nil {
		fmt.Println("File uploaded (not a torrent — no transfer created)")
		return
	}

	fmt.Printf("Transfer created: %s\n", upload.Transfer.Name)

	if interactive {
		m := ui.NewTransfersModel(token)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Headless: poll and show progress
	transferID := upload.Transfer.ID
	for {
		t, err := client.Transfers.Get(context.Background(), transferID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError polling transfer: %v\n", err)
			os.Exit(1)
		}

		speed := ""
		if t.DownloadSpeed > 0 {
			speed = cliHumanSize(int64(t.DownloadSpeed)) + "/s"
		}
		eta := ""
		if t.EstimatedTime > 0 {
			eta = "ETA " + cliFormatDuration(t.EstimatedTime)
		}

		fmt.Printf("\r\033[K  %s  %d%%  %s  %s  %s",
			t.Name, t.PercentDone, speed, eta, t.Status)

		if t.Status == "COMPLETED" || t.Status == "SEEDING" {
			fmt.Printf("\n\nDone! File available in your put.io account.\n")
			return
		}
		if t.Status == "ERROR" {
			fmt.Printf("\n\nTransfer failed: %s\n", t.ErrorMessage)
			os.Exit(1)
		}

		time.Sleep(3 * time.Second)
	}
}

func cliHumanSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func cliFormatDuration(seconds int64) string {
	if seconds <= 0 {
		return ""
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
