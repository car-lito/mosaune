// Package v1 describes a result's appearance independently of its media model.
// It grants no navigation, network or styling privileges to an extension.
package v1

import (
	"errors"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Preview struct {
	AspectRatio float64 `json:"aspectRatio"` // width / height, 0.5 through 3
	Shape       string  `json:"shape"`       // rectangle or circle (ratio 1)
	Fit         string  `json:"fit"`         // cover or contain
}
type Result struct {
	Version  int      `json:"version"`
	Label    string   `json:"label"`
	Keywords []string `json:"keywords,omitempty"`
	Preview  *Preview `json:"preview,omitempty"`
}

func (r *Result) Validate() error {
	if r == nil {
		return nil
	}
	bad := errors.New("invalid result presentation")
	text := func(s string) bool {
		return strings.TrimSpace(s) != "" && utf8.ValidString(s) && utf8.RuneCountInString(s) <= 40 && strings.IndexFunc(s, unicode.IsControl) < 0
	}
	if r.Version != 1 || !text(r.Label) || len(r.Keywords) > 3 {
		return bad
	}
	for _, k := range r.Keywords {
		if !text(k) {
			return bad
		}
	}
	if p := r.Preview; p != nil {
		if math.IsNaN(p.AspectRatio) || math.IsInf(p.AspectRatio, 0) || p.AspectRatio < 0.5 || p.AspectRatio > 3 || (p.Shape != "rectangle" && p.Shape != "circle") || (p.Fit != "cover" && p.Fit != "contain") || (p.Shape == "circle" && p.AspectRatio != 1) {
			return bad
		}
	}
	return nil
}
