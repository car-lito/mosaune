package main

import (
	"context"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/registry"
	"testing"
)

func TestAcquisitionAvailabilityFactoryUpgrade(t *testing.T) {
	for _, mode := range []string{"enabled", "disabled", "removed", "modified", "imported"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := testCatalog(t)
			ctx := context.Background()
			store := registry.SQLStore{DB: c.db}
			s, e := store.Load(ctx)
			if e != nil {
				t.Fatal(e)
			}
			entry, _ := registryEntryFor(s, "mosaune.acquisition.plan")
			entry.Descriptor.Version = "0.9.0"
			entry.Enabled = false
			plan, e := store.Commit(ctx, s.Revision, registry.Change{Operation: "restore", ID: entry.Descriptor.ID, Entry: &entry})
			if e != nil {
				t.Fatal(e)
			}
			if mode == "disabled" || mode == "removed" {
				op := "disable"
				if mode == "removed" {
					op = "remove"
				}
				if e = c.changeRecognition(ctx, "acquisition", op, plan.After.Revision); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "modified" {
				if _, e = c.db.Exec("UPDATE recognition_packages SET code=? WHERE id=?", []byte("altered"), entry.Descriptor.ID); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "imported" {
				if _, e = c.db.Exec("UPDATE recognition_packages SET manifest='{}' WHERE id=?", entry.Descriptor.ID); e != nil {
					t.Fatal(e)
				}
			}
			if _, e = c.db.Exec("PRAGMA user_version=48"); e != nil {
				t.Fatal(e)
			}
			if e = migrateAcquisitionAvailability(c.db); e != nil {
				t.Fatal(e)
			}
			s, e = store.Load(ctx)
			if e != nil {
				t.Fatal(e)
			}
			next, present := registryEntryFor(s, entry.Descriptor.ID)
			if mode == "removed" {
				if present {
					t.Fatal("removed package restored")
				}
				return
			}
			current, _, _ := modules.Lookup("acquisition")
			want := current.Version
			if mode == "modified" || mode == "imported" {
				want = "0.9.0"
			}
			if !present || next.Descriptor.Version != want || next.Enabled != (mode != "disabled") {
				t.Fatal(mode, next)
			}
		})
	}
}
