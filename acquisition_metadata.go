package main

import (
	"context"
	"errors"
	policy "mosaune/extensionkit/capabilities/acquisition/v1"
	"path"
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
func readAcquisitionImports(ctx context.Context, r *cataloguer, q policy.PreviewQuery) (policy.ImportPlan, error) {
	var p policy.ImportPlan
	if err := r.call(ctx, "import-plan", q, &p); err != nil {
		return p, errors.New("Mettez à jour le composant de demandes pour associer les fichiers importés.")
	}
	return p, p.Validate(q)
}
func (c *catalog) prioritizeImport(ctx context.Context, client arrClient, v acquisitionRequest, obj arrObject, episodes []arrObject) error {
	client.config.owner = c
	ctx, done, err := client.openSession(ctx)
	if err != nil {
		return err
	}
	defer done()
	session := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession)
	reader, err := loadFactProjector(ctx, session.descriptor, session.code, policy.Capability)
	if err != nil {
		return err
	}
	defer reader.engine.Close()
	scope := policy.PreviewQuery{Kind: v.Kind, Season: v.Season}
	plan, err := readAcquisitionImports(ctx, reader, scope)
	if err != nil {
		return err
	}
	if plan.Mode == "deferred" {
		return nil
	}
	root := v.Root
	if root == "" && v.Library == client.config.Library {
		root = client.config.Root
	}
	dir := word(obj["path"])
	if _, ok := mappedImport(root, dir, "probe"); !ok {
		return nil
	}
	var files []map[string]any
	if v.Kind == "films" {
		if f, ok := obj["movieFile"].(map[string]any); ok {
			files = append(files, f)
		}
	}
	for _, op := range plan.Reads {
		if err = client.exchange(ctx, op, v.RemoteID, 0, nil, &files); err != nil {
			return err
		}
	}
	rows := make([]map[string]any, len(episodes))
	for i := range episodes {
		rows[i] = episodes[i]
	}
	candidates, err := policy.ProjectImportCandidates(scope, rows, files)
	if err != nil {
		return err
	}
	paths := map[int]string{}
	pathCounts := map[string]int{}
	facts := []policy.ImportCandidate{}
	for _, candidate := range candidates {
		if rel, ok := mappedImport(root, dir, word(files[candidate.Slot]["relativePath"])); ok {
			paths[candidate.Slot] = rel
			pathCounts[rel]++
		}
	}
	for _, candidate := range candidates {
		if rel, ok := paths[candidate.Slot]; ok && pathCounts[rel] == 1 {
			facts = append(facts, candidate)
		}
	}
	query := policy.ImportQuery{Scope: scope, Candidates: facts}
	if err = query.Validate(); err != nil {
		return err
	}
	var selected policy.ImportResult
	if err = reader.call(ctx, "match-imports", query, &selected); err != nil {
		return err
	}
	if err = selected.Validate(query, plan); err != nil {
		return err
	}
	bySlot := map[int]policy.ImportCandidate{}
	for _, f := range facts {
		bySlot[f.Slot] = f
	}
	for _, slot := range selected.Slots {
		if err = ctx.Err(); err != nil {
			return err
		}
		f := bySlot[slot]
		if err = c.prioritizeMedia(ctx, v.Library, paths[slot], v.WorkID, f.Season, f.Episode); err != nil {
			return err
		}
	}
	return nil
}
