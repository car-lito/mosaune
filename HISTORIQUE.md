## 0.0.28 — 20 octobre 2026

- Autorisations persistantes par extension et opération, distinctes des demandes du manifeste. Aucun nouvel accord automatique lors des mises à jour ; une opération retirée puis réintroduite nécessite un nouvel accord.
- Premier adaptateur complet : TMDB. Accès en ligne commun aux moteurs natif et Wasm, modifiable dans sa fiche par un administrateur, avec protection contre les modifications concurrentes.
- Révocation : interruption des opérations en cours et refus des résultats tardifs, même après réautorisation ; conservation des clés, fiches et images enregistrées. Les identifications interrompues restent en attente.
- Catalogue schéma 17 avec sauvegarde. TMDB conserve une fois son accès historique et reste autorisé par défaut sur une installation neuve ; les nouveaux propriétaires génériques démarrent sans accord.
- Autres adaptateurs, aperçu des nouveaux accès dans les installateurs génériques et extraction des modules média restent à raccorder. Aucun déploiement NAS.

## 0.0.27 — 19 octobre 2026

- Réglages des extensions : stockage générique par propriétaire dans SQLite, schémas enregistrés, valeurs ordinaires et secrets séparés, révisions contre les écrasements concurrents. Les secrets ne sont jamais renvoyés par l’API ; pas de chiffrement au repos ajouté.
- Routes de configuration communes réservées à l’administrateur et liées aux modules enregistrés ; compatibilité des anciennes routes conservée. Rechargement proposé après une erreur.
- Mise à jour/restauration : refus de supprimer ou reclassifier une valeur existante sans migration explicite. Le formulaire, l’archive et le registre évoluent dans une même transaction.
- Catalogue schéma 16 avec sauvegarde : déplacement de la clé TMDB dans le stockage commun. Fournisseur intégré et paquet partagent cette clé, conservée lors du retrait du seul paquet.
- Autres familles, services/permissions génériques, migrations privées, chiffrement des secrets et extraction des modules média restent en cours.

## 0.0.26 — 19 octobre 2026

- Moteur : dépendances par capacités, choix explicites entre fournisseurs, protection des consommateurs actifs et désactivation en cascade.
- Registre commun SQLite : plans de changements, révisions contre les écritures périmées, choix persistants et restauration avec états réels ; aucune sélection silencieuse d’une alternative.
- Paquet TMDB installé : archive et registre modifiés dans la même transaction lors des installations, mises à jour, restaurations et retraits ; choix de moteur durable et contrôle avant démarrage. Section Dépendances et démarrage dans sa fiche.
- Migration du catalogue au schéma 15 avec sauvegarde et reprise des paquets existants vérifiés, sans exécuter de code à la migration.
- Le pilote TMDB refuse encore les dépendances externes. Extraction des modules média, autres installateurs, réglages privés et interface générale des choix restent en cours. Lecteurs personnalisés prévus après les extensions, installation NAS neuve ensuite.

## 0.0.25 — 18 octobre 2026

- Recherche : pochettes carrées pour la musique, affiches verticales pour films et séries, type visible sous le titre.
- Distinction Album/Single/EP selon les métadonnées MusicBrainz ; repli « Sortie musicale » pour un type non renseigné. Les morceaux restent distingués des albums.
- Contrat déclaratif partagé pour les extensions : libellé libre, mots-clés, ratio, forme et cadrage de l’aperçu. Validation à la frontière du module et avant stockage, renderer commun respectant le thème, exemple Jeu/Console documenté.
- Pas de nouveau module Jeux ni de recherche globale de personnes/saisons dans cette version ; ces futurs résultats pourront réutiliser le contrat.

## 0.0.24 — 18 octobre 2026

- Interrupteur TMDB global, distinct du moteur natif ou extension ; état conservé après redémarrage et en mode secours.
- Arrêt des nouvelles recherches, suggestions, fiches, images et identifications automatiques ; annulation des opérations et rejet de leurs résultats tardifs.
- Catalogue, affiches enregistrées, corrections, clé et choix du moteur conservés ; MusicBrainz indépendant.
- Recherche et accueil actualisés après le changement, avec explication de l’état désactivé.
- Migration du catalogue vers le format 14 avec sauvegarde préalable. L’ancienne préférence de moteur ne désactive pas automatiquement le fournisseur.

