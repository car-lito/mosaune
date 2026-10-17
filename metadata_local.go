package main

import (
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/dhowden/tag"
)

var episodePattern = regexp.MustCompile(`(?i)(?:s(\d{1,2})[ ._-]*e(\d{1,3})|\b(\d{1,2})x(\d{1,3}))`)
var multiEpisodePattern = regexp.MustCompile(`(?i)^(?:[ ._-]*e|-)(\d{1,3})`)
var yearPattern = regexp.MustCompile(`(?:^|[ ._\[(])((?:19|20)\d{2})(?:$|[ ._\])])`)
var releasePattern = regexp.MustCompile(`(?i)\b(?:2160p|1080p|720p|480p|webrip|web[ .-]?dl|bluray|bdrip|hdtv|x264|x265|h[ .]?264|h[ .]?265)\b`)

func cleanTitle(value string) string {
	return strings.Trim(strings.Join(strings.Fields(strings.NewReplacer(".", " ", "_", " ").Replace(value)), " "), " -[]()")
}
func localVideo(kind, filename string) Metadata {
	base := strings.TrimSuffix(path.Base(filename), path.Ext(filename))
	m := Metadata{Title: cleanTitle(base), Provider: "filename", MatchState: "local", Origins: map[string]string{"title": "filename"}}
	if kind == "series" {
		loc := episodePattern.FindStringSubmatchIndex(base)
		if loc != nil {
			capture := func(a, b int) int {
				if loc[a] < 0 {
					return 0
				}
				v, _ := strconv.Atoi(base[loc[a]:loc[b]])
				return v
			}
			m.Season = capture(2, 3)
			m.Episode = capture(4, 5)
			if loc[2] < 0 {
				m.Season = capture(6, 7)
				m.Episode = capture(8, 9)
			}
			m.Series = cleanTitle(base[:loc[0]])
			if m.Series == "" {
				parent := path.Dir(filename)
				if strings.HasPrefix(strings.ToLower(path.Base(parent)), "saison") || strings.HasPrefix(strings.ToLower(path.Base(parent)), "season") {
					parent = path.Dir(parent)
				}
				if parent != "." {
					m.Series = cleanTitle(path.Base(parent))
				}
			}
			rest := base[loc[1]:]
			if match := multiEpisodePattern.FindStringSubmatch(rest); match != nil {
				m.EpisodeEnd, _ = strconv.Atoi(match[1])
				rest = rest[len(match[0]):]
			}
			if i := releasePattern.FindStringIndex(rest); i != nil {
				rest = rest[:i[0]]
			}
			m.Title = cleanTitle(rest)
			if m.Title == "" {
				m.Title = fmt.Sprintf("Épisode %d", m.Episode)
			}
		}
	} else {
		if loc := yearPattern.FindStringSubmatchIndex(base); loc != nil {
			m.Year, _ = strconv.Atoi(base[loc[2]:loc[3]])
			m.Title = cleanTitle(base[:loc[0]])
		} else if i := releasePattern.FindStringIndex(base); i != nil {
			m.Title = cleanTitle(base[:i[0]])
		}
	}
	return m
}

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
func localMusic(r io.ReadSeeker, filename string) (m Metadata, picture []byte) {
	m = Metadata{Title: strings.TrimSuffix(path.Base(filename), path.Ext(filename)), Provider: "filename", MatchState: "local", Origins: map[string]string{"title": "filename"}}
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
