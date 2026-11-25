# Construire, vérifier et lancer

Toutes les commandes ci-dessous partent de la racine du dépôt. Git, Node.js 24/npm et la version Go de `go.mod` sont nécessaires. `MOSAUNE_GO` peut désigner un exécutable Go particulier. Les dépendances sont verrouillées ; leur installation nécessite le réseau. Les tests ordinaires utilisent des fixtures et services simulés, pas vos clés ou vos médias.

## Préparation et contrôles

```sh
npm ci
node scripts/check-repository.mjs
node scripts/project.mjs check
node scripts/project.mjs build
node scripts/test-minimal.mjs
```

`node scripts/project.mjs ci` combine check et build. Le script prépare les modules Wasm et exemples du SDK, compile le web dans `resources/web/dist/`, vérifie Svelte/Node, puis teste et analyse les deux modules Go. Le serveur reste un module à la racine ; son paquet exécutable est `./server`. Après préparation des ressources, un test ciblé peut utiliser `go test ./server -run TestNom`.

Le build produit dans `build/` trois exécutables (Windows amd64, Linux amd64, Linux arm64), leurs empreintes et `THIRD-PARTY-NOTICES.txt`. Garder les notices avec les fichiers redistribués. La compilation ARM64 n’est pas une mesure de performance sur Pi.

Le test minimal démarre **un serveur temporaire isolé**, sans fournisseur ni famille, crée son administrateur, installe Musique et vérifie un redémarrage. Il n’ouvre pas la base personnelle. Il nécessite le droit d’écouter et joindre une adresse locale ; un blocage par l’environnement de test doit être distingué d’un échec de l’application.

## Lancement volontaire

```sh
node scripts/run-server.mjs
```

Le lanceur lit la version courante dans `package.json` et démarre le binaire correspondant. L’API écoute par défaut sur `127.0.0.1:47831`. Ouvrir cette adresse ; conserver le terminal pour le code initial administrateur. Les données sont dans `data/`, ignoré par Git. Arrêter avec Ctrl+C. Le mode secours se lance par `node scripts/run-server.mjs --safe-mode`.

Sous Windows, les deux raccourcis dans `scripts/windows/` appellent ce même lanceur. Ils ne restent plus bloqués sur un ancien numéro de version. Pour des options avancées (port, fournisseurs, familles initiales, HTTPS), lancer directement le binaire de `build/` avec `-help`. Ne pas utiliser la base d’une installation importante pour expérimenter une migration.

## Construction propre et Docker

Une réorganisation doit être vérifiée depuis une copie neuve des fichiers suivis, avec `npm ci` puis `node scripts/project.mjs ci`. Ne pas compter sur un ancien `web/dist/` ou un paquet non suivi laissé dans le poste de développement.

```sh
docker build -t mosaune:local .
```

Le contexte est la racine. Vite et les modules Wasm sont construits dans les étapes de compilation ; le runtime n’embarque ni Node ni compilateur. Pour réseau HTTPS et volumes : [déploiement](../../deploy/README.md). Le jalon NAS de la [roadmap](../ROADMAP.md) reste fermé tant que ses prérequis ne sont pas remplis.

## GitHub Actions

Le workflow `.github/workflows/verify.yml` teste Windows/Linux, construit les trois binaires, contrôle les conteneurs HTTPS amd64 et arm64 sous émulation puis peut publier les préversions GHCR depuis main. Aucun job ne déploie sur un NAS. Les artefacts de test ne sont pas une release stable ; consulter le statut du commit précis avant téléchargement.

## Après une modification

Vérifier `git diff --check`, les tests adaptés, l’arborescence et les liens. Pour un changement de ressources, comparer si nécessaire les archives exportées avant/après afin de vérifier qu’un simple déplacement n’a pas changé les paquets. Mettre à jour [STATUS](../STATUS.md), [ROADMAP](../ROADMAP.md) et [CHANGELOG](../history/CHANGELOG.md) seulement selon le résultat réel.
