# Préparer Mosaune pour Docker et CasaOS

Version 0.0.26 : registre durable des extensions et premier installateur raccordé, sur la base réseau et conteneur introduite en 0.0.23. **Préversion de développement ; le premier déploiement sur le NAS attend encore les autres points du [plan avant NAS](../PLAN-AVANT-NAS.md).** Aucun certificat, secret, média ou catalogue personnel n'est inclus dans l'image.

## Ce que cette version apporte

- Mode local habituel inchangé : HTTP sur 127.0.0.1, accessible depuis cet ordinateur.
- Mode réseau explicite avec HTTPS directement dans Mosaune, certificat fourni par l'administrateur, TLS 1.2 minimum.
- Une seule adresse publique autorisée, avec vérification de l'adresse et de l'origine des requêtes. Les en-têtes de proxy ne peuvent pas faire passer une connexion HTTP pour du HTTPS.
- Cookies de session Secure, HttpOnly et SameSite Strict en HTTPS ; code initial d'installation conservé.
- Arrêt sur SIGTERM : fermeture des connexions et des tâches avant la base SQLite, délai de grâce de cinq secondes pour les requêtes actives.
- Image sans Node, compilateur, shell ni service supplémentaire, exécutée avec UID/GID 65532. Le Compose monte les médias et certificats en lecture seule, et conserve les données dans un dossier distinct.

## Les éléments à préparer ensemble avant le NAS

1. **Adresse stable** : une adresse HTTPS exacte, par exemple `https://192.168.1.50:47831` (exemple fictif), ou un nom résolu sur le réseau. Le certificat doit couvrir cette IP ou ce nom. Un changement de port publié doit aussi être reporté dans cette adresse.
2. **Certificat et confiance** : un certificat serveur avec sa chaîne (`server.crt`) et sa clé privée (`server.key`). Pour une autorité locale, son certificat public (`ca.crt`) doit être reconnu par chaque appareil client. Un certificat public reconnu peut aussi être utilisé. Ne pas désactiver les vérifications HTTPS pour contourner une erreur de certificat.
3. **Dossiers** : données Mosaune, certificats et médias. Le dossier des données doit être accessible en écriture au compte du conteneur ; les certificats et médias en lecture. La clé privée doit rester accessible uniquement aux comptes nécessaires. Adapter UID/GID ou les droits des dossiers, sans lancer le service en administrateur pour éviter le réglage.
4. **Migration** : les chemins `N:\films` du PC ne correspondent pas aux chemins `/media/...` du conteneur. Ne pas brancher directement la base Windows sans préparer cette migration et sauvegarder les données.

La gestion automatique et le renouvellement des certificats ne sont pas fournis par cette première version. Le serveur vérifie l'adresse et les dates de validité du certificat au démarrage ; après remplacement des fichiers, il faut redémarrer le service. La terminaison HTTPS par un reverse proxy n'est pas encore prise en charge : utiliser HTTPS jusqu'au serveur, sans configurer de mode HTTP réseau.

## Options du serveur

| Option | Variable d'environnement | Rôle |
|---|---|---|
| `-listen` | `MOSAUNE_LISTEN` | IP d'écoute ; 127.0.0.1 par défaut, 0.0.0.0 dans l'image |
| `-port` | — | Port interne ; 47831 par défaut |
| `-public-url` | `MOSAUNE_PUBLIC_URL` | Adresse HTTPS exacte sans chemin ni barre finale |
| `-tls-cert` | `MOSAUNE_TLS_CERT` | Certificat et chaîne PEM |
| `-tls-key` | `MOSAUNE_TLS_KEY` | Clé privée PEM |
| `-tls-ca` | `MOSAUNE_TLS_CA` | Autorité locale facultative utilisée par le contrôle de santé |
| `-data-dir` | `MOSAUNE_DATA_DIR` | Données persistantes ; /data dans l'image |
| `-healthcheck` | — | Contrôle le serveur déjà démarré puis quitte, sans ouvrir la base |

Les options explicites priment sur les variables. Un accès réseau sans configuration HTTPS complète est refusé avant l'ouverture de la base. Le contrôle de santé vérifie le certificat, y compris pour une autorité locale ; il rejoint directement le port interne sans dépendre de la résolution externe du nom public.

