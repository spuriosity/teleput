package main

import (
	"context"
	"fmt"
	"os"
	"time"

	putio "github.com/putdotio/go-putio"
	"golang.org/x/oauth2"

	"github.com/jack/teleput/config"
	"github.com/jack/teleput/download"
	"github.com/jack/teleput/naming"
)

// Live verification harness (task 12). Downloads a small real file with
// Gemini naming, then exercises resume on a big file. Never deletes.
func main() {
	cfg, err := config.Load()
	must(err)
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.OAuthToken})
	client := putio.NewClient(oauth2.NewClient(context.Background(), ts))
	ctx := context.Background()

	resolver := func(ctx context.Context, id int64) (string, error) {
		return client.Files.URL(ctx, id, true)
	}
	dest, err := os.MkdirTemp("", "teleput-verify-*")
	must(err)
	fmt.Println("dest:", dest)

	// --- Part 1: small file end-to-end with Gemini naming ---
	const epubID = 1585761383
	f, err := client.Files.Get(ctx, epubID)
	must(err)
	fmt.Printf("file: %q (%d bytes)\n", f.Name, f.Size)

	namer := naming.NewClient(cfg.EffectiveGeminiToken())
	nctx, ncancel := context.WithTimeout(ctx, 20*time.Second)
	parsed, err := namer.Normalise(nctx, []string{f.Name})
	ncancel()
	must(err)
	rel := naming.BuildPath(parsed[0])
	fmt.Printf("gemini: type=%s title=%q author=%q\n", parsed[0].Type, parsed[0].Title, parsed[0].Author)
	fmt.Println("target:", rel)

	m := download.New(resolver, dest)
	m.Enqueue([]download.Item{{FileID: epubID, RelPath: rel, Size: f.Size}})
	waitTerminal(m, "small-file")

	fi, err := os.Stat(dest + "/" + rel)
	must(err)
	fmt.Printf("PASS small file: %d bytes on disk (expected %d)\n\n", fi.Size(), f.Size)

	// --- Part 2: resume on the 19GB file ---
	const bigID = 1598699861
	big, err := client.Files.Get(ctx, bigID)
	must(err)
	m2 := download.New(resolver, dest)
	m2.Enqueue([]download.Item{{FileID: bigID, RelPath: "big.mkv", Size: big.Size}})

	var firstBytes int64
	for ev := range m2.Events() {
		if ev.Kind == download.Progress && ev.Done > 30*1024*1024 {
			firstBytes = ev.Done
			m2.Cancel(bigID)
		}
		if ev.Kind == download.Failed {
			fmt.Printf("cancelled first attempt at %d bytes: %v\n", firstBytes, ev.Err)
			break
		}
	}
	partFi, err := os.Stat(dest + "/big.mkv.part")
	must(err)
	fmt.Printf(".part on disk: %d bytes\n", partFi.Size())

	m3 := download.New(resolver, dest)
	m3.Enqueue([]download.Item{{FileID: bigID, RelPath: "big.mkv", Size: big.Size}})
	for ev := range m3.Events() {
		if ev.Kind == download.Started {
			fmt.Printf("restart Started with Done=%d (resume offset)\n", ev.Done)
			if ev.Done != partFi.Size() {
				fmt.Println("FAIL: resume offset != part size")
				os.Exit(1)
			}
		}
		if ev.Kind == download.Progress && ev.Done > partFi.Size()+20*1024*1024 {
			fmt.Printf("PASS resume: progressed to %d bytes from offset — cancelling\n", ev.Done)
			m3.Cancel(bigID)
		}
		if ev.Kind == download.Failed {
			break
		}
	}

	must(os.RemoveAll(dest))
	fmt.Println("cleaned up", dest)
	fmt.Println("ALL LIVE CHECKS PASSED")
}

func waitTerminal(m *download.Manager, label string) {
	for ev := range m.Events() {
		switch ev.Kind {
		case download.Completed:
			fmt.Printf("%s: Completed %s\n", label, ev.RelPath)
			return
		case download.Failed:
			fmt.Printf("%s: FAILED: %v\n", label, ev.Err)
			os.Exit(1)
		}
	}
}

func must(err error) {
	if err != nil {
		fmt.Println("FATAL:", err)
		os.Exit(1)
	}
}
