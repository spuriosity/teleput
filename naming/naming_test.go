package naming

import "testing"

func TestShouldBuildMediaServerPaths(t *testing.T) {
	tests := []struct {
		name string
		in   Parsed
		want string
	}{
		{
			"movie with year",
			Parsed{Original: "No.Country.For.Old.Men.2007.2160p.x265-GRP.mkv", Type: "movie", Title: "No Country for Old Men", Year: 2007},
			"Movies/No Country for Old Men (2007)/No Country for Old Men (2007).mkv",
		},
		{
			"movie without year",
			Parsed{Original: "some.movie.mkv", Type: "movie", Title: "Some Movie"},
			"Movies/Some Movie/Some Movie.mkv",
		},
		{
			"movie missing title falls back",
			Parsed{Original: "mystery.mkv", Type: "movie"},
			"Other/mystery.mkv",
		},
		{
			"tv with episode title",
			Parsed{Original: "eureka.s01e03.1080p.mkv", Type: "tv", Title: "Eureka", Season: 1, Episode: 3, EpisodeTitle: "Before I Forget"},
			"TV/Eureka/Season 01/Eureka - S01E03 - Before I Forget.mkv",
		},
		{
			"tv without episode title",
			Parsed{Original: "xfiles.mkv", Type: "tv", Title: "The X-Files", Season: 2, Episode: 12},
			"TV/The X-Files/Season 02/The X-Files - S02E12.mkv",
		},
		{
			"tv missing season falls back",
			Parsed{Original: "weird.mkv", Type: "tv", Title: "Weird", Episode: 4},
			"Other/weird.mkv",
		},
		{
			"book with author",
			Parsed{Original: "dcc-book-1.epub", Type: "book", Title: "Dungeon Crawler Carl", Author: "Matt Dinniman"},
			"Books/Matt Dinniman/Dungeon Crawler Carl.epub",
		},
		{
			"book without author",
			Parsed{Original: "anon.epub", Type: "book", Title: "Anonymous Tales"},
			"Books/Anonymous Tales.epub",
		},
		{
			"music",
			Parsed{Original: "album.flac", Type: "music", Title: "Blue Train"},
			"Music/Blue Train.flac",
		},
		{
			"extra under movie",
			Parsed{Original: "Extras/making-of.mkv", Type: "extra", Title: "No Country for Old Men", Year: 2007},
			"Movies/No Country for Old Men (2007)/Extras/making-of.mkv",
		},
		{
			"other passes through",
			Parsed{Original: "readme.nfo", Type: "other"},
			"Other/readme.nfo",
		},
		{
			"sanitises unsafe characters",
			Parsed{Original: "x.mkv", Type: "movie", Title: "Face/Off: Special?", Year: 1997},
			"Movies/Face-Off - Special (1997)/Face-Off - Special (1997).mkv",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildPath(tt.in); got != tt.want {
				t.Fatalf("BuildPath = %q, want %q", got, tt.want)
			}
		})
	}
}
