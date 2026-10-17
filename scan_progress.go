package main

import "time"

// In-memory progress is separate from SQLite so it remains readable while the
// transactional index is being written. A restart leaves the prior catalog intact.
type scanProgress struct {
	LibraryID string `json:"libraryId"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	State     string `json:"state"`
	Found     int    `json:"found"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Message   string `json:"message"`
	Updated   int64  `json:"updated"`
}

func (c *catalog) setScanProgress(v scanProgress) {
	c.scanStatusMu.Lock()
	defer c.scanStatusMu.Unlock()
	v.Updated = time.Now().Unix()
	c.scanStatus = v
}
func (c *catalog) getScanProgress() scanProgress {
	c.scanStatusMu.Lock()
	defer c.scanStatusMu.Unlock()
	return c.scanStatus
}
