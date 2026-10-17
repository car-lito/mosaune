package main

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (a *authentication) takeAttempt() bool {
	if time.Since(a.window) > 5*time.Minute {
		a.window = time.Now()
		a.attempts = 0
	}
	if a.attempts >= 10 {
		return false
	}
	a.attempts++
	return true
}
func (c *catalog) accountRoutes(mux *secureMux) {
	mux.HandleFunc("GET /api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		rows, e := c.db.QueryContext(r.Context(), `SELECT data FROM accounts ORDER BY name COLLATE NOCASE`)
		if e != nil {
			problem(w, 500, "Comptes indisponibles.")
			return
		}
		defer rows.Close()
		list := []account{}
		for rows.Next() {
			var raw string
			var a account
			if rows.Scan(&raw) != nil || json.Unmarshal([]byte(raw), &a) != nil {
				problem(w, 500, "Compte illisible.")
				return
			}
			list = append(list, a)
		}
		if rows.Err() != nil {
			problem(w, 500, "Comptes indisponibles.")
			return
		}
		sendJSON(w, 200, list)
	})
	for _, method := range []string{"POST /api/v1/accounts", "PUT /api/v1/accounts/{id}"} {
		mux.HandleFunc(method, func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Account  account `json:"account"`
				Password string  `json:"password"`
			}
			if !readJSON(w, r, &in) {
				return
			}
			a := in.Account
			a.Name = strings.TrimSpace(a.Name)
			if !validName(a.Name) || in.Password != "" && !passwordValid(in.Password) {
				problem(w, 400, "Nom invalide ou mot de passe trop court (12 caractères minimum).")
				return
			}
			if a.Theme != "light" && a.Theme != "dark" && a.Theme != "system" {
				a.Theme = "light"
			}
			if a.Kinds == nil {
				a.Kinds = []string{}
			}
			if a.Libraries == nil {
				a.Libraries = map[string]libraryGrant{}
			}
			for _, k := range a.Kinds {
				if !contentKind(k) {
					problem(w, 400, "Type de média inconnu.")
					return
				}
			}
			for id := range a.Libraries {
				var n int
				if c.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM libraries WHERE id=?`, id).Scan(&n) != nil || n != 1 {
					problem(w, 400, "Bibliothèque inconnue.")
					return
				}
			}
			if !c.auth.mu.TryLock() {
				problem(w, 409, "Un changement de compte est en cours.")
				return
			}
			defer c.auth.mu.Unlock()
			var hash string
			invitation := ""
			if r.Method == "POST" {
				if c.accountCount() >= 100 {
					problem(w, 400, "Limite de 100 comptes atteinte.")
					return
				}
				a.ID = rand.Text()
				a.Pending = in.Password == ""
				if a.Pending {
					invitation = rand.Text()
				}
			} else {
				a.ID = r.PathValue("id")
				old, e := c.account(r.Context(), a.ID)
				if e != nil {
					problem(w, 404, "Compte inconnu.")
					return
				}
				a.Theme = old.Theme
				a.Pending = old.Pending
				if a.ID == userID(r.Context()) && (!a.Admin || a.Disabled) {
					problem(w, 400, "Vous ne pouvez pas retirer vos propres droits administrateur.")
					return
				}
			}
			if in.Password != "" {
				hash = hashPassword(in.Password)
				a.Pending = false
			}
			raw, _ := json.Marshal(a)
			tx, e := c.db.BeginTx(r.Context(), nil)
			if e != nil {
				problem(w, 500, "Enregistrement impossible.")
				return
			}
			defer tx.Rollback()
			if r.Method == "PUT" && (!a.Admin || a.Disabled) {
				var remaining int
				if e = tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM accounts WHERE id<>? AND json_extract(data,'$.admin')=1 AND json_extract(data,'$.disabled')=0 AND COALESCE(json_extract(data,'$.pending'),0)=0`, a.ID).Scan(&remaining); e != nil || remaining == 0 {
					problem(w, 409, "Conservez au moins un administrateur actif.")
					return
				}
			}
			if r.Method == "POST" {
				_, e = tx.ExecContext(r.Context(), `INSERT INTO accounts VALUES(?,?,?,?)`, a.ID, a.Name, hash, string(raw))
			} else {
				_, e = tx.ExecContext(r.Context(), `UPDATE accounts SET name=?,data=? WHERE id=?`, a.Name, string(raw), a.ID)
				if e == nil && hash != "" {
					_, e = tx.ExecContext(r.Context(), `UPDATE accounts SET password=? WHERE id=?`, hash, a.ID)
				}
				if e == nil {
					_, e = tx.ExecContext(r.Context(), `DELETE FROM sessions WHERE user_id=?`, a.ID)
				}
			}
			if e == nil && invitation != "" {
				_, e = tx.ExecContext(r.Context(), `INSERT INTO invitations VALUES(?,?,?)`, digest(invitation), a.ID, time.Now().Add(7*24*time.Hour).Unix())
			}
			if e == nil && hash != "" {
				_, e = tx.ExecContext(r.Context(), `DELETE FROM invitations WHERE user_id=?`, a.ID)
			}
			if e != nil {
				problem(w, 409, "Enregistrement impossible : ce nom est peut-être déjà utilisé.")
				return
			}
			if tx.Commit() != nil {
				problem(w, 500, "Enregistrement impossible.")
				return
			}
			sendJSON(w, 200, map[string]any{"user": a, "invitation": invitation})
		})
	}
	mux.HandleFunc("POST /api/v1/accounts/{id}/invitation", func(w http.ResponseWriter, r *http.Request) {
		c.auth.mu.Lock()
		defer c.auth.mu.Unlock()
		a, e := c.account(r.Context(), r.PathValue("id"))
		if e != nil || !a.Pending || a.Disabled {
			problem(w, 400, "Ce compte est déjà activé ou désactivé.")
			return
		}
		code := rand.Text()
		if _, e = c.db.ExecContext(r.Context(), `INSERT INTO invitations VALUES(?,?,?) ON CONFLICT(user_id) DO UPDATE SET id=excluded.id,expires=excluded.expires`, digest(code), a.ID, time.Now().Add(7*24*time.Hour).Unix()); e != nil {
			problem(w, 500, "Invitation non créée.")
			return
		}
		sendJSON(w, 200, map[string]string{"invitation": code})
	})
	mux.HandleFunc("PUT /api/v1/account/preferences", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Theme string `json:"theme"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		if in.Theme != "light" && in.Theme != "dark" && in.Theme != "system" {
			problem(w, 400, "Thème inconnu.")
			return
		}
		c.auth.mu.Lock()
		defer c.auth.mu.Unlock()
		a, e := c.account(r.Context(), userID(r.Context()))
		if e != nil {
			problem(w, 500, "Compte indisponible.")
			return
		}
		a.Theme = in.Theme
		raw, _ := json.Marshal(a)
		if _, e = c.db.ExecContext(r.Context(), `UPDATE accounts SET data=? WHERE id=?`, string(raw), a.ID); e != nil {
			problem(w, 500, "Préférence non enregistrée.")
			return
		}
		sendJSON(w, 200, a)
	})
	mux.HandleFunc("PUT /api/v1/account/password", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Current  string `json:"current"`
			Password string `json:"password"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		if !passwordValid(in.Password) || len(in.Current) > 1024 {
			problem(w, 400, "Le nouveau mot de passe doit contenir au moins 12 caractères.")
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
		var hash string
		id := userID(r.Context())
		if c.db.QueryRowContext(r.Context(), `SELECT password FROM accounts WHERE id=?`, id).Scan(&hash) != nil || !verifyPassword(hash, in.Current) {
			problem(w, 403, "Mot de passe actuel incorrect.")
			return
		}
		if c.replacePassword(r.Context(), id, hashPassword(in.Password)) != nil {
			problem(w, 500, "Mot de passe non modifié.")
			return
		}
		w.WriteHeader(204)
	})
}
