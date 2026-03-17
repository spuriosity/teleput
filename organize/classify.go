package organize

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// claudeResponse is the outer envelope from `claude -p ... --output-format json`.
type claudeResponse struct {
	Result string `json:"result"`
}

// classifyResult is the parsed inner JSON from Claude's response.
type classifyResult struct {
	Files []Classification `json:"files"`
}

// Classify sends a list of files to the Claude CLI for AI-powered classification.
// Returns per-file classifications with category, new name, and metadata.
func Classify(ctx context.Context, files []FileInfo) ([]Classification, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return nil, fmt.Errorf("claude CLI not found — install it from https://docs.anthropic.com/en/docs/claude-code")
	}

	if len(files) == 0 {
		return nil, nil
	}

	prompt := buildPrompt(files)

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--output-format", "json")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude CLI failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("running claude CLI: %w", err)
	}

	// Parse outer envelope
	var envelope claudeResponse
	if err := json.Unmarshal(out, &envelope); err != nil {
		return nil, fmt.Errorf("parsing claude response envelope: %w", err)
	}

	// The result field contains the actual JSON (possibly with markdown fences)
	inner := extractJSON(envelope.Result)

	var result classifyResult
	if err := json.Unmarshal([]byte(inner), &result); err != nil {
		return nil, fmt.Errorf("parsing classification result: %w\nraw: %s", err, inner)
	}

	return result.Files, nil
}

func buildPrompt(files []FileInfo) string {
	var b strings.Builder

	b.WriteString("You are a media file classifier. Analyze the following files and classify each one.\n\n")
	b.WriteString("Files:\n")
	for _, f := range files {
		kind := "file"
		if f.IsDir {
			kind = "dir"
		}
		b.WriteString(fmt.Sprintf("- id=%d name=%q size=%d type=%q kind=%s\n",
			f.ID, f.Name, f.Size, f.ContentType, kind))
	}

	b.WriteString(`
Categories:
- movie: Feature films. Rename to "Title (Year).ext"
- tv: TV episodes. Rename to "Show Name - S01E02 - Episode Title.ext"
- music: Music files. Rename to "Artist - Album - 01 Track.ext"
- audiobook: Audiobook files. Rename to "Author - Title.ext" (or keep chapter structure)
- book: Ebooks (epub, pdf, mobi). Rename to "Author - Title.ext"
- junk: Non-media files like .nfo, .txt, .jpg, .png, sample/, .exe, .url, .nzb
- other: Cannot determine category

Rules:
- Preserve the original file extension
- For directories, classify based on name and contents would likely be
- Strip scene tags like 1080p, x264, BluRay, RARBG, YTS, etc from names
- Use proper title casing for names
- For TV shows, always include season and episode numbers
- If unsure, use "other"

Respond with ONLY valid JSON (no markdown fences), in this exact format:
{"files": [{"file_id": 123, "category": "movie", "new_name": "Movie Title (2024).mkv", "show_name": "", "season": 0, "episode": 0, "artist": "", "album": ""}]}
`)

	return b.String()
}

// extractJSON strips markdown code fences if present.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}
