package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/configuration"
	"path/filepath"
	"testing"
)

func TestConfigurationRetirementEpoch(t *testing.T) {
	ctx := context.Background()
	db := registryDB(t, filepath.Join(t.TempDir(), "config.db"))
	s := configuration.SQLStore{DB: db}
	if err := s.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(ctx, "test.retire", configSchema()); err != nil {
		t.Fatal(err)
	}
	b, _ := s.For("test.retire")
	old, err := b.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = b.Remove(ctx, old.Revision); err != nil {
		t.Fatal(err)
	}
	if err = s.Register(ctx, "test.retire", configSchema()); err != nil {
		t.Fatal(err)
	}
	now, err := b.Read(ctx)
	if err != nil || now.Revision <= old.Revision || len(now.Values) != 0 {
		t.Fatal(now, err)
	}
	if err = b.Apply(ctx, configuration.Patch{Revision: &old.Revision}); !errors.Is(err, configuration.ErrConflict) {
		t.Fatal("stale initial form accepted", err)
	}
	if err = b.Remove(ctx, old.Revision); !errors.Is(err, configuration.ErrConflict) {
		t.Fatal("stale removal accepted", err)
	}
}
