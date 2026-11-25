package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

type libraryGrant struct {
	View    bool `json:"view"`
	Play    bool `json:"play"`
	Edit    bool `json:"edit"`
	Request bool `json:"request"`
}
type account struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Admin     bool                    `json:"admin"`
	Disabled  bool                    `json:"disabled"`
	Kinds     []string                `json:"kinds"`
	Libraries map[string]libraryGrant `json:"libraries"`
	Theme     string                  `json:"theme"`
	Pending   bool                    `json:"pending"`
}
type identity struct {
	account
	Session string
	CSRF    string
}
type identityKey struct{}

func viewer(ctx context.Context) *identity { v, _ := ctx.Value(identityKey{}).(*identity); return v }
func userID(ctx context.Context) string {
	if u := viewer(ctx); u != nil {
		return u.ID
	}
	return ""
}
func canKind(ctx context.Context, kind string) bool {
	u := viewer(ctx)
	if u == nil || u.Admin {
		return true
	}
	for _, k := range u.Kinds {
		if k == kind {
			return true
		}
	}
	return false
}
func canLibrary(ctx context.Context, id, kind, action string) bool {
	u := viewer(ctx)
	if u == nil || u.Admin {
		return true
	}
	if !canKind(ctx, kind) {
		return false
	}
	g := u.Libraries[id]
	if !g.View {
		return false
	}
	switch action {
	case "view":
		return true
	case "play":
		return g.Play
	case "edit":
		return g.Edit
	case "request":
		return g.Request
	}
	return false
}

type authentication struct {
	mu       sync.Mutex
	code     string
	recovery string
	window   time.Time
	attempts int
}

