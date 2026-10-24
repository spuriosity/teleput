package naming

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultBaseURL        = "https://generativelanguage.googleapis.com/v1beta"
	defaultModel          = "gemini-flash-latest"
	defaultAttemptTimeout = 30 * time.Second
)

// defaultBackoff gives three attempts; Gemini 503s ("high demand") and
// stalled requests are usually transient.
var defaultBackoff = []time.Duration{2 * time.Second, 5 * time.Second}

type Client struct {
	Token   string
	BaseURL string
	Model   string
	HTTP    *http.Client
	// AttemptTimeout caps each request; Backoff holds the waits between
	// attempts, so len(Backoff)+1 attempts are made in total.
	AttemptTimeout time.Duration
	Backoff        []time.Duration
}

func NewClient(token string) *Client {
	return &Client{
		Token: token, BaseURL: defaultBaseURL, Model: defaultModel, HTTP: http.DefaultClient,
		AttemptTimeout: defaultAttemptTimeout, Backoff: defaultBackoff,
	}
}

// MaxDuration is the longest Normalise can take across all attempts.
func (c *Client) MaxDuration() time.Duration {
	d := c.AttemptTimeout * time.Duration(len(c.Backoff)+1)
	for _, b := range c.Backoff {
		d += b
	}
	return d
}

type statusError struct {
	code int
	body []byte
}

func (e *statusError) Error() string {
	var env struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(e.body, &env) == nil && env.Error.Message != "" {
		return fmt.Sprintf("gemini status %d: %s", e.code, env.Error.Message)
	}
	return fmt.Sprintf("gemini status %d: %.200s", e.code, e.body)
}

var errAttemptTimeout = errors.New("gemini did not reply in time")

func retryable(err error) bool {
	var se *statusError
	if errors.As(err, &se) {
		switch se.code {
		case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway,
			http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
		return false
	}
	var ue *url.Error
	return errors.Is(err, errAttemptTimeout) || errors.As(err, &ue)
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

// Normalise classifies a batch of release names in one Gemini call, retrying
// overload errors and stalled requests. Results are matched by the echoed
// original name, falling back to input order.
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

	var text string
	for attempt := 0; ; attempt++ {
		text, err = c.generate(ctx, body)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !retryable(err) {
			return nil, err
		}
		if attempt >= len(c.Backoff) {
			return nil, fmt.Errorf("%w (gave up after %d attempts)", err, attempt+1)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.Backoff[attempt]):
		}
	}

	var parsed []Parsed
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
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

// generate makes one generateContent call and returns the candidate text.
func (c *Client) generate(ctx context.Context, body []byte) (string, error) {
	if c.AttemptTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(ctx, c.AttemptTimeout, errAttemptTimeout)
		defer cancel()
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent", c.BaseURL, c.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.Token)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		if errors.Is(context.Cause(ctx), errAttemptTimeout) {
			return "", fmt.Errorf("%w within %s", errAttemptTimeout, c.AttemptTimeout)
		}
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(context.Cause(ctx), errAttemptTimeout) {
			return "", fmt.Errorf("%w within %s", errAttemptTimeout, c.AttemptTimeout)
		}
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", &statusError{code: resp.StatusCode, body: raw}
	}

	var gr geminiResponse
	if err := json.Unmarshal(raw, &gr); err != nil {
		return "", fmt.Errorf("decoding gemini envelope: %w", err)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}
	return gr.Candidates[0].Content.Parts[0].Text, nil
}
