// Package video holds rules shared by the Films and Series modules, not the server.
package video

import (
	"fmt"
	recognition "mosaune/extensionkit/capabilities/recognition/v1"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var episodePattern = regexp.MustCompile(`(?i)(?:s(\d{1,2})[ ._-]*e(\d{1,3})|\b(\d{1,2})x(\d{1,3}))`)
var multiEpisodePattern = regexp.MustCompile(`(?i)^(?:[ ._-]*e|-)(\d{1,3})`)
var yearPattern = regexp.MustCompile(`(?:^|[ ._\[(])((?:19|20)\d{2})(?:$|[ ._\])])`)
var releasePattern = regexp.MustCompile(`(?i)\b(?:2160p|1080p|720p|480p|webrip|web[ .-]?dl|bluray|bdrip|hdtv|x264|x265|h[ .]?264|h[ .]?265)\b`)

func cleanTitle(value string) string {
	return strings.Trim(strings.Join(strings.Fields(strings.NewReplacer(".", " ", "_", " ").Replace(value)), " "), " -[]()")
}
func Identify(kind, filename string) recognition.Item {
	base := strings.TrimSuffix(path.Base(filename), path.Ext(filename))
	m := recognition.Item{Title: cleanTitle(base)}
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
	if m.Title == "" {
		m.Title = path.Base(filename)
	}
	return m
}
