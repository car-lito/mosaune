package main

import (
	"strings"
	"unicode"
)

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