# Notes historiques de développement

Ces notes décrivent les étapes précédentes. Pour compiler la version actuelle, suivre DEVELOPPEMENT.md ; pour le périmètre actuel, lire README.md. Les mentions de version, de statut de publication et de chemins ci-dessous sont historiques.

# Projet Mosaune — premier atelier

## Version 0.0.22 — cartes sans doublon

La vue « Toutes les sources » regroupe les paquets identiques des catalogues embarqué et distant. Priorité au dépôt officiel ; le filtre Source conserve chaque dépôt accessible. Identifiant, version, empreinte et métadonnées doivent correspondre pour regrouper les cartes. Aucun paquet ni donnée supprimé. Fermez le serveur puis relancez DEMARRER-WINDOWS.cmd.

## Version 0.0.21 — dépôt officiel en ligne

Le [dépôt officiel](https://github.com/car-lito/mosaune-extensions) est publié et proposé dans Réglages → Extensions → À télécharger. Cliquez sur **Charger le catalogue officiel**. Ambre et TMDB sont récupérés depuis GitHub puis vérifiés avant installation. Dernier catalogue conservé hors ligne ; actualisation volontaire, aucune installation automatique. Un dépôt officiel déjà ajouté en 0.0.20 est repris sans doublon. [Guide pas à pas](ESSAYER-LE-DEPOT-OFFICIEL-021.md).

## Version 0.0.20 — catalogue d’extensions

Cartes disponibles, filtres, choix de version et installation directe depuis le catalogue. Sources GitHub publiques ajoutables, actualisation manuelle, cache conservé hors ligne et retrait sans désinstaller les paquets. Exemples intégrés disponibles sans GitHub. Le pilote exécutable reste limité aux paquets TMDB signés.

Guide : [Essayer le catalogue](ESSAYER-LE-CATALOGUE-020.md). Modèle complet : [dépôt d’exemple](packages/mosaune-repository-example.zip). Contrat auteur : [dépôts GitHub](extension-kit/DEPOTS-GITHUB.md). Fermer Mosaune et sauvegarder data avant de lancer DEMARRER-WINDOWS.cmd ; migration du catalogue au format 13. Le dépôt officiel est désormais publié (voir la 0.0.21 ci-dessus).

Prototype local 0.0.22 — comptes et permissions, catalogue et paquets d’extensions installables ; demandes Radarr/Sonarr, suivi des arrivées ; photos, saisons, correction des albums et lecture au survol ; accueil en rangées, recherche contextuelle, grandes fiches et profils, albums MusicBrainz, playlists avec pochettes et reprise de tous les médias. Ce n'est pas encore la bêta ni un serveur à déployer sur le NAS.

## Nouveautés et mise à jour

**0.0.19 — premier fournisseur installable :** paquets TMDB signés, aperçu de compatibilité et accès, contrôle de démarrage, installation à chaud, mise à jour conservant une version de retour et restauration. Deux paquets prêts à télécharger dans Réglages → Extensions → À télécharger : 0.4.0 puis 0.4.1. Le serveur reste en 0.0.19 pendant leur mise à jour. Base au format 12 avec sauvegarde préalable ; autres fournisseurs et clés communautaires non pris en charge à ce stade. [Guide de test](ESSAYER-LES-FOURNISSEURS-019.md), [contrat et publication](extension-kit/PAQUETS-FOURNISSEURS.md).

**0.0.18 — réglages déclaratifs des extensions :** formulaire TMDB décrit dans un fichier, contrôles typés côté serveur, secrets conservés si laissés vides et retrait explicite. Services déclarés par le module et vérifiés à chaque invocation. Interrupteurs éteints plus contrastés ; barres de défilement masquées, défilement natif conservé. Base inchangée, clé existante réutilisée. [Guide de la 0.0.18](ESSAYER-LA-018.md), [contrat pour les créateurs](extension-kit/configuration/README.md).

**0.0.17 — extensions en cartes :** navigation des paramètres à défilement indépendant, onglets Sur le serveur / À télécharger, recherche et grandes fiches de réglages. TMDB peut aussi produire ses fiches riches via le moteur d’extensions. Activation dans **Réglages → Extensions → TMDB**. Les modules média restent intégrés et la boutique reste à construire. [Guide pas à pas](ESSAYER-LES-EXTENSIONS-017.md).

Les notes suivantes décrivent les versions précédentes.

**0.0.16 — premier branchement réel du moteur d’extensions :** la recherche TMDB peut maintenant s’exécuter dans un module séparé. Activation facultative dans **Réglages → Métadonnées → Affiches et fournisseurs** ; même clé, même catalogue, corrections conservées, retour au moteur habituel et mode de secours. Les fiches riches et images restent natives ; les paquets exécutables communautaires ne sont pas encore installables. [Guide de test](ESSAYER-TMDB-EXTENSION.md), [contrat des services hôte](extension-kit/SERVICES-HOTE.md).

Pour reconstruire les sources, après installation des dépendances web, utiliser `scripts/build-app.ps1` : le programme TMDB est compilé en Wasm avant son inclusion dans le serveur, puis les versions Windows/Linux sont construites. Les tests du module séparé demandent aussi `extension-kit/scripts/build-lab.ps1`.

**Chantier des extensions exécutables (14 octobre) :** un [laboratoire séparé](extension-kit/README.md) exécute une extension de démonstration et teste ses limites, son annulation et son cycle de vie. Le lanceur **TESTER-LE-MOTEUR-EXTENSIONS.cmd** est prêt à l’emploi. Le serveur reste en 0.0.15 ; ce prototype n’installe pas encore de fournisseur dans l’application et ne touche pas au catalogue.

**0.0.15 :** refonte des thèmes : fichier déclaratif complet, palette réutilisable, polices, tailles, arrondis et états des composants. Même moteur pour Mosaune et les thèmes installés ; paramètres inclus, exceptions par rubrique facultatives. Ambre 2.0.0 et anciens paquets compatibles. Guide : [ESSAYER-LES-THEMES.md](ESSAYER-LES-THEMES.md). Les points ci-dessous décrivent les étapes précédentes.

**0.0.14 :** Réglages → Extensions, import de paquets ZIP de thèmes avec vérification, activation/désactivation, mise à jour et désinstallation. Choix personnel et autorisations d’usage par compte, exemple Ambre clair/sombre, mode de secours. À cette étape les paramètres gardaient leur apparence native ; ils suivent le thème depuis 0.0.15. Guide : [ESSAYER-LES-EXTENSIONS.md](ESSAYER-LES-EXTENSIONS.md), contrat pour les auteurs : [CREER-UNE-EXTENSION.md](CREER-UNE-EXTENSION.md). Les modules/fournisseurs/connecteurs restent intégrés ; dépôts GitHub et extensions exécutables ne sont pas encore livrés.

**0.0.13 :** premier administrateur, comptes invités ou créés avec un mot de passe, permissions par bibliothèque, sessions révocables et données personnelles séparées. Mon compte, Utilisateurs et thèmes clair/sombre/système. Guide : [ESSAYER-LES-COMPTES.md](ESSAYER-LES-COMPTES.md). Le serveur reste local à cet ordinateur.

**0.0.12 :** choix d’affiches de films/séries (propositions TMDB ou import JPEG/PNG), aperçu, retour à l’affiche automatique et stockage permanent séparé du cache. Paramètres répartis entre Bibliothèques, Métadonnées, Téléchargements et Serveur. Guide : [ESSAYER-LES-AFFICHES.md](ESSAYER-LES-AFFICHES.md). Première étape du plan avant NAS ; les extensions installables restent à réaliser.

**Depuis 0.0.11 :** identification vidéo continue en arrière-plan, progression par bibliothèque, pause/reprise persistante et erreurs relançables. Le scan des fichiers a sa propre barre de progression. Les imports Radarr/Sonarr identifiés par leur chemin exact passent en priorité. Guide : [ESSAYER-LE-SCAN.md](ESSAYER-LE-SCAN.md). Les fiches en attente reprennent sans rescanner les dossiers.

Depuis 0.0.10 : une liste de dossiers vide dans Téléchargements explique maintenant quoi faire. Le bouton « Ajouter un dossier de films/séries » permet de saisir le chemin accessible par Mosaune, de l’analyser et de le sélectionner directement, sans perdre les réglages en cours.

Arrêter l'ancien serveur avec **Ctrl+C**, copier le dossier **data** pour garder un point de retour, puis relancer **DEMARRER-WINDOWS.cmd** et recharger la page. Le lanceur cible `build/mosaune-0.0.19-windows-amd64.exe`. Les dossiers, progressions, playlists, comptes et configuration fournisseur existants sont conservés ; la base est mise à niveau au démarrage.

La barre de recherche est en haut ; les résultats apparaissent dans un menu contextuel. Les affiches ouvrent une grande fiche, avec suggestions en bas de page. **Ma liste**, **Favoris**, **Playlists** et **Réglages** ont leurs entrées dans le menu. Guide actuel : [ESSAYER-LES-FICHES.md](ESSAYER-LES-FICHES.md). La navigation générale reste décrite dans [ESSAYER-LA-REFONTE.md](ESSAYER-LA-REFONTE.md). Les demandes vidéo peuvent maintenant être envoyées à Radarr/Sonarr après configuration et confirmation. Guide : [CONFIGURER-LES-DEMANDES.md](CONFIGURER-LES-DEMANDES.md). Les boutons musicaux restent préparatoires.

Après une modification des fichiers sur le disque, cliquer sur **Actualiser** pour le dossier concerné. Les fichiers déjà présents n’ont pas besoin d’un nouveau scan pour reprendre leur identification. Ouvrir ensuite une carte d'album ou de série pour afficher ses pistes/épisodes. **Fiche** ouvre les informations du fichier ; la recherche comprend aussi artistes et albums. Les versions MP3 et FLAC restent deux fichiers distincts dans l'album.

Pour les affiches, résumés et acteurs, suivre [CONFIGURER-TMDB.md](CONFIGURER-TMDB.md). Le fournisseur est facultatif. Les correspondances ambiguës se corrigent dans la fiche du média ; elles ne sont pas choisies automatiquement au hasard.

Le contrat Go prévu pour d'autres fournisseurs est expliqué dans [EXTENSIONS-METADONNEES.md](EXTENSIONS-METADONNEES.md). Les paquets installables de cette version concernent uniquement les thèmes déclaratifs : les fournisseurs demandent encore du code compilé dans le serveur.

La musique dispose maintenant d’une **file d’écoute** modifiable, du mélange de la suite, des commandes précédent/suivant et de trois modes de répétition. Les **playlists** sont enregistrées sur le serveur ; la file reste propre au navigateur et se retrouve sans démarrage automatique. Guide pas à pas : [ESSAYER-LA-MUSIQUE.md](ESSAYER-LA-MUSIQUE.md). Limites de cette première version : 200 pistes par liste, 100 playlists. Depuis 0.0.7, la progression musicale est conservée par fichier ; les albums identifiés peuvent être favoris. Le guide musical précédent est historique pour la disposition des écrans.

La base passe au format 11 avec la 0.0.16. Une copie SQLite cohérente `mosaune-before-extensions-…db` est créée dans data avant migration. Les anciens exécutables ne peuvent pas ouvrir ce format ; un retour à une ancienne version demande la copie du dossier data sauvegardée avant la mise à jour.

## Pour commencer sans savoir coder

1. Ouvrir le dossier du projet dans l'Explorateur Windows.
2. Double-cliquer sur **DEMARRER-WINDOWS.cmd**. Une fenêtre reste ouverte pendant le fonctionnement du serveur.
3. Ouvrir **http://127.0.0.1:47831** dans Edge ou Chrome si le navigateur ne s'ouvre pas automatiquement.
4. Au premier lancement, créer l’administrateur avec le code affiché dans la fenêtre du serveur. Puis dans **Réglages → Bibliothèques**, cliquer sur **Ajouter un dossier**. Commencer avec un petit dossier d'essai, contenant quelques morceaux ou vidéos.
5. Dans l'Explorateur Windows, ouvrir ce dossier puis faire **Ctrl+L**, **Ctrl+C** pour copier son adresse. Coller cette adresse dans **Chemin complet du dossier**.
6. Choisir **Musique**, **Films** ou **Séries**, donner éventuellement un nom, puis cliquer sur **Ajouter et analyser**. Les sous-dossiers sont aussi parcourus. Prévoir un dossier distinct par type de média.
7. Ouvrir la carte d'un album, film ou série, puis cliquer sur **Lire** ou sur le bouton ▶ d’une piste/un épisode. Le titre d’un morceau ouvre sa fiche. La musique continue lors de la navigation. Recharger la page : les fichiers du dossier doivent toujours être là.
8. Lire une vidéo compatible, avancer, mettre en pause puis fermer le lecteur. Après rechargement, la rangée **Reprendre le fil** permet de poursuivre. Son menu ••• propose de marquer le média vu/écouté ou non lu, avec position remise à zéro.
9. Après avoir ajouté ou retiré vous-même un fichier dans le dossier, cliquer sur **Actualiser** à côté de ce dossier. Mosaune actualise son catalogue sans supprimer de fichier sur le disque.

Pour arrêter le serveur lancé par ce script, revenir dans sa fenêtre et appuyer sur **Ctrl+C**.

Vérifier **v0.0.19** en bas de la barre latérale sur une grande fenêtre. Les anciens exécutables sont conservés. Le chantier du lecteur et des pistes est reporté ; son découpage est expliqué dans [LECTEUR-A-VENIR.md](LECTEUR-A-VENIR.md).

Le mode **Choisir des fichiers** reste disponible pour un essai ponctuel : ses entrées disparaissent au rechargement et leur progression n'est pas conservée. Les dossiers ajoutés dans **Vos dossiers**, eux, sont mémorisés par le serveur. Dans les deux cas, les originaux ne sont jamais modifiés.

## Ce qui fonctionne dans ce prototype

- Accueil et navigation Films / Séries / Musique, interface adaptée aux petites fenêtres.
- Serveur Go qui fournit l'interface et annonce les modules actifs.
- Catalogue SQLite conservé dans `data/mosaune.db` avec le lanceur Windows.
- Ajout de dossiers par type, analyse récursive des extensions reconnues, rafraîchissement manuel.
- Recherche dans le catalogue et les fichiers temporaires ; lecture depuis le serveur avec déplacement dans le média.
- Progression films/séries/musique enregistrée toutes les cinq secondes pendant la lecture, à la pause, au déplacement et à la fermeture du lecteur. Reprise sur lecture explicite ; commandes vu/écouté et non lu. Fin naturelle marquée terminée.
- Si le dossier est inaccessible ou l'analyse échoue, le catalogue précédent reste intact. Les fichiers absents après une analyse réussie sont masqués ; une modification de taille/date réinitialise la progression.
- Lecture persistante pendant la navigation interne, passage au fichier suivant du même album ou de la même série ; les fichiers temporaires gardent leur file par type.
- Tags musicaux et pochettes intégrées ; classement par album et artiste d'album, titres et numéros de pistes disponibles.
- Séries regroupées, saisons et épisodes issus des noms usuels S01E02 ou 1x02. Présentation en cartes et fiches détaillées.
- Fournisseur TMDB facultatif, mise en cache locale des textes/images, identification prudente et correction manuelle de correspondance.
- Demandes Radarr/Sonarr avec configuration guidée, confirmation, déduplication, suivi persistant, contrôle de disponibilité locale et notification dans l’interface.
- Démarrage possible avec vidéo seulement ou musique seulement.

Les deux modules sont encore des capacités intégrées au prototype, activables au démarrage. Ce n'est pas le futur mécanisme de plugins installables, ni une preuve de son isolation. L'apparence est une première proposition modifiable.

## Ce qui viendra ensuite

Installation NAS, recommandations personnalisées, connecteur musical, abonnements, webhooks d’import, comptes, apps Android et TV, hors-ligne mobile et extensions installables. Les commandes audio/vidéo de base restent celles du navigateur ; les commandes de file et de répétition sont fournies par Mosaune.

Cette première analyse est manuelle, parcourt tout le dossier et doit se terminer avant de fermer la page. Les arrivées liées aux demandes bénéficient désormais d’une analyse périodique des dossiers associés ; pas encore de surveillance générale du disque ni de tâche de scan reprenable, de correction des dossiers sources ni de pagination des grands catalogues. Les liens symboliques sont ignorés. Les noms viennent des fichiers, de leurs tags ou du fournisseur sélectionné. La progression est unique pour cette installation, sans comptes ni résolution des conflits entre onglets/appareils. Une fermeture brutale peut perdre les dernières secondes non envoyées. Les épisodes multiples gardent pour l'instant leur identification locale ; la correction d'une correspondance porte sur un fichier à la fois.

Ne pas placer la base SQLite sur un partage réseau. Pour sauvegarder le catalogue, arrêter Mosaune puis copier **tout le dossier data** ; les médias restent à sauvegarder séparément. En dehors du lanceur, l'option `-data-dir` choisit ce dossier. Sans option, il est créé à côté de l'exécutable. Les quatre médias fournis ont été utilisés en lecture seule avec un catalogue de test séparé. Aucun accès au NAS effectué.

Le serveur écoute uniquement sur **127.0.0.1**, une adresse accessible depuis cet ordinateur. Il n'a pas encore d'authentification et ne doit pas être publié sur Internet. Il peut lire un partage NAS si l’utilisateur l’ajoute explicitement dans ses dossiers. Aucun accès au NAS personnel n’a été effectué pendant le développement.

## La notion du jour

La **base de données** est le carnet de Mosaune : elle retient où se trouvent les médias et où vous en êtes dans une vidéo. SQLite range ce carnet dans un fichier, sans service supplémentaire à installer. Vos films restent dans leurs dossiers. Node et les outils de développement ne sont pas nécessaires pour lancer l'exécutable Go.

## Retours utiles pour la prochaine étape

- L'organisation Films / Séries / Musique est-elle claire ?
- Le style visuel te plaît-il, et le texte est-il suffisamment lisible ?
- La musique continue-t-elle quand tu changes d'espace ?
- Quel type de fichier fonctionne ou échoue, et dans quel navigateur ? Inutile d'envoyer le média lui-même.

## Pour reconstruire le projet plus tard

Prérequis : Node 24+, npm, et Go 1.26 ou supérieur. Versions exactes utilisées et résultats dans **VALIDATION.md**. Dépendances verrouillées dans `package-lock.json`, `go.mod` et `go.sum`. Le pilote [SQLite en Go](https://pkg.go.dev/modernc.org/sqlite) permet la construction avec `CGO_ENABLED=0`.

Depuis le dossier du projet, sous Windows ou Linux :

```sh
npm ci --ignore-scripts
npm run check
node --test web/src/progress.test.mjs web/src/collection.test.mjs web/src/music.test.mjs
npm run build
go test ./...
```

Sous Windows :

```powershell
New-Item -ItemType Directory -Force build
go build -trimpath -o build/mosaune-0.0.16-windows-amd64.exe .
./build/mosaune-0.0.16-windows-amd64.exe -data-dir ./data
```

Sous Linux x86-64 :

```sh
mkdir -p build
go build -trimpath -o build/mosaune-linux-amd64 .
./build/mosaune-linux-amd64 -data-dir ./data
```

Exemple pour n'activer que la musique : ajouter `-modules music` à la commande de démarrage. `-modules video` active films/séries. `-port 47832` change le port local. Ces options ne rendent pas le serveur accessible sur le réseau.

Les exécutables ARM64 et x86-64 produits sur Windows sont des vérifications de compilation. Ils ne prouvent pas encore un fonctionnement validé sur le Pi ou Omarchy. Aucun environnement de développement n'est installé globalement par cet atelier : les outils temporaires résident dans le dossier `work` voisin.

## Reprendre sur un autre poste

Conserver les sources, les guides et les fichiers de verrouillage. Les dossiers `node_modules`, `web/dist` et `build` se reconstruisent et sont exclus de Git. `data` contient l'état personnel, également exclu de Git ; ne pas le supprimer en pensant qu'il se reconstruit. Les chemins Windows du catalogue ne fonctionneront pas tels quels sous Linux : la migration des sources reste à préparer. Le dépôt distant n'est pas encore créé. La feuille de route maîtresse reste dans `outputs/ROADMAP-MEDIA.md`.

Avant une publication, choisir la licence du projet et ajouter les notices des dépendances. Aucun dépôt n'a été publié pendant cet atelier.
