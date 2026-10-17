// lab runs only the embedded, source-provided example. It never opens Mosaune's
// catalogue or accepts arbitrary binary paths, credentials or network addresses.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	metadata "mosaune/extensionkit/capabilities/metadata/v1"
	"mosaune/extensionkit/examples/demo"
	"mosaune/extensionkit/protocol"
	ext "mosaune/extensionkit/runtime"
	"os"
	"runtime"
	"slices"
	"time"
)

type heap struct {
	AllocatedBytes       uint64 `json:"allocatedBytes"`
	RuntimeReservedBytes uint64 `json:"runtimeReservedBytes"`
}

func memory() heap {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return heap{m.HeapAlloc, m.Sys}
}

type report struct {
	Platform                               string    `json:"platform"`
	GoVersion                              string    `json:"goVersion"`
	Engine                                 string    `json:"engine"`
	CodeBytes                              int       `json:"moduleBytes"`
	SHA256                                 string    `json:"moduleSHA256"`
	LoadMS                                 float64   `json:"loadMs"`
	FirstMS                                float64   `json:"firstCallMs"`
	MedianMS                               float64   `json:"warmMedianMs"`
	Samples                                []float64 `json:"warmSamplesMs"`
	MemoryLimitMiB                         uint32    `json:"linearMemoryLimitMiB"`
	Before, Loaded, AfterCalls, AfterClose heap
	Checks                                 []string `json:"checks"`
	FilmTitle                              string   `json:"filmTitle"`
}

func run() (report, error) {
	ctx := context.Background()
	l := ext.DefaultLimits()
	hash := sha256.Sum256(demo.Code)
	r := report{Platform: runtime.GOOS + "/" + runtime.GOARCH, GoVersion: runtime.Version(), Engine: "wazero v1.12.0", CodeBytes: len(demo.Code), SHA256: hex.EncodeToString(hash[:]), MemoryLimitMiB: l.MemoryMiB, Before: memory()}
	e, err := ext.New(ctx, l)
	if err != nil {
		return r, err
	}
	defer e.Close()
	d := demo.Descriptor()
	start := time.Now()
	if err = e.Load(ctx, d, demo.Code); err != nil {
		return r, err
	}
	r.LoadMS = float64(time.Since(start).Microseconds()) / 1000
	r.Loaded = memory()
	query := protocol.Request{Protocol: 1, ID: "lab-search", Capability: "metadata.search.v1", Operation: "search", Payload: json.RawMessage(`{"kind":"films","title":"Voyage"}`)}
	if _, err = e.Call(ctx, d.ID, query); !errors.Is(err, ext.ErrUnavailable) {
		return r, errors.New("loaded extension unexpectedly active")
	}
	r.Checks = append(r.Checks, "chargement inactif")
	if err = e.Enable(d.ID); err != nil {
		return r, err
	}
	first, err := e.Call(ctx, d.ID, query)
	if err != nil {
		return r, err
	}
	if first.Response.Error != nil {
		return r, errors.New(first.Response.Error.Message)
	}
	r.FirstMS = first.Metrics.ElapsedMS
	var search metadata.SearchResult
	if err = json.Unmarshal(first.Response.Result, &search); err != nil || search.Validate() != nil || len(search.Items) != 1 {
		return r, errors.New("invalid search result")
	}
	r.FilmTitle = search.Items[0].Title
	r.Checks = append(r.Checks, "recherche dans l’extension")
	for i := 0; i < 9; i++ {
		out, err := e.Call(ctx, d.ID, query)
		if err != nil {
			return r, err
		}
		r.Samples = append(r.Samples, out.Metrics.ElapsedMS)
	}
	sorted := slices.Clone(r.Samples)
	slices.Sort(sorted)
	r.MedianMS = sorted[len(sorted)/2]
	selected, _ := json.Marshal(map[string]string{"id": search.Items[0].ID})
	detail := protocol.Request{Protocol: 1, ID: "lab-detail", Capability: "metadata.details.v1", Operation: "details", Payload: selected}
	info, err := e.Call(ctx, d.ID, detail)
	if err != nil {
		return r, err
	}
	var film metadata.Candidate
	if json.Unmarshal(info.Response.Result, &film) != nil || film.Validate() != nil || film.Title != r.FilmTitle || film.Overview == "" {
		return r, errors.New("invalid details")
	}
	r.Checks = append(r.Checks, "fiche normalisée et identifiant opaque")
	if err = e.Disable(d.ID); err != nil {
		return r, err
	}
	if _, err = e.Call(ctx, d.ID, query); !errors.Is(err, ext.ErrUnavailable) {
		return r, errors.New("disabled extension still callable")
	}
	r.Checks = append(r.Checks, "désactivation")
	if err = e.Enable(d.ID); err != nil {
		return r, err
	}
	if _, err = e.Call(ctx, d.ID, query); err != nil {
		return r, err
	}
	r.Checks = append(r.Checks, "réactivation")
	r.AfterCalls = memory()
	if err = e.Unload(d.ID); err != nil {
		return r, err
	}
	if err = e.Enable(d.ID); !errors.Is(err, ext.ErrUnavailable) {
		return r, errors.New("unloaded extension still present")
	}
	r.Checks = append(r.Checks, "retrait")
	if err = e.Close(); err != nil {
		return r, err
	}
	r.AfterClose = memory()
	return r, nil
}
func main() {
	jsonOutput := flag.Bool("json", false, "Print measurements as JSON")
	flag.Parse()
	r, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Le laboratoire n’a pas terminé :", err)
		os.Exit(1)
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(r)
		return
	}
	fmt.Println("Mosaune — laboratoire des extensions (prototype)")
	fmt.Println("Exemple local, sans connexion au NAS ni modification de votre collection.")
	fmt.Println()
	for _, check := range r.Checks {
		fmt.Println("OK ·", check)
	}
	fmt.Printf("\nL’extension a trouvé : %s (film fictif).\n", r.FilmTitle)
	fmt.Printf("Préparation : %.0f ms. Appel suivant médian : %.1f ms.\n", r.LoadMS, r.MedianMS)
	fmt.Printf("Module : %.2f Mio. Mémoire linéaire autorisée : %d Mio maximum.\n", float64(r.CodeBytes)/(1<<20), r.MemoryLimitMiB)
	fmt.Println("Cette limite n’est pas la consommation totale : le moteur utilise aussi de la mémoire.")
	fmt.Println("Le pilote TMDB se teste dans les paramètres de Mosaune ; les fournisseurs installables restent une prochaine étape.")
}
