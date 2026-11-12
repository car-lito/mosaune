package main

import (
	"context"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"strings"
)

func readAcquisitionPreview(ctx context.Context, r *cataloguer, q a.PreviewQuery) (a.PreviewPlan, error) {
	var p a.PreviewPlan
	if err := r.call(ctx, "preparation-plan", q, &p); err != nil {
		return p, errors.New("Mettez à jour le composant de demandes pour préparer cet aperçu.")
	}
	return p, p.Validate(q)
}
func (client arrClient) previewPlan(ctx context.Context, q a.PreviewQuery) (a.PreviewPlan, error) {
	s, ok := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession)
	if !ok || s.owner != client.config.owner {
		return a.PreviewPlan{}, errors.New("Session de demandes absente.")
	}
	r, err := loadFactProjector(ctx, s.descriptor, s.code, a.Capability)
	if err != nil {
		return a.PreviewPlan{}, err
	}
	defer r.engine.Close()
	return readAcquisitionPreview(ctx, r, q)
}

// The plan has already been checked against the user's requested scope. Every
// operation here is read-only, using the same cancellable installed session.
func (client arrClient) previewResource(ctx context.Context, q a.PreviewQuery, p a.PreviewPlan, external int) (arrObject, arrOptions, bool, error) {
	var obj arrObject
	var opt arrOptions
	if err := p.Validate(q); err != nil {
		return nil, opt, false, err
	}
	if client.config.ID != p.Connector || external <= 0 {
		return nil, opt, false, errors.New("Connecteur ou identifiant de préparation invalide.")
	}
	for _, operation := range p.Reads {
		switch operation {
		case "status":
			var status struct {
				AppName string `json:"appName"`
			}
			if err := client.exchange(ctx, operation, 0, 0, nil, &status); err != nil {
				return nil, opt, false, err
			}
			if !strings.EqualFold(status.AppName, p.Connector) {
				return nil, opt, false, errors.New("Ce service ne correspond pas au connecteur choisi.")
			}
		case "profiles":
			if err := client.exchange(ctx, operation, 0, 0, nil, &opt.Profiles); err != nil {
				return nil, opt, false, err
			}
		case "roots":
			if err := client.exchange(ctx, operation, 0, 0, nil, &opt.Roots); err != nil {
				return nil, opt, false, err
			}
		case "find":
			var err error
			obj, err = client.find(ctx, external)
			if err != nil {
				return nil, opt, false, err
			}
		}
		if len(opt.Profiles) > 5000 || len(opt.Roots) > 5000 {
			return nil, opt, false, errors.New("Trop de destinations dans la réponse du service.")
		}
	}
	if obj != nil {
		return obj, opt, true, nil
	}
	if p.Missing == "reject" {
		return nil, opt, false, errors.New(p.MissingMessage)
	}
	if !validChoices(client.config, opt) {
		return nil, opt, false, errors.New("Le profil ou le dossier n’existe plus dans le service. Actualisez les réglages.")
	}
	var err error
	obj, err = client.lookup(ctx, external)
	return obj, opt, false, err
}