func migrateAccounts(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v >= 9 {
		return nil
	}
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`CREATE TABLE accounts(id TEXT PRIMARY KEY,name TEXT NOT NULL UNIQUE COLLATE NOCASE,password TEXT NOT NULL,data TEXT NOT NULL);
 CREATE TABLE sessions(id TEXT PRIMARY KEY,user_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,csrf TEXT NOT NULL,expires INTEGER NOT NULL);
 CREATE TABLE invitations(id TEXT PRIMARY KEY,user_id TEXT NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,expires INTEGER NOT NULL);
 CREATE TABLE user_progress(user_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,media_id TEXT NOT NULL REFERENCES media(id) ON DELETE CASCADE,revision TEXT NOT NULL,position REAL NOT NULL DEFAULT 0,duration REAL NOT NULL DEFAULT 0,completed INTEGER NOT NULL DEFAULT 0,last_played INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(user_id,media_id));
 CREATE TABLE user_works(user_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,work_id TEXT NOT NULL REFERENCES works(id) ON DELETE CASCADE,favorite INTEGER NOT NULL DEFAULT 0,watchlist INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(user_id,work_id));
 ALTER TABLE playlists ADD COLUMN owner TEXT NOT NULL DEFAULT '';
 PRAGMA user_version=9;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func hashPassword(password string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)
	return "$argon2id$v=19$m=19456,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
}
func verifyPassword(encoded, password string) bool {
	p := strings.Split(encoded, "$")
	if len(p) != 6 || p[1] != "argon2id" || p[2] != "v=19" || p[3] != "m=19456,t=2,p=1" {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(p[4])
	if e != nil || len(salt) != 16 {
		return false
	}
	key, e := base64.RawStdEncoding.DecodeString(p[5])
	if e != nil || len(key) != 32 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)
	return subtle.ConstantTimeCompare(got, key) == 1
}
func passwordValid(p string) bool {
	return utf8.ValidString(p) && utf8.RuneCountInString(p) >= 12 && len(p) <= 1024
}
func validName(n string) bool {
	if len(n) < 1 || len(n) > 48 {
		return false
	}
	for _, r := range n {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}
func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (c *catalog) account(ctx context.Context, id string) (account, error) {
	var a account
	var raw string
	e := c.authRead.QueryRowContext(ctx, `SELECT data FROM accounts WHERE id=?`, id).Scan(&raw)
	if e == nil {
		e = json.Unmarshal([]byte(raw), &a)
	}
	return a, e
}
func (c *catalog) accountCount() int {
	var n int
	if c.authRead.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&n) != nil {
		return -1
	}
	return n
}
func (c *catalog) issueSession(w http.ResponseWriter, r *http.Request, a account) error {
	raw := rand.Text()
	csrf := rand.Text()
	expires := time.Now().Add(7 * 24 * time.Hour)
	_, e := c.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE expires < unixepoch()`)
	if e != nil {
		return e
	}
	// Limit stored sessions per account; oldest sessions are revoked automatically.
	_, e = c.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE user_id=? AND id IN (SELECT id FROM sessions WHERE user_id=? ORDER BY expires DESC LIMIT -1 OFFSET 9)`, a.ID, a.ID)
	if e != nil {
		return e
	}
	_, e = c.db.ExecContext(r.Context(), `INSERT INTO sessions VALUES(?,?,?,?)`, digest(raw), a.ID, csrf, expires.Unix())
	if e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "mosaune_session", Value: raw, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil, MaxAge: 7 * 24 * 3600})
	w.Header().Set("Clear-Site-Data", `"cache"`)
	sendJSON(w, 200, map[string]any{"user": a, "token": csrf})
	return nil
}
func (c *catalog) authenticate(r *http.Request) *identity {
	cookie, e := r.Cookie("mosaune_session")
	if e != nil || len(cookie.Value) > 128 {
		return nil
	}
	var id, csrf string
	e = c.authRead.QueryRowContext(r.Context(), `SELECT user_id,csrf FROM sessions WHERE id=? AND expires>unixepoch()`, digest(cookie.Value)).Scan(&id, &csrf)
	if e != nil {
		return nil
	}
	a, e := c.account(r.Context(), id)
	if e != nil || a.Disabled {
		return nil
	}
	return &identity{account: a, Session: digest(cookie.Value), CSRF: csrf}
}
func (c *catalog) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		u := c.authenticate(r)
		if u == nil {
			problem(w, 401, "Connectez-vous à Mosaune.")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Mosaune-Token")), []byte(u.CSRF)) != 1 {
			problem(w, 403, "Session modifiée. Rechargez la page.")
			return
		}
		// Personal media and images must not remain in a shared browser's HTTP cache.
		next.ServeHTTP(&privateResponse{ResponseWriter: w}, r.WithContext(context.WithValue(r.Context(), identityKey{}, u)))
	})
}

type privateResponse struct{ http.ResponseWriter }

func (w *privateResponse) WriteHeader(code int) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.ResponseWriter.WriteHeader(code)
}
func (w *privateResponse) Write(p []byte) (int, error) {
	w.Header().Set("Cache-Control", "private, no-store")
	return w.ResponseWriter.Write(p)
}

func (c *catalog) authRoutes(mux *http.ServeMux, token string) {
	mux.HandleFunc("GET /api/v1/auth/state", func(w http.ResponseWriter, r *http.Request) {
		c.auth.mu.Lock()
		recovery := c.auth.recovery != ""
		c.auth.mu.Unlock()
		u := c.authenticate(r)
		var a *account
		stateToken := token
		if u != nil {
			a = &u.account
			stateToken = u.CSRF
		}
		sendJSON(w, 200, map[string]any{"setup": c.accountCount() == 0, "recovery": recovery, "user": a, "token": stateToken})
	})
	mux.HandleFunc("POST /api/v1/auth/activate", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Code     string `json:"code"`
			Password string `json:"password"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		if !passwordValid(in.Password) || len(in.Code) > 128 {
			problem(w, 400, "Mot de passe : au moins 12 caractères.")
			return
		}
		if !c.auth.mu.TryLock() {
			problem(w, 429, "Réessayez dans un instant.")
			return
		}
		defer c.auth.mu.Unlock()
		if !c.auth.takeAttempt() {
			problem(w, 429, "Trop de tentatives. Réessayez dans cinq minutes.")
			return
		}
		var id string
		if c.db.QueryRowContext(r.Context(), `SELECT user_id FROM invitations WHERE id=? AND expires>unixepoch()`, digest(in.Code)).Scan(&id) != nil {
			problem(w, 403, "Invitation expirée ou déjà utilisée. Demandez un nouveau lien à l’administrateur.")
			return
		}
		a, e := c.account(r.Context(), id)
		if e != nil || a.Disabled || !a.Pending {
			problem(w, 403, "Invitation indisponible.")
			return
		}
		a.Pending = false
		raw, _ := json.Marshal(a)
		hash := hashPassword(in.Password)
		tx, e := c.db.BeginTx(r.Context(), nil)
		if e != nil {
			problem(w, 500, "Activation impossible.")
			return
		}
		defer tx.Rollback()
		result, e := tx.ExecContext(r.Context(), `DELETE FROM invitations WHERE id=? AND expires>unixepoch()`, digest(in.Code))
		if e != nil {
			problem(w, 500, "Activation impossible.")
			return
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			problem(w, 409, "Invitation déjà utilisée.")
			return
		}
		if _, e = tx.ExecContext(r.Context(), `UPDATE accounts SET password=?,data=? WHERE id=?`, hash, string(raw), id); e != nil {
			problem(w, 500, "Activation impossible.")
			return
		}
		if tx.Commit() != nil {
			problem(w, 500, "Activation impossible.")
			return
		}
		if c.issueSession(w, r, a) != nil {
			problem(w, 500, "Compte activé. Connectez-vous.")
		}
	})
	for _, action := range []string{"setup", "login", "recover"} {
		mux.HandleFunc("POST /api/v1/auth/"+action, func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Name     string `json:"name"`
				Password string `json:"password"`
				Code     string `json:"code"`
			}
			if !readJSON(w, r, &in) {
				return
			}
			if len(in.Password) > 1024 || len(in.Name) > 48 || len(in.Code) > 128 {
				problem(w, 400, "Identifiants invalides.")
				return
			}
			if !c.auth.mu.TryLock() {
				problem(w, 429, "Connexion en cours. Réessayez dans un instant.")
				return
			}
			defer c.auth.mu.Unlock()
			if time.Since(c.auth.window) > 5*time.Minute {
				c.auth.window = time.Now()
				c.auth.attempts = 0
			}
			if c.auth.attempts >= 10 {
				w.Header().Set("Retry-After", "300")
				problem(w, 429, "Trop de tentatives. Réessayez dans cinq minutes.")
				return
			}
			c.auth.attempts++
			var a account
			if action == "login" {
				var raw, hash string
				e := c.db.QueryRowContext(r.Context(), `SELECT data,password FROM accounts WHERE name=? COLLATE NOCASE`, in.Name).Scan(&raw, &hash)
				if e != nil {
					hash = "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
				}
				valid := verifyPassword(hash, in.Password)
				if e != nil || !valid || json.Unmarshal([]byte(raw), &a) != nil || a.Disabled || a.Pending {
					problem(w, 401, "Nom ou mot de passe incorrect.")
					return
				}
			} else {
				if c.auth.code == "" || subtle.ConstantTimeCompare([]byte(c.auth.code), []byte(in.Code)) != 1 {
					problem(w, 403, "Code d’installation ou de récupération incorrect. Consultez la fenêtre du serveur.")
					return
				}
				if !passwordValid(in.Password) || !validName(in.Name) {
					problem(w, 400, "Nom : lettres, chiffres, point, tiret. Mot de passe : au moins 12 caractères.")
					return
				}
				if action == "setup" {
					if c.accountCount() != 0 {
						problem(w, 409, "Le serveur est déjà configuré.")
						return
					}
					a = account{ID: rand.Text(), Name: in.Name, Admin: true, Kinds: []string{"films", "series", "music"}, Libraries: map[string]libraryGrant{}, Theme: "light"}
					if e := c.createFirstAccount(r.Context(), a, hashPassword(in.Password)); e != nil {
						problem(w, 500, "Compte non créé.")
						return
					}
				} else {
					if c.auth.recovery == "" || !strings.EqualFold(in.Name, c.auth.recovery) {
						problem(w, 403, "Récupération non autorisée pour ce compte.")
						return
					}
					var id string
					if c.db.QueryRowContext(r.Context(), `SELECT id FROM accounts WHERE name=? COLLATE NOCASE`, in.Name).Scan(&id) != nil {
						problem(w, 400, "Administrateur inconnu.")
						return
					}
					var e error
					a, e = c.account(r.Context(), id)
					if e != nil || !a.Admin {
						problem(w, 403, "Administrateur requis.")
						return
					}
					if e = c.replacePassword(r.Context(), id, hashPassword(in.Password)); e != nil {
						problem(w, 500, "Mot de passe non modifié.")
						return
					}
				}
				c.auth.code = ""
				c.auth.recovery = ""
			}
			// Successful logins do not reset the global budget, so another account cannot bypass it.
			if e := c.issueSession(w, r, a); e != nil {
				problem(w, 500, "Session non créée.")
			}
		})
	}
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		u := c.authenticate(r)
		if u == nil {
			w.WriteHeader(204)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Mosaune-Token")), []byte(u.CSRF)) != 1 {
			problem(w, 403, "Session modifiée.")
			return
		}
		if _, e := c.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE id=?`, u.Session); e != nil {
			problem(w, 500, "Déconnexion impossible.")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "mosaune_session", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
		w.WriteHeader(204)
	})
}

func (c *catalog) createFirstAccount(ctx context.Context, a account, hash string) error {
	tx, e := c.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var n int
	if e = tx.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&n); e != nil {
		return e
	}
	if n != 0 {
		return errors.New("already configured")
	}
	raw, _ := json.Marshal(a)
	if _, e = tx.Exec(`INSERT INTO accounts VALUES(?,?,?,?)`, a.ID, a.Name, hash, string(raw)); e != nil {
		return e
	}
	if _, e = tx.Exec(`INSERT INTO user_progress SELECT ?,id,revision,position,duration,completed,last_played FROM media WHERE position>0 OR completed=1 OR last_played>0`, a.ID); e != nil {
		return e
	}
	if _, e = tx.Exec(`INSERT INTO user_works SELECT ?,id,favorite,watchlist FROM works WHERE favorite=1 OR watchlist=1`, a.ID); e != nil {
		return e
	}
	if _, e = tx.Exec(`UPDATE playlists SET owner=? WHERE owner=''`, a.ID); e != nil {
		return e
	}
	if _, e = tx.Exec(`UPDATE media SET position=0,duration=0,completed=0,last_played=0; UPDATE works SET favorite=0,watchlist=0;`); e != nil {
		return e
	}
	return tx.Commit()
}
func (c *catalog) replacePassword(ctx context.Context, id, hash string) error {
	tx, e := c.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `UPDATE accounts SET password=? WHERE id=?`, hash, id); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=?`, id); e != nil {
		return e
	}
	return tx.Commit()
}
