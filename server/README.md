# Carte du serveur

Ce dossier contient le paquet exécutable Go `main`. Construire depuis la racine avec `go build ./server` **après** préparation des ressources, ou utiliser le [script complet](../docs/development/BUILD.md). Les tests sont adjacents à leur code et utilisent des données temporaires.

| Domaine | Fichiers / préfixes |
|---|---|
| Démarrage, HTTP, réseau | `main.go`, `network.go`, `library_http.go` |
| Comptes et permissions | `accounts*.go`, `permissions.go` |
| Catalogue, fichiers, scan | `library*.go`, `scan_progress.go`, `metadata_local.go`, `metadata_queue.go` |
| Reconnaissance et tags | `media_recognition.go`, `recognition_*.go`, `audio_tags*.go`, `extraction_packages.go` |
| Familles et installation initiale | `media_bundles.go`, `media_bootstrap.go`, `media_sets.go`, `bundle_*.go` |
| Registre, configuration, droits | `extension_registry.go`, `extension_configuration*.go`, `extension_authorizations.go` |
| Dépôts et archives | `extension_repositories*.go`, `family_repository_export.go` |
| Fournisseurs communs | `metadata_provider.go`, `metadata_store.go`, `provider_composition.go`, `provider_management.go`, `provider_package*.go`, `provider_repository.go` |
| Durée de vie et transport | `provider_lifecycle.go`, `provider_module.go`, `provider_http.go`, `provider_authorizations.go`, `provider_transport*.go` |
| Adaptateurs spécifiques restants | `provider_tmdb*.go`, `provider_musicbrainz*.go`, `music_transport.go`, `music_visuals.go` |
| Images, personnes, association | `provider_images.go`, `visual_metadata.go`, `provider_album_matches.go`, `posters.go` |
| Découverte et fiches | `discovery.go`, `experience.go`, `detail_*.go`, `catalogue_*.go` |
| Demandes et suivi | `acquisition*.go` : orchestration, réglages, exécution, suivi, rapprochement et disponibilité |
| Données personnelles / musique | `personal.go`, `music.go` |
| Thèmes, navigation, pages | `theme.go`, `extensions*.go`, `ui_*.go` |

Les fichiers de migration ne sont pas des déchets. Ils protègent des catalogues déjà créés et doivent rester couverts par des tests. Les noms de fichiers ne suffisent pas à déterminer le niveau d’extraction : consulter les appels vers `extension-kit` et les [limites actuelles](../docs/STATUS.md).
