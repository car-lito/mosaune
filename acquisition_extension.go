package main

import (
	"context"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/protocol"
	"strings"
	"sync"
)

type acquisitionSessionKey struct{}
type acquisitionSession struct {
	owner       *catalog
	descriptor  protocol.Descriptor
	code        []byte
	checksum    string
	transportMu sync.Mutex
	transports  map[string]a.Transport
	formats     map[string]a.WireFormat
}

func acquisitionImport(p packageformat.Executable) (string, error) {
	m := p.Manifest
	d := m.Module
	if strings.HasPrefix(d.ID, "mosaune.") || packageformat.Compare(m.MinCore, "0.0.59") < 0 || len(m.Contributions) != 1 || m.Contributions[0].Role != a.Capability || m.Contributions[0].Target != "video" || len(d.Capabilities) != 1 || d.Capabilities[0] != a.Capability || len(d.Services) != 0 || len(d.Dependencies) != 0 {
		return "", errors.New("Composant de demandes incompatible.")
	}
	return "acquisition", nil
}

func (client arrClient) openSession(ctx context.Context) (context.Context, func(), error) {
	if session, ok := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession); ok && session.owner == client.config.owner {
		return ctx, func() {}, context.Cause(ctx)
	}
	var descriptor protocol.Descriptor
	var code []byte
	var err error
	done := func() {}
	if client.config.owner == nil {
		// In-memory transport fixtures have no catalogue. Production settings
		// always carry the catalogue that loaded or validated them.
		descriptor, code, err = modules.Lookup("acquisition")
	} else {
		ctx, done, descriptor, code, err = client.config.owner.beginRecognition(ctx, "acquisition")
	}
	if err != nil {
		return ctx, func() {}, errors.New("Le composant de demandes est désactivé ou absent. Activez-le dans les composants de Films ou Séries.")
	}
	session := &acquisitionSession{owner: client.config.owner, descriptor: descriptor, code: code, checksum: descriptor.ID + ":" + recognitionHash(code)}
	return context.WithValue(ctx, acquisitionSessionKey{}, session), done, nil
}

func (client arrClient) submissionPlan(ctx context.Context, obj arrObject, season int) (a.Plan, error) {
	var plan a.Plan
	session, ok := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession)
	if !ok || session.owner != client.config.owner {
		return plan, errors.New("Session de demandes absente.")
	}
	if client.config.planChecksum != "" && client.config.planChecksum != session.checksum {
		return plan, errors.New("Le composant de demandes a changé. Vérifiez de nouveau la demande.")
	}
	query := a.Query{Connector: client.config.ID, RemoteID: number(obj["id"]), HasFile: truth(obj["hasFile"]), Season: season, SeriesType: word(obj["seriesType"])}
	if query.Connector == "sonarr" {
		query.Seasons = selectedSeasons(obj, -1)
	}
	if err := query.Validate(); err != nil {
		return plan, err
	}
	reader, err := loadFactProjector(ctx, session.descriptor, session.code, a.Capability)
	if err != nil {
		return plan, err
	}
	defer reader.engine.Close()
	if err = reader.call(ctx, "plan", query, &plan); err != nil {
		return plan, err
	}
	if err = plan.Validate(query); err != nil {
		return plan, errors.New("Le composant propose une demande hors du périmètre autorisé.")
	}
	return plan, nil
}
