# État réel — 0.0.85

Référence au 26 novembre 2026. Mosaune est un **prototype utilisable pour essais locaux**, pas encore une bêta complète à déployer sur le NAS cible. La 0.0.85 réorganise le dépôt et sa documentation ; elle n’ajoute pas de lecteur ni ne termine l’indépendance des familles.

## Fonctionnalités existantes

| Domaine | Présent | Limites / preuve à consulter |
|---|---|---|
| Bibliothèques | Dossiers persistants, scan et progression, détection de fichiers, états locaux | Grosses bibliothèques à mesurer sur ARM ; `server/library*_test.go`, `metadata_queue_test.go` |
| Films / Séries | Identification TMDB, fiches, distribution, saisons, bandes-annonces, images, correction et choix d’affiches | Fournisseur et ressources disponibles nécessaires ; `visual_metadata_test.go`, `posters_test.go` |
| Musique | Tags et pochettes locales, albums, pistes, métadonnées MusicBrainz, profils, correction d’association | Résultats externes dépendants des sources ; `audio_tags_test.go`, `experience_test.go`, `provider_album_matches_test.go` |
| Lecture | Flux locaux, reprise, file musicale, lecture musicale persistante pendant la navigation | Moteurs/contrôles navigateur, pas de compatibilité universelle ni transcodage automatique |
| Organisation personnelle | Favoris, liste, progression, playlists et pochette personnalisée | Certaines opérations restent dans les services hôtes ; `music_test.go`, `experience_test.go` |
| Découverte / recherche | Résultats locaux et distants, suggestions et sélections déclarées, types et formats d’aperçu | Pas encore un moteur personnel complet d’abonnements/recommandations ; tests `web/src/*search*`, `provider_discovery_surface_test.go` |
| Acquisition | Radarr/Sonarr configurables, aperçu/demande, suivi, rapprochement après import, disponibilité locale | Services externes nécessaires ; pas de client torrent intégré ; tests `server/acquisition*_test.go` |
| Comptes | Premier admin, invitations, mot de passe facultatif à la création, réinitialisation, permissions, préférences | Comptes Mosaune locaux ; pas d’identité Google/TV ; `accounts_test.go` |
| UI | Accueil en rangées, grandes fiches, recherche déroulante, paramètres classés, cartes d’extensions | Pas d’app Android/TV/iOS ; pas de lecteur immersif final |
| Thèmes | Tokens, modes, variantes par rubrique, paquets et rendu commun | Traductions et contributions complexes à généraliser ; `theme_test.go`, `web/src/theme.test.mjs` |
| Réseau / distribution | HTTPS direct, Docker sans shell, compilation Windows/Linux amd64/ARM64, définitions CasaOS | Compilation/émulation ne remplacent pas un essai Pi/CasaOS réel ; pas de déploiement personnel autorisé |

## Ce qui est extensible aujourd’hui

- Runtime WASI limité et contrats versionnés, contrôle des entrées/sorties et des services autorisés.
- Reconnaissance, tags audio, catalogues/groupements/recherche locale, portions des fiches et expériences médias dans des composants installables/remplaçables.
- Écrans, navigation, pages, réglages et thèmes déclaratifs ; exemples de remplacement.
- Familles regroupées avec aperçu, installation, activation, retrait, mise à jour et restauration coordonnées des composants déjà extraits.
- Dépôts GitHub, paquets signés pilotes, import local et résolution du gestionnaire par capacités ; pas de repli imposé vers TMDB.
- Contrats communs de fournisseurs : composition initiale, cycle de vie, autorisations, gestion/configuration, paquets, profils, identification d’artistes, recherche d’éditions et références d’images.
- Règles des demandes et du suivi largement déplacées vers un composant partagé, réglages/secrets communs et nettoyage au retrait du dernier utilisateur.
- Test reproductible d’un serveur sans fournisseur ni famille installée, ajout de Musique seule et redémarrage : [test minimal](../scripts/test-minimal.mjs).

## Ce qui n’est pas terminé

Les adaptateurs Go TMDB/MusicBrainz sont toujours embarqués. Des formulaires, commandes, services de profils/association/playlists, parties d’orchestration et initialisations restent dans le serveur ou l’interface. Le rôle exact de chacun doit être audité avant de les déclarer noyau générique ou extension complète. Les contrats communs ajoutés en 0.0.79–84 sont des points d’extension, pas la preuve d’une distribution indépendante de tous les adaptateurs.

L’installation minimale testée désactive/omet des familles dans un exécutable qui contient encore des ressources officielles. Il reste à prouver tout le cycle de chaque famille et un exemple communautaire complet sans modification du noyau. Les contrats de confiance, migrations, budgets globaux et dépendances inter-familles doivent être finalisés.

Les lecteurs personnalisés, applications natives, hors-ligne mobile, autres médias, abonnements aux créateurs et recommandations personnelles complètes restent prévus. La [roadmap](ROADMAP.md) est la seule liste active de ces travaux.

Les anciennes estimations « environ 97 % » étaient indicatives et non pondérées. Elles ne valent pas validation de fin : les critères E1–E6 de la roadmap font foi.

## Validation et publications

La CI teste Windows/Linux, reconstruit les binaires, puis les conteneurs amd64/arm64 et publie une préversion seulement après ses contrôles. Vérifier l’exécution liée au commit courant : la réussite d’une ancienne version n’est pas celle de la tête actuelle. Les guides d’archives ne sont pas des résultats de tests récents.

La licence de Mosaune reste à définir. Les sources publiques ne constituent pas encore une licence open source générale. Ne pas publier de données personnelles pour rendre une démonstration reproductible.
