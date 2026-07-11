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
	Original     string `json:"original"`
	Type         string `json:"type"` // movie|tv|music|book|extra|other
	Title        string `json:"title"`
	Year         int    `json:"year"`
	Season       int    `json:"season"`
	Episode      int    `json:"episode"`
	EpisodeTitle string `json:"episode_title"`
	Author       string `json:"author"`
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
		if p.Author != "" {
			return filepath.Join("Books", sanitize(p.Author), sanitize(p.Title)+ext)
		}
		return filepath.Join("Books", sanitize(p.Title)+ext)
	case "music":
		if p.Title == "" {
			break
		}
		return filepath.Join("Music", sanitize(p.Title)+ext)
	}
	return filepath.Join("Other", p.Original)
}

var unsafeChars = strings.NewReplacer(
	"/", "-", "\\", "-", ":", " -", "*", "", "?", "", "\"", "'", "<", "", ">", "", "|", "-",
)

func sanitize(s string) string {
	return strings.Trim(unsafeChars.Replace(s), " .")
}
