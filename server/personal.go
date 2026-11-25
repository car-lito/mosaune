package main

import "context"

func accountModules(ctx context.Context, modules []module) []module {
	result := []module{}
	for _, m := range modules {
		copy := m
		copy.Kinds = []string{}
		for _, k := range m.Kinds {
			if canKind(ctx, k) {
				copy.Kinds = append(copy.Kinds, k)
			}
		}
		if len(copy.Kinds) > 0 {
			result = append(result, copy)
		}
	}
	return result
}
func (c *catalog) personalProgress(ctx context.Context, items []media) error {
	type progress struct {
		revision           string
		position, duration float64
		completed          bool
		played             int64
	}
	values := map[string]progress{}
	rows, e := c.db.QueryContext(ctx, `SELECT media_id,revision,position,duration,completed,last_played FROM user_progress WHERE user_id=?`, userID(ctx))
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var p progress
		if e = rows.Scan(&id, &p.revision, &p.position, &p.duration, &p.completed, &p.played); e != nil {
			return e
		}
		values[id] = p
	}
	if e = rows.Err(); e != nil {
		return e
	}
	for i := range items {
		m := &items[i]
		m.Position = 0
		m.Duration = 0
		m.Completed = false
		m.LastPlayed = 0
		if p, ok := values[m.ID]; ok && p.revision == m.Revision {
			m.Position = p.position
			m.Duration = p.duration
			m.Completed = p.completed
			m.LastPlayed = p.played
		}
	}
	return nil
}
