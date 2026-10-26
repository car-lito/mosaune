# Préparer Mosaune pour Docker et CasaOS

Version 0.0.32 : installation et mise à jour des modules de reconnaissance depuis les dépôts GitHub, sur la base réseau et conteneur introduite en 0.0.23. **Préversion de développement ; le premier déploiement sur le NAS attend encore les autres points du [plan avant NAS](../PLAN-AVANT-NAS.md).** Aucun certificat, secret, média ou catalogue personnel n'est inclus dans l'image.

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

Le fichier [compose.yml](compose.yml) utilise l'image `ghcr.io/car-lito/mosaune:0.0.31-preview`. Cette étiquette est une préversion pouvant évoluer ; chaque publication porte aussi `sha-<commit complet>` pour choisir exactement un build. Aucun canal `latest` ou stable n'est publié à ce stade.

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

## Validation de la préversion 0.0.27

Le 19 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/39320580433) pour le commit `48935f1ffe632af3b8a357697ba1f84c41910ee8`. Tests Windows/Linux, trois exécutables et conteneurs amd64/arm64 (ARM sous émulation). Stockage par propriétaire, secrets absents des réponses, conflits d’écriture, changements de schéma incompatibles, migration et opérations transactionnelles des paquets testés.

Image publique : `ghcr.io/car-lito/mosaune:0.0.27-preview`. Empreinte de l’index : `sha256:58e4715145384b527b47ac2a3fddd7cff46c3ea05ca70a0e0c6bc2b2b41d5b5a`. Téléchargement anonyme des fichiers de chaque architecture avec contrôle SHA-256 ; archive CasaOS publique identique à l’archive locale. Aucun essai sur le NAS réel.

Localement : 21 tests web et 102 tests principaux du serveur réussis ; tests du kit, analyse Go et Svelte sans erreur ni avertissement, exécutables Windows/Linux/ARM64 construits. La suite serveur a terminé en 207.802 secondes, avec un délai local de 12 minutes après des interruptions à cinq minutes ; les limites de production et celles des vérifications GitHub sont inchangées. Un premier lancement avant autorisation réseau avait aussi échoué sur les serveurs de simulation locaux. Aucun catalogue personnel utilisé ; aucun essai visuel du nouveau bouton de rechargement dans le navigateur. Guide : [Extensions 0.0.27](../ESSAYER-LES-EXTENSIONS-027.md).

## Validation de la préversion 0.0.28

Le 20 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/39406608344) pour le commit `8682555da5e6dac94dd23fc680465bce98c15c33`. Tests Windows/Linux, exécutables et conteneurs amd64/arm64 (ARM sous émulation). Autorisations isolées par propriétaire, absence d’accord implicite lors des mises à jour, révocation des deux moteurs TMDB, refus des résultats tardifs, migration, redémarrage, comptes non administrateurs et conservation des données testés.

Image publique : `ghcr.io/car-lito/mosaune:0.0.28-preview`. Empreinte de l’index : `sha256:4897698bef10cd427aa377a1a03ac39c8394d3dce5a216546c02750daee4ece3`. Téléchargement anonyme des fichiers des deux architectures avec contrôle SHA-256 ; archive CasaOS publique identique à l’archive locale. Aucun essai sur le NAS réel.

Localement : 21 tests web et 108 tests principaux du serveur réussis ; tests du kit et analyse Go réussis, Svelte sans erreur ni avertissement, trois exécutables construits. La suite serveur a terminé en 539.981 secondes avec un délai local de 12 minutes ; limites de production et GitHub inchangées. Le formulaire de retrait d’accès et sa persistance après rechargement ont été vérifiés dans le navigateur, avec une installation vide et un compte fictif. L’ajustement final du libellé de la carte a été recompilé et vérifié par Svelte. Guide : [Extensions 0.0.28](../ESSAYER-LES-EXTENSIONS-028.md).

## Validation de la préversion 0.0.29

Le 22 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/39660049112) pour le commit `9059648a8da53d14e0aef87ff841fafe13a4db80`. Tests Windows/Linux, exécutables et conteneurs amd64/arm64 (ARM sous émulation). Trois modules de reconnaissance réellement exécutés, refus des contrats invalides, lots de 32, catalogue conservé en cas d’indisponibilité et identification de 1024 noms testés.

Image publique : `ghcr.io/car-lito/mosaune:0.0.29-preview`. Empreinte de l’index : `sha256:379941b62aa4b43ad6c53038c7441f38b2ea6b5f36024c1cd66ada152b658559`. Téléchargement anonyme des fichiers des deux architectures avec contrôle SHA-256 ; archive CasaOS publique identique à l’archive locale. Aucun essai sur le NAS réel.

