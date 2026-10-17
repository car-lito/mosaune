package main

import (
	"context"
	"path"
	"strconv"
	"strings"
)

// Arr paths belong to the remote machine: do not interpret them with the local
// platform's filepath rules. Reject traversal and require an exact root prefix.
func mappedImport(root, dir, file string) (string, bool) {
	clean := func(s string) string { return strings.ReplaceAll(s, "\\", "/") }
	root, dir, file = clean(root), clean(dir), clean(file)
	if root == "" || dir == "" || file == "" || strings.HasPrefix(file, "/") || strings.Contains(file, ":") {
		return "", false
	}
	for _, s := range []string{root, dir, file} {
		for _, part := range strings.Split(s, "/") {
			if part == ".." {
				return "", false
			}
		}
	}
	root = strings.TrimRight(path.Clean(root), "/") + "/"
	dir = strings.TrimRight(path.Clean(dir), "/") + "/"
	if !strings.HasPrefix(dir, root) {
		return "", false
	}
	relative := path.Clean(strings.TrimPrefix(dir, root) + file)
	if relative == "." || strings.HasPrefix(relative, "/") || relative == ".." || strings.HasPrefix(relative, "../") {
		return "", false
	}
	return relative, true
}
func (c *catalog) prioritizeImport(ctx context.Context, a arrClient, v acquisitionRequest, obj arrObject, episodes []arrObject) error {
	root := v.Root
	if root == "" && v.Library == a.config.Library {
		root = a.config.Root
	} // pre-0.0.11 requests
	dir := word(obj["path"])
	if _, ok := mappedImport(root, dir, "probe"); !ok {
		return nil
	}
	if v.Kind == "films" {
		file, _ := obj["movieFile"].(map[string]any)
		if rel, ok := mappedImport(root, dir, word(file["relativePath"])); ok {
			return c.prioritizeMedia(ctx, v.Library, rel, v.WorkID, 0, 0)
		}
		return nil
	}
	var files []arrObject
	if err := a.call(ctx, "GET", "episodefile?seriesId="+strconv.Itoa(v.RemoteID), nil, &files); err != nil {
		return err
	}
	for _, file := range files {
		matches := []arrObject{}
		for _, ep := range episodes {
			if truth(ep["hasFile"]) && number(ep["episodeFileId"]) > 0 && number(ep["episodeFileId"]) == number(file["id"]) {
				matches = append(matches, ep)
			}
		}
		// Multi-episode files need the normal multi-episode review, not one guessed identity.
		if len(matches) != 1 {
			continue
		}
		ep := matches[0]
		season := number(ep["seasonNumber"])
		if v.Season >= 0 && v.Season != season {
			continue
		}
		if rel, ok := mappedImport(root, dir, word(file["relativePath"])); ok {
			if err := c.prioritizeMedia(ctx, v.Library, rel, v.WorkID, season, number(ep["episodeNumber"])); err != nil {
				return err
			}
		}
	}
	return nil
}
