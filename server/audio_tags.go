package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	tags "mosaune/extensionkit/capabilities/audiotags/v1"
	"mosaune/extensionkit/protocol"
	runtime "mosaune/extensionkit/runtime"
	"time"
)

type audioTagReader struct {
	engine *runtime.Engine
	id     string
}

func loadAudioTags(ctx context.Context, d protocol.Descriptor, code []byte) (*audioTagReader, error) {
	if len(d.Capabilities) != 1 || d.Capabilities[0] != tags.Capability || len(d.Dependencies) != 0 || len(d.Services) != 1 || d.Services[0].Name != tags.Service || len(d.Services[0].Operations) != 1 || d.Services[0].Operations[0] != "range" {
		return nil, errors.New("Contrat d’informations musicales incompatible.")
	}
	limits := runtime.DefaultLimits()
	limits.Modules = 1
	limits.Interpreter = true
	limits.MemoryMiB = 128
	limits.OutputBytes = 12 << 20
	limits.Timeout = 10 * time.Second
	engine, err := runtime.New(ctx, limits)
	if err != nil {
		return nil, err
	}
	if err = engine.Load(ctx, d, code); err == nil {
		err = engine.Enable(d.ID)
	}
	if err != nil {
		engine.Close()
		return nil, err
	}
	return &audioTagReader{engine: engine, id: d.ID}, nil
}

// A callback captures one already-open file. There is no path or URL parameter.
func audioFileService(file io.ReaderAt, size int64) runtime.Service {
	remaining := tags.MaxRead
	return func(ctx context.Context, op string, raw json.RawMessage) (json.RawMessage, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var q tags.Range
		if op != "range" || protocol.DecodeOne(raw, &q) != nil || q.Offset < 0 || q.Offset > size || q.Length < 1 || q.Length > tags.BlockSize || int64(q.Length) > size-q.Offset || q.Length > remaining {
			return nil, errors.New("invalid file range")
		}
		remaining -= q.Length
		b := make([]byte, q.Length)
		n, err := file.ReadAt(b, q.Offset)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n != len(b) {
			return nil, io.ErrUnexpectedEOF
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		return json.Marshal(tags.Block{Data: b})
	}
}
func (r *audioTagReader) extract(ctx context.Context, file io.ReaderAt, size int64, md Metadata) (Metadata, []byte, error) {
	raw, _ := json.Marshal(tags.Query{Size: size})
	result, err := r.engine.CallWithServiceBudget(ctx, r.id, protocol.Request{Protocol: protocol.Version, ID: "audio-tags", Capability: tags.Capability, Operation: "extract", Payload: raw}, runtime.ServiceGrants{tags.Service: audioFileService(file, size)}, tags.MaxCalls)
	if err != nil {
		return md, nil, err
	}
	var out tags.Result
	if result.Response.Error != nil || protocol.DecodeOne(result.Response.Result, &out) != nil || out.Validate() != nil {
		return md, nil, errors.New("Réponse de l’extension musicale invalide ; catalogue conservé.")
	}
	if err = ctx.Err(); err != nil {
		return md, nil, err
	}
	if !out.Matched {
		return md, nil, nil
	}
	if md.Origins == nil {
		md.Origins = map[string]string{}
	}
	assign := func(key string, dst *string, value string) {
		if value != "" {
			*dst = value
			md.Origins[key] = "tags"
		}
	}
	assign("title", &md.Title, out.Title)
	assign("artist", &md.Artist, out.Artist)
	assign("album", &md.Album, out.Album)
	assign("albumArtist", &md.AlbumArtist, out.AlbumArtist)
	md.Year = out.Year
	md.Track = out.Track
	md.Disc = out.Disc
	md.Provider = "tags"
	if out.Genre != "" {
		md.Genres = []string{out.Genre}
	}
	return md, out.Picture, nil
}
