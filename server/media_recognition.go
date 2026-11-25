package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	recognition "mosaune/extensionkit/capabilities/recognition/v1"
	"mosaune/extensionkit/protocol"
	runtime "mosaune/extensionkit/runtime"
)

// A reader owns one short-lived engine for an entire scan. There is no VM per
// file and no persistent recognition engine consuming memory while idle.
type mediaRecognizer struct {
	engine  *runtime.Engine
	id      string
	formats recognition.Formats
}

func (c *catalog) openRecognizer(ctx context.Context, kind string) (context.Context, *mediaRecognizer, func(), error) {
	call, done, descriptor, code, err := c.beginRecognition(ctx, kind)
	if err != nil {
		return call, nil, func() {}, err
	}
	reader, err := loadMediaRecognizer(call, descriptor, code)
	if err != nil {
		done()
		return call, nil, func() {}, err
	}
	return call, reader, func() { reader.close(); done() }, nil
}
func loadMediaRecognizer(ctx context.Context, descriptor protocol.Descriptor, code []byte) (*mediaRecognizer, error) {
	// No filesystem, environment, credentials or services are exposed. Future
	// installable recognizers must pass the same contract and package trust checks.
	if len(descriptor.Capabilities) != 1 || descriptor.Capabilities[0] != recognition.Capability || len(descriptor.Services) != 0 || len(descriptor.Dependencies) != 0 {
		return nil, errors.New("Contrat de reconnaissance incompatible.")
	}
	limits := runtime.DefaultLimits()
	limits.Modules = 1
	limits.Interpreter = true
	engine, err := runtime.New(ctx, limits)
	if err != nil {
		return nil, err
	}
	reader := &mediaRecognizer{engine: engine, id: descriptor.ID}
	failed := true
	defer func() {
		if failed {
			reader.close()
		}
	}()
	if err = engine.Load(ctx, descriptor, code); err != nil {
		return nil, fmt.Errorf("Module de reconnaissance indisponible : %w", err)
	}
	if err = engine.Enable(descriptor.ID); err != nil {
		return nil, err
	}
	if err = reader.call(ctx, "formats", struct{}{}, &reader.formats); err != nil {
		return nil, err
	}
	if err = reader.formats.Validate(); err != nil {
		return nil, errors.New("Formats déclarés par le module invalides.")
	}
	failed = false
	return reader, nil
}
func (r *mediaRecognizer) close() { _ = r.engine.Close() }
func (r *mediaRecognizer) call(ctx context.Context, operation string, payload, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	result, err := r.engine.Call(ctx, r.id, protocol.Request{Protocol: protocol.Version, ID: "recognition-" + operation, Capability: recognition.Capability, Operation: operation, Payload: data})
	if err != nil {
		return fmt.Errorf("Reconnaissance interrompue : %w", err)
	}
	if result.Response.Error != nil || protocol.DecodeOne(result.Response.Result, out) != nil {
		return errors.New("Réponse du module de reconnaissance invalide.")
	}
	return ctx.Err()
}
func (r *mediaRecognizer) recognize(ctx context.Context, paths []string) ([]recognition.Item, error) {
	q := recognition.Query{Paths: paths}
	if err := q.Validate(); err != nil {
		return nil, errors.New("Nom de fichier incompatible avec le contrat de reconnaissance.")
	}
	var result recognition.Result
	if err := r.call(ctx, "recognize", q, &result); err != nil {
		return nil, err
	}
	if err := result.Validate(q, r.formats); err != nil {
		return nil, errors.New("Métadonnées du module de reconnaissance invalides.")
	}
	return result.Items, nil
}
func filenameMetadata(item recognition.Item) Metadata {
	return Metadata{Title: item.Title, Year: item.Year, Series: item.Series, Season: item.Season, Episode: item.Episode, EpisodeEnd: item.EpisodeEnd, Provider: "filename", MatchState: "local", Origins: map[string]string{"title": "filename"}}
}
