package v1

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestPresentationOpenLabelsAndBoundedGeometry(t *testing.T) {
	var absent *Result
	if absent.Validate() != nil {
		t.Fatal("optional legacy field")
	}
	for _, label := range []string{"Album", "Single", "Saison", "Artiste", "Acteur", "Jeu", "Console"} {
		r := &Result{Version: 1, Label: label, Keywords: []string{"Nintendo Switch"}, Preview: &Preview{1, "circle", "contain"}}
		if e := r.Validate(); e != nil {
			t.Fatal(e)
		}
	}
	for _, r := range []Result{{Version: 2, Label: "Jeu"}, {Version: 1, Label: "\nFilm"}, {Version: 1, Label: strings.Repeat("é", 41)}, {Version: 1, Label: "Jeu", Keywords: []string{"1", "2", "3", "4"}}, {Version: 1, Label: "Jeu", Preview: &Preview{math.NaN(), "rectangle", "cover"}}, {Version: 1, Label: "Jeu", Preview: &Preview{4, "rectangle", "cover"}}, {Version: 1, Label: "Acteur", Preview: &Preview{2, "circle", "cover"}}, {Version: 1, Label: "Jeu", Preview: &Preview{1, "url(x)", "cover"}}} {
		if r.Validate() == nil {
			t.Fatal("accepted", r)
		}
	}
	var r Result
	if json.Unmarshal([]byte(`{"version":1,"label":"Jeu","keywords":["Console"],"preview":{"aspectRatio":0.62,"shape":"rectangle","fit":"contain"}}`), &r) != nil || r.Validate() != nil {
		t.Fatal("extension example not supported")
	}
}
