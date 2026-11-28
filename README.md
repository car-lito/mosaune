# Mosaune

**Votre collection, vos découvertes, une interface commune.**

Mosaune est un serveur multimédia personnel auto-hébergé en Go, avec une interface web Svelte. Films, séries et musique constituent les premières familles ; les autres médias et clients viendront comme extensions et applications optionnelles.

**État : prototype 0.0.86.** Bibliothèques, métadonnées, recherche, fiches, reprise, playlists, comptes et demandes Radarr/Sonarr fonctionnent déjà. Le découpage complet en extensions reste à terminer avant les lecteurs personnalisés, puis le premier essai NAS. Les lecteurs actuels dépendent des codecs du navigateur. [Fonctionnalités et limites](docs/STATUS.md).

## Reprendre le projet

Pour un contributeur ou une IA découvrant Mosaune : lire [AGENTS.md](AGENTS.md), puis les [décisions produit](docs/product/DECISIONS.md), l’[architecture](docs/architecture/OVERVIEW.md) et la [roadmap](docs/ROADMAP.md). Ces documents consolident les choix du projet sans nécessiter l’historique d’une conversation.

## Construire et essayer

Prérequis : Git, Go selon [go.mod](go.mod), Node.js 24 et npm.

```sh
git clone https://github.com/car-lito/mosaune.git
cd mosaune
npm ci
node scripts/project.mjs ci
node scripts/run-server.mjs
```

Le dernier appel démarre explicitement le serveur local et conserve ses données dans `data/`. Ouvrir l’adresse affichée dans le terminal ; le code de création du premier administrateur y apparaît aussi. Les tests n’utilisent pas vos bibliothèques. [Guide complet](docs/development/BUILD.md).

Les exécutables Windows amd64, Linux amd64 et Linux arm64 sont produits dans `build/`. Docker/CasaOS et HTTPS sont préparés, mais la compilation ARM64 ne remplace pas une recette sur NAS. Aucune action GitHub ne déploie sur une machine personnelle.

## Repères

| Dossier | Contenu |
|---|---|
| [server/](server/README.md) | Serveur, services hôtes, adaptateurs et tests Go |
| [web/](web/) | Interface web et tests |
| [extension-kit/](extension-kit/README.md) | Contrats, runtime WASI, composants et outils de contribution |
| [resources/](resources/) | Manifestes, thèmes, paquets signés et données embarquées |
| [scripts/](scripts/) | Construction, validation, lancement et publication |
| [deploy/](deploy/README.md) | Docker, HTTPS et CasaOS |
| [docs/](docs/README.md) | État, roadmap, architecture, guides et archives |

Le modèle vise **quelques grandes extensions regroupées**, avec des composants internes remplaçables, des données communes et un cycle de vie contrôlé. Certains adaptateurs restent embarqués : Mosaune n’est pas encore un chargeur universel d’extensions natives. [Contrats fournisseurs](docs/extensions/PROVIDERS.md).

Le [dépôt officiel d’extensions](https://github.com/car-lito/mosaune-extensions) suit un cycle distinct du serveur. Les [notes de versions](docs/history/CHANGELOG.md) détaillent les étapes ; les anciens guides d’essai sont séparés des instructions actuelles.

## Données et licence

Médias, comptes, configurations et clés privées ne doivent jamais être publiés. Voir les [règles et le périmètre de l’audit](docs/security/PRIVACY.md).

La licence propre à Mosaune reste à choisir : des sources publiques ne constituent pas encore une licence open source générale. Les dépendances conservent leurs licences et notices, générées avec les exécutables. [Notices métadonnées](docs/legal/METADATA.md).
