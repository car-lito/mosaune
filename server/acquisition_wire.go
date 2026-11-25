package main

import (
	"context"
	"errors"
	a "mosaune/extensionkit/capabilities/acquisition/v1"
)

func readAcquisitionWire(ctx context.Context, r *cataloguer, connector string) (a.WireFormat, error) {
	var format a.WireFormat
	if err := r.call(ctx, "describe-wire", a.TransportQuery{Connector: connector}, &format); err != nil {
		return format, errors.New("Mettez à jour le composant de demandes pour prendre en charge les formats du service.")
	}
	if err := format.Validate(connector); err != nil {
		return format, errors.New("Formats du connecteur incompatibles.")
	}
	return format, nil
}

func (client arrClient) wireProfile(ctx context.Context) (a.WireFormat, error) {
	s, ok := ctx.Value(acquisitionSessionKey{}).(*acquisitionSession)
	if !ok || s.owner != client.config.owner {
		return a.WireFormat{}, errors.New("Session de demandes absente.")
	}
	s.transportMu.Lock()
	defer s.transportMu.Unlock()
	format, ok := s.formats[client.config.ID]
	if !ok {
		return format, errors.New("Formats du connecteur indisponibles.")
	}
	return format, nil
}
