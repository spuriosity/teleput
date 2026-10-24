package naming

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

const promptHeader = `Classify this complete dataset of media file names. Analyze all entries together
so repeated creators, titles, and collections use identical canonical metadata. Use established
catalogue knowledge when the filename alone does not establish metadata such as collection membership,
order, year, season, episode, or platform. Do not invent metadata when it cannot be determined reliably.

Return ONLY a JSON array,
one object per input, each with keys:
  original (echo the input exactly), type (one of movie|tv|music|book|audiobook|rpg|game|extra|other),
  title, year (int, 0 if unknown), season (int), episode (int), episode_title,
  author, series, series_index (number, 0 if unknown), genre, platform, system,
  category, part (int, 0 if not multipart), part_title.
Use type "extra" for bonus/behind-the-scenes material belonging to a movie, with
title/year set to that movie. Use "other" for samples, nfo, txt and anything unclear.
Use "audiobook" for narrated books (for example m4b/m4a/mp3 audio releases) and "book"
for textual books (for example epub/mobi). Preserve audiobook track/chapter identity in
part and part_title; all tracks belonging to one work must have identical work metadata.
Use "rpg" for tabletop role-playing game PDFs and set system (for example Delta Green)
and category to one of Core, Rules, Campaigns, Scenarios, Sourcebooks, Player Aids,
Settings, Periodicals and Zines, Artbooks, or Other. Use "game" for computer/video games and
set genre to the canonical broad genre (for example RPG) and platform when identifiable.
For books and audiobooks, title must be the canonical title only: remove the author, release-group
noise, series name, series number, and decorative brackets from it. Put the canonical
series name and volume number in series and series_index instead. Preserve meaningful
leading articles and subtitles. For example, "Matt Dinniman - [Dungeon Crawler Carl
02] - Carl's Doomsday Scenario (epub).epub" has title "Carl's Doomsday Scenario",
author "Matt Dinniman", series "Dungeon Crawler Carl", and series_index 2. Infer series
membership and order from grounded sources even when the dataset contains only one volume.
For movie franchises, put the canonical collection/franchise in series and chronological
release order in series_index as metadata only; do not change the canonical movie title.
For TV episodes, use the canonical show as title and populate
season, episode, and episode_title. Never infer an index merely from dataset order.
Classify companion cover, metadata, subtitle, or artwork files as the same media type and populate
the same work metadata as their parent. Preserve meaningful variants such as language,
forced subtitles, audiobook abridgement, edition, disc, and part identity; do not collapse
distinct files onto one destination.
Input JSON array:
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
	ResponseMimeType string          `json:"responseMimeType"`
	ThinkingConfig   *thinkingConfig `json:"thinkingConfig,omitempty"`
}

// thinking adds seconds of latency for zero gain on name classification
type thinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
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
	namesJSON, err := json.Marshal(names)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(geminiRequest{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: promptHeader + string(namesJSON)}}}},
		GenerationConfig: generationConfig{ResponseMimeType: "application/json", ThinkingConfig: &thinkingConfig{ThinkingBudget: 0}},
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
