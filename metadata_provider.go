package main

import "context"

// MetadataProvider is the version 1, built-in provider contract. The host owns
// credentials, persistence, matching policy and manual choices. A future plugin
// process can implement this contract over RPC without exposing catalog internals.
// In-process providers are trusted code, NOT a sandbox for downloaded extensions.
type MetadataProvider interface {
	Info() ProviderInfo
	Search(context.Context, string, MetadataQuery) ([]MetadataCandidate, error)
	Details(context.Context, string, MetadataSelection) (Metadata, error)
	Artwork(context.Context, string, string) ([]byte, error)
}

type ProviderInfo struct {
	Enabled            bool     `json:"enabled"`
	Discovery          bool     `json:"discovery"`
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	ContractVersion    int      `json:"contractVersion"`
	Kinds              []string `json:"kinds"`
	Configured         bool     `json:"configured"`
	RequiresCredential bool     `json:"requiresCredential"`
	NetworkHosts       []string `json:"networkHosts"`
	Attribution        string   `json:"attribution"`
	Website            string   `json:"website"`
}
type MetadataQuery struct {
	Kind  string
	Title string
	Year  int
}
type MetadataSelection struct {
	Kind    string
	ID      string
	Season  int
	Episode int
}
type MetadataCandidate struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"originalTitle"`
	Year          int    `json:"year"`
	Overview      string `json:"overview"`
}
type Contributor struct {
	Image string `json:"image,omitempty"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role,omitempty"`
}
type CatalogTrack struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Disc     int           `json:"disc"`
	Number   int           `json:"number"`
	Duration int           `json:"duration"`
	Artists  []Contributor `json:"artists,omitempty"`
}
type Trailer struct {
	Thumbnail string `json:"thumbnail,omitempty"`
	Language  string `json:"language,omitempty"`
	Name      string `json:"name"`
	URL       string `json:"url"`
}
type SeasonInfo struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Image    string `json:"image,omitempty"`
	Episodes int    `json:"episodes"`
}
type Metadata struct {
	DetailVersion int            `json:"detailVersion,omitempty"`
	Seasons       []SeasonInfo   `json:"seasons,omitempty"`
	Runtime       int            `json:"runtime,omitempty"`
	ReleaseDate   string         `json:"releaseDate,omitempty"`
	Tagline       string         `json:"tagline,omitempty"`
	Rating        float64        `json:"rating,omitempty"`
	VoteCount     int            `json:"voteCount,omitempty"`
	Status        string         `json:"status,omitempty"`
	Countries     []string       `json:"countries,omitempty"`
	Languages     []string       `json:"languages,omitempty"`
	Contributors  []Contributor  `json:"contributors,omitempty"`
	Trailers      []Trailer      `json:"trailers,omitempty"`
	Tracks        []CatalogTrack `json:"tracks,omitempty"`
	Edition       string         `json:"edition,omitempty"`

	Title       string            `json:"title"`
	Series      string            `json:"series,omitempty"`
	Season      int               `json:"season,omitempty"`
	Episode     int               `json:"episode,omitempty"`
	EpisodeEnd  int               `json:"episodeEnd,omitempty"`
	Artist      string            `json:"artist,omitempty"`
	Album       string            `json:"album,omitempty"`
	AlbumArtist string            `json:"albumArtist,omitempty"`
	Track       int               `json:"track,omitempty"`
	Disc        int               `json:"disc,omitempty"`
	Year        int               `json:"year,omitempty"`
	Overview    string            `json:"overview,omitempty"`
	Cast        []string          `json:"cast,omitempty"`
	Genres      []string          `json:"genres,omitempty"`
	ImageID     string            `json:"imageId,omitempty"`
	ArtworkRef  string            `json:"-"`
	Provider    string            `json:"provider"`
	ProviderID  string            `json:"providerId,omitempty"`
	MatchState  string            `json:"matchState"`
	Manual      bool              `json:"manual,omitempty"`
	Origins     map[string]string `json:"origins,omitempty"`
}

func supportsProvider(p MetadataProvider, kind string) bool {
	for _, k := range p.Info().Kinds {
		if kind == k {
			return true
		}
	}
	return false
}
