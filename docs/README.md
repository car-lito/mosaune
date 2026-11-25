# Documentation Mosaune

Ces documents forment le dossier de reprise actuel. Aucune conversation privée n’est nécessaire pour comprendre les décisions du projet.

| Besoin | Point d’entrée |
|---|---|
| Comprendre le produit et ses limites | [État réel](STATUS.md), [décisions produit](product/DECISIONS.md) |
| Reprendre le développement | [Instructions](../AGENTS.md), [roadmap](ROADMAP.md), [contribution](development/CONTRIBUTING.md) |
| Trouver le bon dossier | [Architecture](architecture/OVERVIEW.md), [carte du serveur](../server/README.md) |
| Construire et tester | [Développement](development/BUILD.md) |
| Créer une extension | [Guide](extensions/AUTHORING.md), [SDK](../extension-kit/README.md) |
| Familles et dépendances | [Regroupement](architecture/GROUPED-EXTENSIONS.md), [manifestes médias](extensions/MEDIA-BUNDLES.md), [archives](extensions/FAMILY-ARCHIVES.md) |
| Fournisseurs | [Contrats](extensions/PROVIDERS.md), [transport](extensions/TRANSPORT.md) |
| Configurer les services | [TMDB](user/TMDB.md), [Radarr/Sonarr](user/DOWNLOADS.md) |
| Personnaliser l’apparence | [Thèmes](user/THEMES.md), [tokens](../resources/theme/contract.json) |
| Préparer les lecteurs | [Spécification](product/PLAYERS.md) |
| Préparer Docker/CasaOS | [Déploiement](../deploy/README.md), [CasaOS](../deploy/casaos/README.md) — jalon NAS encore fermé |
| Utiliser GitHub | [Premiers pas](user/GITHUB.md) |
| Données, confidentialité, licences | [Audit](security/PRIVACY.md), [notices métadonnées](legal/METADATA.md), [licence du lecteur de tags](../licenses/dhowden-tag-LICENSE.txt) |
| Consulter les anciennes étapes | [Historique](history/CHANGELOG.md), [archives](archive/README.md) |

Les contrats du SDK restent près de leur code dans `extension-kit/`. Les chapitres numérotés des contrats fournisseurs expliquent leur introduction ; leur dernière évolution et la roadmap priment sur les limites historiques mentionnées dans un ancien chapitre.
