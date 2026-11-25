# Architecture et arborescence

## Organisation obligatoire

```text
mosaune/
├── AGENTS.md                 # consignes de reprise et invariants
├── README.md                 # introduction publique, pas un journal de versions
├── go.mod, go.sum            # module serveur et dépendances verrouillées
├── package*.json             # outils web et version du produit
├── Dockerfile               # construction depuis la racine
├── server/                  # exécutable Go, services hôtes, adaptateurs, tests
├── web/src/                 # interface Svelte/TypeScript, composants, tests web
├── resources/               # données embarquées, aucun service métier
│   ├── embedded.go          # déclarations go:embed centralisées
│   ├── extensions/          # manifestes des familles et écrans fournis
│   ├── examples/            # exemples déclaratifs de thèmes/pages
│   ├── theme/               # contrat des tokens et thème par défaut
│   ├── packages/            # paquets fournisseurs signés et fixtures de compatibilité
│   └── web/dist/            # interface compilée, générée et ignorée par Git
├── extension-kit/           # module Go indépendant : contrats, runtime, modules, outils
│   ├── capabilities/        # contrats versionnés des composants médias
│   ├── runtime/             # exécution WASI et services autorisés
│   ├── modules/             # implémentations des composants officiels
│   ├── providers/           # implémentations/déclarations des fournisseurs
│   ├── examples/            # exemples exécutables et tests de contribution
│   └── cmd/                 # outils de préparation des paquets
├── scripts/                 # construction, vérification, lancement, emballage
├── deploy/                  # Compose, CasaOS, exemples de configuration
├── docs/                    # documentation actuelle, contrats et archives séparées
├── licenses/                # notices tierces
└── .github/workflows/       # validation et publication de préversions
```

Les fichiers de configuration des outils restent à la racine lorsque ceux-ci les attendent. Aucun nouveau fichier Go métier, guide de version ou binaire de livraison à la racine. Les sorties vont dans `build/`, les données locales dans `data/`, les médias de test personnels dans `media/` : ces dossiers sont ignorés et ne sont pas distribués.

`server/` reste actuellement un paquet Go `main`, avec tests voisins. Cette réorganisation ne prétend pas découpler ses types privés. Une séparation future en paquets internes doit correspondre à une responsabilité réelle, sans dépendance circulaire ni API publique artificielle. Pour trouver un domaine, utiliser la [carte du serveur](../../server/README.md).

## Exécution

Le serveur Go expose l’API et l’interface compilée. SQLite conserve comptes, bibliothèques, œuvres, états, paramètres et registre d’extensions. Les fichiers originaux ne sont pas modifiés. La configuration locale passe par des variables et options, pas par des chemins de développement embarqués.

Le serveur importe le module local `mosaune/extensionkit`. Les contrats décrivent des requêtes et résultats bornés ; les modules WASI exécutent les règles extraites. Les services hôtes valident les entrées/sorties, autorisent les opérations, effectuent les transactions et limitent les appels réseau. Une extension ne reçoit pas librement le système de fichiers ou la base.

Les familles regroupent composants et écrans. Le registre conserve installation, révisions, activation et choix de remplaçants. Les opérations utilisent un aperçu et un état attendu pour éviter de travailler sur une configuration périmée. Les dépendances partagées survivent tant qu’une autre famille les utilise.

Les fournisseurs exposent leurs capacités via des interfaces réellement implémentées. Les routes communes choisissent l’adaptateur explicitement et contrôlent permissions, disponibilité et cycle de vie. Les annulations et gardes de validation empêchent une réponse ancienne d’être enregistrée après désactivation. Le cache commun peut garder les métadonnées et images déjà acquises.

**Limite actuelle :** TMDB et MusicBrainz ont encore des adaptateurs Go embarqués, et certains services/écrans restent spécifiques. Les archives installables et les contrats ne constituent pas encore un chargeur universel de fournisseurs natifs. Le noyau minimal testé n’est pas une preuve que toutes les familles sont indépendantes.

## Interface et ressources

Les composants Svelte affichent des pages, fiches et résultats déclarés par les contrats. La composition n’autorise pas du JavaScript communautaire arbitraire dans la page. Les thèmes fournissent les tokens, pas une deuxième application. Les contrats de recherche peuvent préciser libellés et formats d’aperçu.

Les ressources embarquées sont centralisées dans `resources/embedded.go`. Leurs chemins internes restent stables pour préserver les paquets et les tests. Vite écrit dans `resources/web/dist/`; ne pas y éditer du code. Les fichiers sources web restent dans `web/`. Les modules Wasm sont construits depuis `extension-kit/` par le script commun avant les tests Go.

## Règles pour ajouter un fichier

| Ajout | Emplacement |
|---|---|
| Service ou adaptateur hôte | `server/`, nom indiquant son domaine, test voisin |
| Règle exécutable média | `extension-kit/modules/<composant>/`, contrat versionné dans `capabilities/` |
| Fournisseur et politiques | `extension-kit/providers/<source>/` ; adaptateur approuvé si nécessaire dans `server/provider_…` |
| Élément UI | `web/src/`, réutiliser composants et tokens |
| Donnée embarquée | `resources/`, déclaration explicite dans `embedded.go` |
| Exemple exécutable | `extension-kit/examples/` ; exemple déclaratif : `resources/examples/` |
| Guide actuel | `docs/user/`, `development/`, `architecture/`, `extensions/`, `product/`, `security/` ou `legal/` |
| Note de version | une entrée de `docs/history/CHANGELOG.md`, pas un nouveau fichier à la racine |
| Ancien guide remplacé | `docs/archive/`, avec avertissement non normatif |
| Test, outil ou sortie | test auprès du code, outil dans `scripts/`, sortie ignorée dans `build/` |

Les contrats détaillés restent au plus près de leur code dans le SDK et sont reliés par l’index documentaire. Éviter plusieurs roadmaps concurrentes. Toute suppression doit vérifier usages, imports, migrations, archives signées et références documentaires. Les anciennes versions des paquets TMDB sont encore utiles aux migrations/tests : leur âge seul ne justifie pas leur retrait.
