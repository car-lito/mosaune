# Construire et vérifier Mosaune

Mosaune est un prototype en développement. Le serveur écoute uniquement sur cet ordinateur (`127.0.0.1`). La construction ARM64 ne signifie pas encore que l'installation réseau sur NAS est prête.

## Prérequis

- Git, Go dans la version indiquée par `go.mod` et Node.js 24 LTS, avec npm.
- Windows ou Linux pour les vérifications. Les mêmes sources produisent les exécutables Windows amd64, Linux amd64 et Linux arm64.
- Accès Internet pour télécharger les dépendances. Les tests utilisent des données temporaires et des services simulés ; aucune clé TMDB ni bibliothèque personnelle n'est nécessaire.

## Première préparation

```sh
git clone https://github.com/car-lito/mosaune.git
cd mosaune
npm ci
```

## Vérifications et compilation

```sh
node scripts/project.mjs check
node scripts/project.mjs build
```

Ou les deux : `node scripts/project.mjs ci`.

Le script compile le fournisseur Wasm, construit l'interface embarquée, vérifie Svelte et les tests web, puis les deux modules Go. La construction produit trois exécutables et leurs empreintes SHA-256 dans `build/`. On peut indiquer un chemin Go particulier avec la variable `MOSAUNE_GO`.

Les exécutables restent liés à la machine locale. `DEMARRER-WINDOWS.cmd` lance la version construite sous Windows. Pour Linux :

```sh
./build/mosaune-0.0.22-linux-amd64 -data-dir ./data
```

Remplacer `amd64` par `arm64` pour une machine ARM64. Le code d'installation du premier administrateur est affiché dans le terminal.

## GitHub Actions

Chaque envoi sur `main`, proposition de modification ou lancement manuel déclenche les tests sur Windows et Linux. Le travail Linux produit aussi les exécutables des trois plateformes. Les téléchargements de test restent disponibles quatorze jours dans la section **Artifacts** de l'exécution réussie.

Ces fichiers ne sont pas encore une release stable ni des images Docker. Aucune action ne déploie Mosaune sur un NAS. Les actions n'ont qu'un accès en lecture au dépôt ; aucun secret personnel n'est nécessaire aux tests.

## Données et contributions

Les dossiers `data/`, `media/`, `build/`, `node_modules/`, les fichiers `.env`, bases, journaux et clés privées sont exclus de Git. Les notes locales de reprise et de validation ne sont pas publiées. La clé publique des paquets peut être publiée ; la clé privée de signature ne doit jamais entrer dans ce dépôt.

Ne pas ajouter de vrais médias, identifiants ou exports de configuration aux exemples et rapports de bogue. Le plan de travail est dans [PLAN-AVANT-NAS.md](PLAN-AVANT-NAS.md), les contrats d'extensions dans [extension-kit/README.md](extension-kit/README.md).

La licence du code Mosaune reste à choisir avec le propriétaire du projet. La publication des sources ne constitue pas encore une licence open source ; les dépendances conservent leurs propres licences.
