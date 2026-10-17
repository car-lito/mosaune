package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// The UI is compiled once and included in the server executable.
//
//go:embed all:web/dist
var frontend embed.FS

const version = "0.0.23"

type module struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Kinds []string `json:"kinds"`
}

func enabledModules(value string) ([]module, error) {
	available := map[string]module{
		"video": {ID: "video", Name: "Films et séries", Kinds: []string{"films", "series"}},
		"music": {ID: "music", Name: "Musique", Kinds: []string{"music"}},
	}
	result := []module{}
	seen := map[string]bool{}
	for _, entry := range strings.Split(value, ",") {
		id := strings.TrimSpace(entry)
		if id == "" || seen[id] {
			continue
		}
		m, exists := available[id]
		if !exists {
			return nil, fmt.Errorf("module inconnu : %s (video, music)", id)
		}
		result = append(result, m)
		seen[id] = true
	}
	return result, nil
}

func newHandler(ui fs.FS, modules []module, catalog *catalog) http.Handler {
	return newNetworkHandler(ui, modules, catalog, networkConfig{})
}

func newNetworkHandler(ui fs.FS, modules []module, catalog *catalog, network networkConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": version})
	})
	secured := &secureMux{ServeMux: mux, c: catalog}
	token := libraryRoutes(secured, catalog, modules)
	catalog.accountRoutes(secured)
	catalog.extensionRoutes(secured)
	catalog.authRoutes(mux, token)
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	mux.Handle("GET /", http.FileServer(http.FS(ui)))
	guarded := network.access(catalog.authenticated(mux), token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: blob:; media-src 'self' blob:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		guarded.ServeHTTP(w, r)
	})
}

func run() error {
	port := flag.Int("port", 47831, "port d’écoute")
	listen := flag.String("listen", envDefault("MOSAUNE_LISTEN", "127.0.0.1"), "adresse IP d’écoute")
	publicURL := flag.String("public-url", os.Getenv("MOSAUNE_PUBLIC_URL"), "origine HTTPS exacte utilisée par les clients")
	cert := flag.String("tls-cert", os.Getenv("MOSAUNE_TLS_CERT"), "certificat HTTPS PEM avec sa chaîne")
	key := flag.String("tls-key", os.Getenv("MOSAUNE_TLS_KEY"), "clé privée HTTPS PEM")
	ca := flag.String("tls-ca", os.Getenv("MOSAUNE_TLS_CA"), "autorité locale optionnelle pour le contrôle de santé")
	healthcheck := flag.Bool("healthcheck", false, "vérifier le serveur déjà démarré, puis quitter")
	moduleList := flag.String("modules", "video,music", "modules actifs : video,music")
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	dataDir := flag.String("data-dir", envDefault("MOSAUNE_DATA_DIR", filepath.Join(filepath.Dir(executable), "data")), "dossier du catalogue SQLite")
	recoverName := flag.String("recover-admin", "", "récupérer le compte administrateur indiqué")
	extensionSafeMode := flag.Bool("safe-mode", false, "démarrer sans appliquer les extensions externes")
	flag.Parse()
	network := networkConfig{Listen: *listen, Port: *port, PublicURL: *publicURL, Cert: *cert, Key: *key, CA: *ca}
	if err := network.validate(); err != nil {
		return err
	}
	if *healthcheck {
		return network.healthcheck()
	}
	tlsConfig, err := network.loadTLS()
	if err != nil {
		return err
	}
	modules, err := enabledModules(*moduleList)
	if err != nil {
		return err
	}
	ui, err := fs.Sub(frontend, "web/dist")
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", network.address())
	if err != nil {
		return fmt.Errorf("impossible de démarrer le serveur : %w", err)
	}
	defer listener.Close()
	catalog, err := openCatalog(*dataDir)
	if err != nil {
		listener.Close()
		return fmt.Errorf("catalogue : %w", err)
	}
	defer catalog.close()
	catalog.extensionSafeMode = *extensionSafeMode
	if *recoverName != "" {
		var id string
		if catalog.db.QueryRow(`SELECT id FROM accounts WHERE name=? COLLATE NOCASE`, *recoverName).Scan(&id) != nil {
			return errors.New("administrateur inconnu")
		}
		a, e := catalog.account(context.Background(), id)
		if e != nil || !a.Admin {
			return errors.New("administrateur requis")
		}
		catalog.auth.recovery = a.Name
		catalog.auth.code = rand.Text()
	}
	if catalog.auth.code != "" {
		log.Printf("Code unique de configuration / récupération : %s", catalog.auth.code)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Handler: newNetworkHandler(ui, modules, catalog, network), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		allowed := func(kind string) bool {
			for _, m := range modules {
				for _, k := range m.Kinds {
					if k == kind {
						return true
					}
				}
			}
			return false
		}
		metadataDone := make(chan struct{})
		go func() { defer close(metadataDone); catalog.metadataWorker(ctx, allowed) }()
		catalog.acquisitionWorker(ctx, allowed)
		<-metadataDone
	}()
	defer func() { stop(); <-workerDone }()
	log.Printf("Mosaune %s — %s — Ctrl+C pour arrêter", version, network.origin())
	return serveHTTP(ctx, server, listener, tlsConfig)
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
