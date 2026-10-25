// This executable sees only bounded ranges of the single file selected by the host.
package main

import (
	"encoding/json"
	"errors"
	"github.com/dhowden/tag"
	"io"
	tags "mosaune/extensionkit/capabilities/audiotags/v1"
	"mosaune/extensionkit/protocol"
	"mosaune/extensionkit/sdk"
	"os"
	"strings"
)

type reader struct {
	size, pos, start      int64
	block                 []byte
	remaining, calls, ops int
	serviceErr            error
}

func (r *reader) Read(b []byte) (int, error) {
	r.ops--
	if r.ops <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	if len(b) == 0 {
		return 0, nil
	}
	if r.pos >= r.size {
		return 0, io.EOF
	}
	if r.block == nil || r.pos < r.start || r.pos >= r.start+int64(len(r.block)) {
		if r.calls <= 0 || r.remaining <= 0 {
			return 0, io.ErrUnexpectedEOF
		}
		r.calls--
		r.start = r.pos / tags.BlockSize * tags.BlockSize
		length := int(min(int64(tags.BlockSize), r.size-r.start, int64(r.remaining)))
		raw, err := sdk.ReadServiceSized(tags.Service, "range", tags.Range{Offset: r.start, Length: length}, 96<<10)
		var block tags.Block
		if err == nil {
			err = json.Unmarshal(raw, &block)
		}
		if err != nil || len(block.Data) != length {
			if err == nil {
				err = errors.New("short file range")
			}
			r.serviceErr = err
			return 0, err
		}
		r.block = block.Data
		r.remaining -= len(r.block)
	}
	n := copy(b, r.block[r.pos-r.start:])
	r.pos += int64(n)
	return n, nil
}
func (r *reader) Seek(off int64, whence int) (int64, error) {
	r.ops--
	if r.ops <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	base := int64(0)
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = r.pos
	case io.SeekEnd:
		base = r.size
	default:
		return 0, errors.New("invalid seek")
	}
	if off < -base || off > r.size-base {
		return 0, errors.New("seek outside file")
	}
	r.pos = base + off
	return r.pos, nil
}
func clipped(s string) string { v := []rune(strings.TrimSpace(s)); return string(v[:min(len(v), 500)]) }
func extract(r io.ReadSeeker) (out tags.Result) {
	defer func() {
		if recover() != nil {
			out = tags.Result{}
		}
	}()
	t, err := tag.ReadFrom(r)
	if err != nil {
		return out
	}
	out.Matched = true
	out.Title = clipped(t.Title())
	out.Artist = clipped(t.Artist())
	out.Album = clipped(t.Album())
	out.AlbumArtist = clipped(t.AlbumArtist())
	out.Genre = clipped(t.Genre())
	out.Year = max(0, min(t.Year(), 9999))
	n, _ := t.Track()
	out.Track = max(0, min(n, 100000))
	n, _ = t.Disc()
	out.Disc = max(0, min(n, 100000))
	if p := t.Picture(); p != nil && len(p.Data) <= tags.MaxPicture {
		out.Picture = p.Data
	}
	return out
}
func main() {
	err := sdk.Serve(os.Stdin, os.Stdout, func(q protocol.Request) (any, *protocol.Failure) {
		fail := func() (any, *protocol.Failure) {
			return nil, &protocol.Failure{Code: "invalid_audio_tags", Message: "Lecture des informations musicales indisponible."}
		}
		var in tags.Query
		if q.Capability != tags.Capability || q.Operation != "extract" || protocol.DecodeOne(q.Payload, &in) != nil || in.Size < 0 {
			return fail()
		}
		r := &reader{size: in.Size, remaining: tags.MaxRead, calls: tags.MaxCalls, ops: 20000}
		out := extract(r)
		if r.serviceErr != nil || out.Validate() != nil {
			return fail()
		}
		return out, nil
	})
	if err != nil {
		os.Exit(1)
	}
}