Localement : 21 tests web et 111 tests principaux du serveur réussis ; tests du kit et analyse Go réussis, Svelte sans erreur ni avertissement, trois exécutables construits. Suite serveur complète relancée avec le mode interprété : 124.8 secondes, sans échec. Limites de production inchangées ; la sonde HTTP du scan en CI attend désormais jusqu’à 30 secondes pour laisser le serveur restituer ses erreurs. Le texte de progression a été compilé et vérifié par Svelte ; aucun essai visuel dans le navigateur pour cette étape. Guide : [Extensions 0.0.29](../ESSAYER-LES-EXTENSIONS-029.md).

## Validation de la préversion 0.0.30

Le 23 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/39936296316) pour le commit `523fa2d44e3cc4e4f2e57d1aa79c7fa02f268f6b`. Tests Windows/Linux, exécutables et conteneurs amd64/arm64 (ARM sous émulation). Le test du conteneur vérifie aussi le retrait d’un reconnaisseur, son absence après redémarrage, sa réinstallation désactivée et sa réactivation explicite. Aucun essai sur le NAS réel.

Image publique : `ghcr.io/car-lito/mosaune:0.0.30-preview`. Empreinte de l’index : `sha256:eb440fa0ef66db9310f949842b15312b76513e2b5eaa448aac649f9e4e3c2159`. Téléchargement anonyme des fichiers des deux architectures avec contrôle SHA-256 ; archive CasaOS publique identique à l’archive locale.

Localement : 21 tests web et 116 tests principaux du serveur réussis, kit et analyse Go réussis, Svelte sans erreur ni avertissement, trois exécutables construits. La suite serveur a terminé en 628.902 secondes (délai local de 12 minutes, limites CI inchangées). Après l’optimisation de lecture des manifestes et le raccordement du bouton Actualiser, les tests de reconnaissance, les vérifications web et les trois constructions ont été repassés ; GitHub vérifie ensuite toutes les sources finales.

Parcours navigateur testé sur une installation vide et un compte fictif : cartes, désactivation, fiche, confirmation et retrait, réinstallation depuis les modules disponibles puis activation explicite. Le dernier raccordement du bouton Actualiser est compilé et vérifié par Svelte. Guide : [Extensions 0.0.30](../ESSAYER-LES-EXTENSIONS-030.md).

## Validation de la préversion 0.0.31

Le 24 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/40083863540) pour le commit `ff7b17913df9fb7c65d4f342048bbac66e335185`. Tests Windows/Linux, trois exécutables et conteneurs amd64/arm64 (ARM sous émulation). Les conteneurs importent le paquet contributeur, l’activent et le choisissent, puis vérifient ses titres sur un véritable scan et l’absence de remplacement automatique après retrait. Aucun essai sur le NAS réel.

Image publique : `ghcr.io/car-lito/mosaune:0.0.31-preview`. Empreinte de l’index : `sha256:80934360424329e5f5bc28afb8315d2aa8e21d9e74a95c0dd04faa497d0773bb`. Téléchargement anonyme des fichiers des deux architectures avec contrôle SHA-256 ; archive CasaOS publique identique à l’archive locale.

Localement : 21 tests web et 120 tests principaux du serveur réussis, kit et analyse Go réussis, Svelte sans erreur ni avertissement, trois exécutables construits. Suite serveur : 169.754 secondes. Après un premier blocage des connexions par le bac à sable, la suite complète a été repassée avec l’autorisation réseau, sans échec.

Parcours navigateur vérifié sur une installation isolée et un compte fictif : carte externe, activation, sélection explicite et restauration 1.1.0 vers 1.0.0. L’import et les erreurs sont couverts par les tests HTTP et conteneurs. Le sélecteur de fichiers de l’outil navigateur n’a pas pu être utilisé de bout en bout ; le parcours d’import complet n’est donc pas déclaré validé visuellement. Guide : [Extensions 0.0.31](../ESSAYER-LES-EXTENSIONS-031.md).

## Validation de la préversion 0.0.32

