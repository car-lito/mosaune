package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Export only public example bytes, without opening a catalogue or reading configuration.
// Requiring a new directory prevents accidental replacement of a contributor's repository.
func exportFamilyRepository(dir string) error {
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(dir, "packages"), 0755); err != nil {
		return err
	}
	index := repositoryIndex{Schema: 1, Name: "Mosaune — familles médias (exemples)", Packages: []repositoryEntry{}}
	for _, kind := range []string{"films", "series", "music"} {
		for _, release := range []string{"13.0.0", "13.1.0"} {
			raw, err := exampleBundleArchive(kind, release)
			if err != nil {
				return err
			}
			archive, err := parseBundleArchive(raw)
			if err != nil {
				return err
			}
			m := archive.Manifest
			path := "packages/family-" + kind + "-" + release + ".zip"
			if err = os.WriteFile(filepath.Join(dir, filepath.FromSlash(path)), raw, 0644); err != nil {
				return err
			}
			index.Packages = append(index.Packages, repositoryEntry{ID: m.ID, Name: m.Name, Publisher: m.Publisher, Version: m.Version, Kind: "family", MinCore: m.MinCore, Path: path, SHA256: archive.Checksum})
		}
	}
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, repositoryIndexFile), append(raw, '\n'), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Familles Mosaune — exemples\n\nMosaune 0.0.64 ou ultérieur pour ce catalogue. Publiez ces fichiers dans un dépôt GitHub public puis ajoutez son adresse depuis Extensions. Versions 13.0.0 et 13.1.0 pour tester installation et mise à jour. Programmes et écrans déjà extraits seulement, pas des familles métier complètes. Aucune donnée utilisateur ni configuration dans cet export. L’import ne les active pas automatiquement.\n"), 0644)
}