Exemple de lancement technique, après préparation d'un certificat valide :

```sh
./mosaune -listen 0.0.0.0 -public-url https://nas.example:47831 -tls-cert /chemin/server.crt -tls-key /chemin/server.key -data-dir /chemin/data
```

## Docker Compose

Le fichier [compose.yml](compose.yml) utilise l'image `ghcr.io/car-lito/mosaune:0.0.26-preview`. Cette étiquette est une préversion pouvant évoluer ; chaque publication porte aussi `sha-<commit complet>` pour choisir exactement un build. Aucun canal `latest` ou stable n'est publié à ce stade.

Après les préparatifs, copier [.env.example](.env.example) vers `.env` dans ce dossier et adapter les chemins, l'adresse et l'identité du compte. La liaison réseau doit correspondre à une IP réellement attribuée au NAS. Le Compose refuse de créer silencieusement les dossiers manquants. Pour un certificat public reconnu, laisser `MOSAUNE_TLS_CA` vide.

```sh
docker compose --env-file .env -f compose.yml config --quiet
docker compose --env-file .env -f compose.yml pull
docker compose --env-file .env -f compose.yml up -d
```

Le code unique pour créer le premier administrateur se trouve dans les journaux du conteneur. Ne pas le publier dans une capture ou un rapport de problème.

Pour mettre à jour : arrêter le conteneur, sauvegarder tout le dossier des données, choisir l'image souhaitée, la récupérer puis recréer le conteneur avec les mêmes dossiers. Ne pas supprimer ces dossiers. Un retour à une version antérieure peut nécessiter la sauvegarde d'avant migration ; une ancienne image ne sait pas forcément lire une nouvelle base.

## CasaOS 0.4.15

Le catalogue suit le format CasaOS 0.4.x avec un dossier `Apps` et les métadonnées `x-casaos`. Le ZIP est généré depuis les fichiers lisibles du dépôt avec `go run scripts/package-casaos.go`.

Lien du catalogue préparé :

```text
https://raw.githubusercontent.com/car-lito/mosaune/main/deploy/mosaune-casaos.zip
```

L'ajout de ce lien au magasin ne suffit pas à configurer le serveur. Avant installation, renseigner `MOSAUNE_PUBLIC_URL`, les dossiers réels du NAS, les certificats et les droits du compte. Le champ adresse est volontairement vide dans le modèle ; le serveur explique ce qui manque au lieu de démarrer sans HTTPS.

La définition [Apps/Mosaune/docker-compose.yml](casaos/Apps/Mosaune/docker-compose.yml) peut aussi être importée comme application personnalisée. Les fichiers sont vérifiés avec Docker Compose ; l'import et la conservation de tous les réglages dans l'interface CasaOS 0.4.15 restent à confirmer sur une installation d'essai. Aucun test sur ton NAS n'a encore été effectué.

## Vérifications automatiques

GitHub teste d'abord les sources sur Windows et Linux. Il construit ensuite l'image amd64, puis arm64, et vérifie HTTPS, le contrôle de santé, la création d'un compte, la lecture partielle d'un média fictif, l'arrêt SIGTERM et la conservation du compte/catalogue après redémarrage. ARM64 est exécuté sous émulation : cela ne mesure pas les performances du Pi. La publication GHCR ne démarre qu'après réussite de ces étapes, sur la branche principale.

Références : [Docker — constructions multiarchitectures](https://docs.docker.com/build/building/multi-platform/), [GitHub — publication d'images](https://docs.github.com/en/actions/tutorials/publish-packages/publish-docker-images), [CasaOS — exemples de définitions](https://github.com/IceWhaleTech/CasaOS-AppStore/tree/main/Apps). Le magasin récent ZimaOS utilise un protocole différent ; cette préparation cible CasaOS 0.4.15.

## Validation de la préversion 0.0.23

Le 17 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/39049806714), tests Windows/Linux puis exécution des conteneurs amd64 et arm64 sous émulation pour ce dernier. HTTPS, refus d’origine étrangère, cookies sécurisés, lecture partielle d’un média fictif, arrêt propre et conservation du compte/catalogue après redémarrage sont vérifiés. Aucun test sur le NAS réel.

