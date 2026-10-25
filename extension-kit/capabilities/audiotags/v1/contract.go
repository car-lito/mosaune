// Package audiotags defines file-scoped metadata extraction, independent of the catalogue.
package audiotags

import (
	"errors"
	"unicode/utf8"
)

const Capability = "media.audio-tags.v1"
const Service = "media.file.read.v1"
const BlockSize = 64 << 10
const MaxRead = 16 << 20
const MaxCalls = 512
const MaxPicture = 8 << 20

type Query struct {
	Size int64 `json:"size"`
}
type Range struct {
	Offset int64 `json:"offset"`
	Length int   `json:"length"`
}
type Block struct {
	Data []byte `json:"data"`
}
type Result struct {
	Matched     bool   `json:"matched"`
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	AlbumArtist string `json:"albumArtist,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Year        int    `json:"year,omitempty"`
	Track       int    `json:"track,omitempty"`
	Disc        int    `json:"disc,omitempty"`
	Picture     []byte `json:"picture,omitempty"`
}

func (r Result) Validate() error {
	for _, s := range []string{r.Title, r.Artist, r.Album, r.AlbumArtist, r.Genre} {
		if !utf8.ValidString(s) || utf8.RuneCountInString(s) > 500 {
			return errors.New("invalid tag text")
		}
	}
	if r.Year < 0 || r.Year > 9999 || r.Track < 0 || r.Track > 100000 || r.Disc < 0 || r.Disc > 100000 || len(r.Picture) > MaxPicture {
		return errors.New("invalid tag data")
	}
	if !r.Matched && (r.Title != "" || r.Artist != "" || r.Album != "" || r.AlbumArtist != "" || r.Genre != "" || r.Year != 0 || r.Track != 0 || r.Disc != 0 || len(r.Picture) != 0) {
		return errors.New("unmatched result contains tags")
	}
	return nil
}
