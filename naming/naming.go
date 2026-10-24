package naming

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// Parsed is Gemini's classification of one release name. Original is echoed
// back so results can be matched to inputs regardless of response order.
type Parsed struct {
	Original     string  `json:"original"`
	Type         string  `json:"type"` // movie|tv|music|book|audiobook|rpg|game|extra|other
	Title        string  `json:"title"`
	Year         int     `json:"year"`
	Season       int     `json:"season"`
	Episode      int     `json:"episode"`
	EpisodeTitle string  `json:"episode_title"`
	Author       string  `json:"author"`
	Series       string  `json:"series"`
	SeriesIndex  float64 `json:"series_index"`
	Genre        string  `json:"genre"`
	Platform     string  `json:"platform"`
	System       string  `json:"system"`
	Category     string  `json:"category"`
	Part         int     `json:"part"`
	PartTitle    string  `json:"part_title"`
}

// BuildPath maps a classification onto the media-server layout. Anything
// missing a required field lands untouched under Other/.
func BuildPath(p Parsed) string {
	ext := path.Ext(p.Original)
	switch p.Type {
	case "movie":
		if p.Title == "" {
			break
		}
		base := sanitize(p.Title)
		if p.Year > 0 {
			base = fmt.Sprintf("%s (%d)", base, p.Year)
		}
		return filepath.Join("Movies", base, base+ext)
	case "extra":
		if p.Title == "" {
			break
		}
		base := sanitize(p.Title)
		if p.Year > 0 {
			base = fmt.Sprintf("%s (%d)", base, p.Year)
		}
		return filepath.Join("Movies", base, "Extras", path.Base(p.Original))
	case "tv":
		if p.Title == "" || p.Season <= 0 || p.Episode <= 0 {
			break
		}
		show := sanitize(p.Title)
		name := fmt.Sprintf("%s - S%02dE%02d", show, p.Season, p.Episode)
		if p.EpisodeTitle != "" {
			name += " - " + sanitize(p.EpisodeTitle)
		}
		return filepath.Join("TV", show, fmt.Sprintf("Season %02d", p.Season), name+ext)
	case "book":
		if p.Title == "" {
			break
		}
		title := seriesTitle(p)
		if p.Author != "" {
			return filepath.Join("Books", sanitize(p.Author), sanitize(title)+ext)
		}
		return filepath.Join("Books", sanitize(title)+ext)
	case "audiobook":
		if p.Title == "" {
			break
		}
		title := sanitize(p.Title)
		parts := []string{"Books", "Audiobooks"}
		if p.Author != "" {
			parts = append(parts, sanitize(p.Author))
		}
		if p.Series != "" && p.SeriesIndex > 0 {
			parts = append(parts, sanitize(p.Series), fmt.Sprintf("(%s) %s", formatOrdinal(p.SeriesIndex), title))
		} else {
			parts = append(parts, title)
		}
		filename := title
		if p.Part > 0 {
			filename = fmt.Sprintf("%02d - %s", p.Part, title)
			if p.PartTitle != "" {
				filename = fmt.Sprintf("%02d - %s", p.Part, sanitize(p.PartTitle))
			}
		}
		return filepath.Join(append(parts, filename+ext)...)
	case "rpg":
		if p.Title == "" || p.System == "" {
			break
		}
		parts := []string{"Books", "RPG", "Systems", sanitize(p.System)}
		if p.Category != "" {
			parts = append(parts, sanitize(p.Category))
		}
		return filepath.Join(append(parts, sanitize(p.Title)+ext)...)
	case "game":
		if p.Title == "" {
			break
		}
		title := sanitize(p.Title)
		if p.Year > 0 {
			title = fmt.Sprintf("%s (%d)", title, p.Year)
		}
		parts := []string{"Games"}
		if p.Genre != "" {
			parts = append(parts, sanitize(p.Genre))
		}
		if p.Platform != "" {
			parts = append(parts, sanitize(p.Platform))
		}
		return filepath.Join(append(parts, title, title+ext)...)
	case "music":
		if p.Title == "" {
			break
		}
		return filepath.Join("Music", sanitize(p.Title)+ext)
	}
	return filepath.Join("Other", p.Original)
}

func seriesTitle(p Parsed) string {
	if p.Series != "" && p.SeriesIndex > 0 {
		return fmt.Sprintf("%s %s - %s", p.Series, formatOrdinal(p.SeriesIndex), p.Title)
	}
	return p.Title
}

func formatOrdinal(index float64) string {
	if index == float64(int(index)) {
		return fmt.Sprintf("%02d", int(index))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%05.2f", index), "0"), ".")
}

var unsafeChars = strings.NewReplacer(
	"/", "-", "\\", "-", ":", " -", "*", "", "?", "", "\"", "'", "<", "", ">", "", "|", "-",
)

func sanitize(s string) string {
	return strings.Trim(unsafeChars.Replace(s), " .")
}
