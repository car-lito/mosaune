package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func readAcquisitionTransport(ctx context.Context, r *cataloguer, connector string) (a.Transport, error) {
	var p a.Transport
	if err := r.call(ctx, "describe-transport", a.TransportQuery{Connector: connector}, &p); err != nil {
		return p, errors.New("Le composant de demandes doit être mis à jour pour fournir ses opérations réseau.")
	}
	if err := p.Validate(connector); err != nil {
		return p, errors.New("Opérations réseau du composant de demandes incompatibles.")
	}
	return p, nil
}
func (client arrClient) transportProfile(ctx context.Context) (a.Transport, error) {
	s, ok := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession)
	if !ok || s.owner != client.config.owner {
		return a.Transport{}, errors.New("Session de demandes absente.")
	}
	s.transportMu.Lock()
	defer s.transportMu.Unlock()
	if p, ok := s.transports[client.config.ID]; ok {
		return p, nil
	}
	r, err := loadFactProjector(ctx, s.descriptor, s.code, a.Capability)
	if err != nil {
		return a.Transport{}, err
	}
	defer r.engine.Close()
	p, err := readAcquisitionTransport(ctx, r, client.config.ID)
	if err == nil {
		if s.transports == nil {
			s.transports = map[string]a.Transport{}
		}
		s.transports[client.config.ID] = p
	}
	return p, err
}

func (client arrClient) exchange(ctx context.Context, operation string, id, page int, body, out any) error {
	ctx, done, err := client.openSession(ctx)
	if err != nil {
		return err
	}
	defer done()
	p, err := client.transportProfile(ctx)
	if err != nil {
		return err
	}
	method, relative, err := p.Resolve(client.config.ID, operation, id, page)
	if err != nil {
		return errors.New("Opération du connecteur invalide.")
	}
	// The extension supplies no authority, credentials, headers or arbitrary
	// request parameters. Preserve the administrator's configured subpath too.
	base, err := url.Parse(client.config.URL)
	if err != nil || base.Hostname() == "" || (base.Scheme != "http" && base.Scheme != "https") || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return errors.New("Adresse du service invalide.")
	}
	target := strings.TrimRight(base.String(), "/") + "/" + relative
	var data []byte
	if body != nil {
		if method == "GET" {
			return errors.New("Une lecture ne peut pas transmettre de modification.")
		}
		data, err = json.Marshal(body)
		if err != nil || len(data) > 8<<20 {
			return errors.New("Demande invalide ou trop volumineuse.")
		}
	} else if method != "GET" {
		return errors.New("Modification sans contenu refusée.")
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(data))
	if err != nil {
		return errors.New("Adresse du service invalide.")
	}
	req.Header.Set("X-Api-Key", client.config.Key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	transport := http.Client{Timeout: time.Duration(p.TimeoutSeconds) * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirection refusée") }}
	resp, err := transport.Do(req)
	if err != nil {
		return errors.New("Service inaccessible ou réponse trop lente. Vérifiez son adresse et son accès réseau.")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return errors.New("Le service refuse la clé API.")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Le service a refusé l’opération (HTTP %d). Consultez ses événements.", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, p.MaxBytes+1))
	if err != nil || int64(len(b)) > p.MaxBytes {
		return errors.New("Réponse du service incomplète ou trop volumineuse.")
	}
	if out != nil && json.Unmarshal(b, out) != nil {
		return errors.New("Réponse inattendue : vérifiez l’URL du service.")
	}
	return ctx.Err()
}