Le 24 octobre 2026 : [vérifications GitHub réussies](https://github.com/car-lito/mosaune/actions/runs/40111611668) sur le commit `fe792912a8f5672c0f8eca7fc1fad0e9428d6583`. Tests Windows/Linux et conteneurs HTTPS amd64/arm64 (ARM sous émulation), puis publication. Aucun déploiement sur le NAS personnel.

Image publique : `ghcr.io/car-lito/mosaune:0.0.32-preview`. Index : `sha256:a2fe213d157cd58f3749070488cff264efc80cb422b94b414b2378836547fee4`. Téléchargement anonyme des fichiers des deux architectures et contrôles SHA-256 réussis ; archive CasaOS publique identique à la copie locale.

Localement : 122 tests du serveur et 22 tests de l’interface réussis, kit et analyse Go réussis, Svelte sans erreur ni avertissement. Suite serveur : 126.553 secondes. Exécutables Windows/Linux amd64/Linux arm64 reconstruits après la correction finale de libellé.

Parcours navigateur exécuté sur une installation temporaire et un compte fictif, avec le véritable dépôt public et sa branche `recognition-preview` : téléchargement/vérification de la version 1.0.0, installation et ouverture de sa fiche, activation, sélection, filtre du catalogue, mise à jour 1.1.0 et restauration 1.0.0 avec choix conservé. Le test a révélé le libellé « Import local », corrigé en « Module externe » dans le dernier commit. Le sélecteur de fichiers local n’est pas nécessaire à ce parcours ; il n’est pas déclaré validé visuellement par cette étape.

Guide : [Extensions 0.0.32](../ESSAYER-LES-EXTENSIONS-032.md). Catalogue d’exemple : [branche publique](https://github.com/car-lito/mosaune-extensions/tree/recognition-preview), commit `cd0bfbbc0de6479b2358450f60fdd411d1c022b2`. La branche principale du catalogue d’extensions n’a pas été modifiée.

## Validation de la préversion 0.0.33

Le 25 octobre 2026 : [tests GitHub réussis](https://github.com/car-lito/mosaune/actions/runs/40196649971) sur le commit a8a2019386a1f56912b3ed1cd05c8e2bff3e3703. Windows/Linux, puis conteneurs HTTPS amd64/arm64 (ARM sous émulation), puis publication. Le test de conteneur lit un MP3 synthétique contenant des tags et une vraie image PNG, sert sa pochette normalisée et vérifie sa conservation après désactivation du module et redémarrage. Aucun déploiement sur le NAS personnel.

Image publique : ghcr.io/car-lito/mosaune:0.0.33-preview. Index : sha256:a0181b3bfed34435a4031520580e0c0f6322c30f33d4bb20de2d0aae4eaf658c. Téléchargement anonyme et SHA-256 des deux architectures vérifiés, archive CasaOS publique identique à la copie locale.

Localement : 127 tests du serveur et 22 tests de l’interface réussis, kit et analyse Go réussis, Svelte sans erreur ni avertissement. Suite serveur : 128.183 secondes. Exécutables Windows/Linux amd64/Linux arm64 construits.

Parcours navigateur sur une installation temporaire : carte installée active, désactivation, ouverture de la fiche, retrait, disparition de la liste installée, présence dans À télécharger, réinstallation désactivée puis activation. [Guide utilisateur](../ESSAYER-LES-EXTENSIONS-033.md).

## Validation de la préversion 0.0.34

Vérification finale le 26 octobre 2026 : [tests GitHub réussis](https://github.com/car-lito/mosaune/actions/runs/40223492293) sur le commit 891b2b8acfa52b797c453123c9d3ea8923d283eb. Tests Windows/Linux, conteneurs HTTPS amd64/arm64 (ARM sous émulation), puis publication. Les conteneurs installent et sélectionnent le lecteur ID3v1 externe, analysent un MP3 synthétique et vérifient ses métadonnées ainsi que la conservation du fichier original. Aucun déploiement sur le NAS personnel.

Image publique : ghcr.io/car-lito/mosaune:0.0.34-preview. Index : sha256:02173825a231ae9e14e390d61119d7df8dba0639f55d451abdd75e3838ff8a91. Téléchargement anonyme et SHA-256 des deux architectures vérifiés ; archive CasaOS publique identique à la copie locale.

Localement : 131 tests serveur et 22 tests interface réussis, kit et analyse Go réussis, Svelte sans erreur ni avertissement. Exécutables Windows/Linux amd64/Linux arm64 construits. Les tests couvrent notamment migration, sélection explicite, annulation, refus des accès excessifs, mise à jour, restauration et conservation des métadonnées.

Parcours navigateur sur une installation temporaire : dépôt public audiotags-preview, installation 1.0.0, activation sans sélection implicite, sélection, mise à jour 1.1.0 et restauration 1.0.0 en conservant le choix. [Guide utilisateur](../ESSAYER-LES-EXTENSIONS-034.md).
