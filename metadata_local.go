package main

import (
	"io"
	"strings"
	"unicode"

	"github.com/dhowden/tag"
)

// The parser may seek over audio, but actual reads and operations are bounded.
// Huge or broken tags fall back to the filename without aborting the library scan.
type tagReader struct {
	io.ReadSeeker
	remaining  int64
	operations int
}

func (r *tagReader) Read(b []byte) (int, error) {
	r.operations--
	if r.operations <= 0 || r.remaining <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	if int64(len(b)) > r.remaining {
		b = b[:r.remaining]
	}
	n, err := r.ReadSeeker.Read(b)
	r.remaining -= int64(n)
	return n, err
}
func (r *tagReader) Seek(offset int64, whence int) (int64, error) {
	r.operations--
	if r.operations <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	return r.ReadSeeker.Seek(offset, whence)
}
func localMusic(r io.ReadSeeker, initial Metadata) (m Metadata, picture []byte) {
	m = initial
	defer func() {
		if recover() != nil {
			picture = nil
		}
	}()
	tags, err := tag.ReadFrom(&tagReader{ReadSeeker: r, remaining: 16 << 20, operations: 20000})
	if err != nil {
		return m, nil
	}
	assign := func(field string, dst *string, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			*dst = truncate(value, 500)
			m.Origins[field] = "tags"
		}
	}
	assign("title", &m.Title, tags.Title())
	assign("artist", &m.Artist, tags.Artist())
	assign("album", &m.Album, tags.Album())
	assign("albumArtist", &m.AlbumArtist, tags.AlbumArtist())
	m.Year = tags.Year()
	m.Track, _ = tags.Track()
	m.Disc, _ = tags.Disc()
	m.Provider = "tags"
	if genre := tags.Genre(); genre != "" {
		m.Genres = []string{truncate(genre, 200)}
	}
	if p := tags.Picture(); p != nil && len(p.Data) <= 8<<20 {
		picture = p.Data
	}
	return m, picture
}
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
func normalizedTitle(s string) string {
	s = strings.ReplaceAll(strings.ToLower(s), "&", "and")
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}
func exactCandidate(q MetadataQuery, candidates []MetadataCandidate) (MetadataCandidate, bool) {
	var match MetadataCandidate
	count := 0
	for _, c := range candidates {
		if (normalizedTitle(c.Title) == normalizedTitle(q.Title) || normalizedTitle(c.OriginalTitle) == normalizedTitle(q.Title)) && (q.Year == 0 || q.Year == c.Year) {
			match = c
			count++
		}
	}
	return match, count == 1
}
