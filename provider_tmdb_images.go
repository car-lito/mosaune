package main

import "strings"

func (*tmdbProvider) ImageReferenceKinds(ref string) []string {
	if artworkPath.MatchString(ref) || strings.HasPrefix(ref, "yt:") && youtubeKey.MatchString(strings.TrimPrefix(ref, "yt:")) {
		return []string{"films", "series"}
	}
	return nil
}