Image publique : `ghcr.io/car-lito/mosaune:0.0.23-preview`. Empreinte de l’index des deux architectures : `sha256:cedacea3659104943b6baf1084ce3b5beb83f007404f493c1fb8cffe73df51b3`. Le catalogue CasaOS publié a été comparé au ZIP généré localement. Ces vérifications ne valident pas encore l’import dans l’interface CasaOS 0.4.15, les performances du Pi ni la migration des données Windows.

## Validation de la préversion 0.0.24

Le 18 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/39110256760) pour le commit `9488407c81711108f4fd5e4ae73cfbadbfa1a26c`. Tests Windows/Linux, construction des trois exécutables et essais des conteneurs amd64 et arm64 (ARM sous émulation). Désactivation TMDB testée sur moteurs natif/Wasm, annulation du scan sans échec consommé, réponses tardives rejetées, permissions et état après redémarrage.

Image publique : `ghcr.io/car-lito/mosaune:0.0.24-preview`. Empreinte de l’index : `sha256:1dbbc271367a508cc1fb91d848d5f64c820565413d174e8aab5a97c8cf5c003c`. Les fichiers de chaque architecture ont été téléchargés anonymement et leur SHA-256 vérifié ; le catalogue CasaOS publié est identique à l’archive locale. Aucun essai sur le NAS réel.

La vérification locale du serveur livré avec comptes fictifs a réussi. Le contrôle visuel de cette version dans le navigateur intégré n’a pas pu être effectué : connexion à l’instance de test en échec. La compilation et la vérification statique Svelte sont réussies.

## Validation de la préversion 0.0.25

Le 18 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/39131942173) pour le commit `6fe626ca22ca83da24e54d48ad950f749c3d9243`. Tests Windows/Linux, trois exécutables et conteneurs amd64/arm64 (ARM sous émulation). Contrat de présentation validé côté Go et interface, libellés MusicBrainz, préservation des anciennes fiches détaillées et rejet des formats invalides testés.

Image publique : `ghcr.io/car-lito/mosaune:0.0.25-preview`. Empreinte de l’index : `sha256:33acccbc28dca64d43800d18f5322627027371a6dc9428edc43c7312284cbd64`. Téléchargement anonyme des fichiers de chaque architecture avec contrôle SHA-256 ; archive CasaOS publiée identique à l’archive locale. Aucun essai sur le NAS réel.

Localement : 21 tests web réussis, Svelte sans erreur ni avertissement, tests et analyse Go du serveur et du kit réussis, trois exécutables construits. Le serveur livré a accepté une bibliothèque synthétique de trois fichiers. La vérification visuelle reste à faire : le navigateur intégré n’a pas pu joindre cette instance de test, malgré son fonctionnement via l’API locale. Guide : [Essayer la recherche](../ESSAYER-RECHERCHE-025.md).

## Validation de la préversion 0.0.26

Le 19 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/39270056280) pour le commit `8eafc143d736bc5cc3f0b3324ca6f73b203623f4`. Tests Windows/Linux, trois exécutables et conteneurs amd64/arm64 (ARM sous émulation). Résolution des dépendances, protection des consommateurs, choix persistants, restauration du moteur et échecs isolés testés. Le paquet TMDB installé est raccordé au registre commun ; les autres familles ne sont pas encore extraites.

Image publique : `ghcr.io/car-lito/mosaune:0.0.26-preview`. Empreinte de l’index : `sha256:25a254ce47b7476e5161edc33aafd0964e0bb6d230ab2e7c3c3033006853414d`. Téléchargement anonyme des fichiers de chaque architecture avec contrôle SHA-256 ; archive CasaOS publiée identique à l’archive locale. Aucun essai sur le NAS réel.

Localement : 21 tests web réussis, Svelte sans erreur ni avertissement, tests et analyse Go du serveur et du kit réussis, trois exécutables construits. Tests de migration du schéma 14 vers 15 avec sauvegarde, d’installation/mise à jour/restauration/retrait transactionnels du paquet TMDB et de refus des écritures concurrentes. Aucun catalogue personnel utilisé pour ces essais. La nouvelle section de la fiche TMDB n’a pas fait l’objet d’une validation visuelle dans le navigateur. Guide : [Extensions 0.0.26](../ESSAYER-LES-EXTENSIONS-026.md).
