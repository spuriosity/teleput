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
			"book series uses consistent sortable name",
			Parsed{Original: "Matt Dinniman - [Dungeon Crawler Carl 02] - Carl's Doomsday Scenario (epub).epub", Type: "book", Title: "Carl's Doomsday Scenario", Author: "Matt Dinniman", Series: "Dungeon Crawler Carl", SeriesIndex: 2},
			"Books/Matt Dinniman/Dungeon Crawler Carl 02 - Carl's Doomsday Scenario.epub",
		},
		{
			"book series pads single-digit volume numbers",
			Parsed{Original: "The Eye of the Bedlam Bride_ Du - Matt Dinniman.epub", Type: "book", Title: "The Eye of the Bedlam Bride", Author: "Matt Dinniman", Series: "Dungeon Crawler Carl", SeriesIndex: 6},
			"Books/Matt Dinniman/Dungeon Crawler Carl 06 - The Eye of the Bedlam Bride.epub",
		},
		{
			"book series preserves fractional volume numbers",
			Parsed{Original: "novella.epub", Type: "book", Title: "The Butcher's Masquerade Side Story", Author: "Matt Dinniman", Series: "Dungeon Crawler Carl", SeriesIndex: 5.5},
			"Books/Matt Dinniman/Dungeon Crawler Carl 05.5 - The Butcher's Masquerade Side Story.epub",
		},
		{
			"audiobook series uses author and sortable series name",
			Parsed{Original: "The Gate of the Feral Gods.m4b", Type: "audiobook", Title: "The Gate of the Feral Gods", Author: "Matt Dinniman", Series: "Dungeon Crawler Carl", SeriesIndex: 4},
			"Books/Audiobooks/Matt Dinniman/Dungeon Crawler Carl/(04) The Gate of the Feral Gods/The Gate of the Feral Gods.m4b",
		},
		{
			"movie franchise metadata does not damage media-server matching",
			Parsed{Original: "Aliens.1986.mkv", Type: "movie", Title: "Aliens", Year: 1986, Series: "Alien", SeriesIndex: 2},
			"Movies/Aliens (1986)/Aliens (1986).mkv",
		},
		{
			"multipart audiobook preserves chapter identity",
			Parsed{Original: "The First Chronicles - 02 - Prologue.mp3", Type: "audiobook", Title: "The First Chronicles of Druss the Legend", Author: "David Gemmell", Part: 2, PartTitle: "Book 1, Prologue"},
			"Books/Audiobooks/David Gemmell/The First Chronicles of Druss the Legend/02 - Book 1, Prologue.mp3",
		},
		{
			"tabletop rpg follows hubstore system categories",
			Parsed{Original: "Impossible Landscapes.pdf", Type: "rpg", Title: "Impossible Landscapes", System: "Delta Green", Category: "Campaigns"},
			"Books/RPG/Systems/Delta Green/Campaigns/Impossible Landscapes.pdf",
		},
		{
			"role-playing game uses genre and platform",
			Parsed{Original: "Baldurs.Gate.3.iso", Type: "game", Title: "Baldur's Gate 3", Year: 2023, Genre: "RPG", Platform: "PC"},
			"Games/RPG/PC/Baldur's Gate 3 (2023)/Baldur's Gate 3 (2023).iso",
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
