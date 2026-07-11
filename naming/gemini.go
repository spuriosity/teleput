package naming

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	defaultModel   = "gemini-flash-latest"
)

type Client struct {
	Token   string
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func NewClient(token string) *Client {
	return &Client{Token: token, BaseURL: defaultBaseURL, Model: defaultModel, HTTP: http.DefaultClient}
}

const promptHeader = `Classify each of these media release file names. Return ONLY a JSON array,
one object per input, each with keys:
  original (echo the input exactly), type (one of movie|tv|music|book|extra|other),
  title, year (int, 0 if unknown), season (int), episode (int), episode_title, author.
Use type "extra" for bonus/behind-the-scenes material belonging to a movie, with
title/year set to that movie. Use "other" for samples, nfo, txt and anything unclear.
Input names, one per line:
`

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMimeType string `json:"responseMimeType"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// Normalise classifies a batch of release names in one Gemini call. Results
// are matched by the echoed original name, falling back to input order.
func (c *Client) Normalise(ctx context.Context, names []string) ([]Parsed, error) {
	body, err := json.Marshal(geminiRequest{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: promptHeader + strings.Join(names, "\n")}}}},
		GenerationConfig: generationConfig{ResponseMimeType: "application/json"},
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", c.BaseURL, c.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.Token)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini status %d: %.200s", resp.StatusCode, raw)
	}

	var gr geminiResponse
	if err := json.Unmarshal(raw, &gr); err != nil {
		return nil, fmt.Errorf("decoding gemini envelope: %w", err)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	var parsed []Parsed
	if err := json.Unmarshal([]byte(gr.Candidates[0].Content.Parts[0].Text), &parsed); err != nil {
		return nil, fmt.Errorf("decoding gemini result: %w", err)
	}

	byOriginal := make(map[string]Parsed, len(parsed))
	for _, p := range parsed {
		byOriginal[p.Original] = p
	}
	out := make([]Parsed, len(names))
	for i, n := range names {
		if p, ok := byOriginal[n]; ok {
			out[i] = p
		} else if i < len(parsed) {
			p := parsed[i]
			p.Original = n
			out[i] = p
		} else {
			out[i] = Parsed{Original: n, Type: "other"}
		}
	}
	return out, nil
}
