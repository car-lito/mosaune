package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/modules"
	"mosaune/extensionkit/packageformat"
	"mosaune/extensionkit/registry"
	"testing"
)

type moduleProviderFixture struct {
	MetadataProvider
	binding providerModuleBinding
}

func (p *moduleProviderFixture) Info() ProviderInfo {
	return ProviderInfo{ID: "fixture-module-provider"}
}
func (p *moduleProviderFixture) providerModule() providerModuleBinding { return p.binding }

func moduleProviderChange(t *testing.T, c *catalog, id, op string) error {
	t.Helper()
	state, err := (registry.SQLStore{DB: c.db}).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return c.changeRecognition(context.Background(), id, op, state.Revision)
}

func TestProviderModuleFollowsInstalledComposition(t *testing.T) {
	c, err := openCatalogWithComposition(t.TempDir(), suppliedMetadataProviders(), mediaBootstrapSelection{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	p := c.provider("musicbrainz")
	if providerAvailable(p) || providerEnabled(p) {
		t.Fatal("absent component advertised available")
	}
	if _, done, err := beginProvider(context.Background(), p); err == nil {
		done()
		t.Fatal("operation without dependency")
	}
	if err = bundleApply(c, bundlePreview(t, c, "music", "prepare")); err != nil {
		t.Fatal(err)
	}
	if !providerAvailable(p) || !providerEnabled(p) {
		t.Fatal("installation not reflected")
	}
	if err = moduleProviderChange(t, c, "music-results", "disable"); err != nil {
		t.Fatal(err)
	}
	if providerAvailable(p) || providerEnabled(p) {
		t.Fatal("disabled component advertised available")
	}
	for _, info := range c.providerInfo(context.Background()) {
		if info.ID == "musicbrainz" && (info.Enabled || info.Discovery) {
			t.Fatal("inactive provider advertised", info)
		}
	}
	if err = moduleProviderChange(t, c, "music-results", "enable"); err != nil {
		t.Fatal(err)
	}
	if !providerAvailable(p) {
		t.Fatal("reactivation requires server restart")
	}
	c.extensionSafeMode = true
	if providerAvailable(p) {
		t.Fatal("safe mode ignored")
	}
}

func TestProviderModuleProtectsResultAfterDecoderCloses(t *testing.T) {
	c, _ := testCatalog(t)
	defer c.close()
	p := c.provider("musicbrainz").(*musicBrainzProvider)
	call, done, err := beginProvider(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	decode, closeReader, err := p.openMusicResults(call)
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.decodeMusic(decode, "releases", "", 1, []byte(`{"count":1,"releases":[{"id":"11111111-1111-1111-1111-111111111111","title":"Result before pause"}]}`))
	closeReader()
	if err != nil || len(result.Items) != 1 {
		t.Fatal(result, err)
	}
	if call.Err() != nil {
		t.Fatal("nested reader ended outer operation")
	}
	if err = moduleProviderChange(t, c, "music-results", "disable"); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(context.Cause(call), errRecognitionUnavailable) {
		t.Fatal("outer result lease survived pause")
	}
	if err = moduleProviderChange(t, c, "music-results", "enable"); err != nil {
		t.Fatal(err)
	}
	if _, err = c.rememberWork(call, "music", musicResultMetadata(result.Items[0]), false); !errors.Is(err, errRecognitionUnavailable) {
		t.Fatal("late result persisted", err)
	}
	var count int
	if err = c.db.QueryRow("SELECT count(*) FROM works WHERE id=?", workID("music", "musicbrainz", result.Items[0].ProviderID)).Scan(&count); err != nil || count != 0 {
		t.Fatal("late work present", count, err)
	}
	fresh, finish, err := beginProvider(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	if _, err = c.rememberWork(fresh, "music", musicResultMetadata(result.Items[0]), false); err != nil {
		t.Fatal("fresh result rejected", err)
	}
}

func TestProviderModuleGenericBindingAndFailedMutation(t *testing.T) {
	c, _ := testCatalog(t)
	defer c.close()
	p := &moduleProviderFixture{binding: providerModuleBinding{owner: c, slot: "catalogue-films"}}
	call, done, err := beginProvider(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	if _, err = c.db.Exec(`CREATE TRIGGER refuse_module_remove BEFORE DELETE ON recognition_packages WHEN OLD.kind='catalogue-films' BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	if moduleProviderChange(t, c, "catalogue-films", "remove") == nil {
		t.Fatal("fixture failure missing")
	}
	if call.Err() != nil || !providerAvailable(p) {
		t.Fatal("failed mutation changed availability")
	}
	unlock, err := providerCommit(call)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	if _, err = c.db.Exec("DROP TRIGGER refuse_module_remove"); err != nil {
		t.Fatal(err)
	}
	if err = moduleProviderChange(t, c, "catalogue-films", "remove"); err != nil {
		t.Fatal(err)
	}
	if providerAvailable(p) {
		t.Fatal("removed dependency still available")
	}
	if _, err = providerCommit(call); !errors.Is(err, errRecognitionUnavailable) {
		t.Fatal("retired operation committed", err)
	}
}

func TestProviderModuleSelectionInvalidatesOldOperation(t *testing.T) {
	c, _ := testCatalog(t)
	defer c.close()
	p := c.provider("musicbrainz").(*musicBrainzProvider)
	call, done, err := beginProvider(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	desc, code, err := modules.Lookup("music-results")
	if err != nil {
		t.Fatal(err)
	}
	desc.ID = "org.example.module-lease"
	raw, err := packageformat.CreateExecutable(packageformat.ExecutableManifest{Schema: 2, Name: "Lease fixture", Publisher: "Fixture", MinCore: "0.0.55", Module: desc, Contributions: []packageformat.Contribution{{Role: "metadata.music-results.v1", Target: "music"}}}, code)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := packageformat.ParseExecutable(raw, version)
	if err != nil {
		t.Fatal(err)
	}
	installRecognitionExample(t, c, raw, pack)
	for _, op := range []string{"enable", "select"} {
		if err = moduleProviderChange(t, c, desc.ID, op); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = providerCommit(call); !errors.Is(err, errRecognitionUnavailable) {
		t.Fatal("old selection result accepted", err)
	}
	if _, closeReader, err := p.openMusicResults(call); err == nil {
		closeReader()
		t.Fatal("cancelled selection reused")
	}
	fresh, finish, err := beginProvider(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	if fresh.Value(providerModuleLeaseKey{}).(*providerModuleLease).descriptor.ID != desc.ID {
		t.Fatal("replacement not resolved")
	}
	if err = moduleProviderChange(t, c, "music-results", "remove"); err != nil {
		t.Fatal(err)
	}
	if !providerAvailable(p) || fresh.Err() != nil {
		t.Fatal("factory removal affected selected replacement")
	}
}

func TestProviderModuleRejectsCrossBindingAndCorruption(t *testing.T) {
	c, _ := testCatalog(t)
	defer c.close()
	ctx := context.Background()
	p := &moduleProviderFixture{binding: providerModuleBinding{owner: c, slot: "catalogue-films"}}
	other := &moduleProviderFixture{binding: providerModuleBinding{owner: c, slot: "catalogue-music"}}
	call, done, err := beginProvider(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	if _, finish, err := beginProvider(call, other); err == nil {
		finish()
		t.Fatal("cross binding accepted")
	}
	if _, err = c.db.Exec("UPDATE recognition_packages SET code='broken' WHERE kind='catalogue-films'"); err != nil {
		t.Fatal(err)
	}
	if _, finish, err := beginProvider(ctx, p); err == nil {
		finish()
		t.Fatal("corrupt executable accepted")
	}
}
