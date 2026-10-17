# Première refonte — Mosaune 0.0.7

Ce guide décrit la refonte initiale. Depuis la 0.0.8, les portraits, saisons, corrections d’albums et miniatures complètent ces parcours : voir [ESSAYER-LES-FICHES.md](ESSAYER-LES-FICHES.md).

## Lancer la nouvelle version

1. Arrêter Mosaune avec **Ctrl+C** dans sa fenêtre.
2. Copier le dossier **data** ailleurs, serveur arrêté, pour garder un point de retour.
3. Relancer **DEMARRER-WINDOWS.cmd**, puis recharger le navigateur. Vérifier **v0.0.7** dans la barre latérale sur une grande fenêtre.

Le catalogue passe de SQLite v4 à v5. Dossiers, identifications, listes, progressions et playlists sont conservés. Les anciennes versions refusent ce format ; revenir en arrière nécessite la sauvegarde correspondante. Aucun fichier média n'est modifié.

## Où retrouver les choses

| Vous voulez… | Où aller |
|---|---|
| Reprendre un média | Première rangée de l'accueil : **Reprendre le fil** |
| Ajouter un dossier / configurer TMDB | **Réglages**, en bas du menu |
| Chercher un titre | Barre de recherche en haut, accessible partout |
| Voir une fiche complète | Cliquer sur une affiche ou une pochette |
| Retrouver les œuvres gardées pour plus tard | **Ma liste** |
| Gérer les playlists | **Playlists** |
| Voir des suggestions | En bas de la page d'un film, d'une série ou d'un album identifié |

Le menu Découvrir disparaît ; ses fonctions sont réparties dans ces parcours. La lecture reste indépendante de la navigation.

## Un parcours de cinq minutes

1. **Accueil** : parcourir les rangées de films, séries et albums. Les sélections cinéma apparaissent si TMDB est configuré. Elles sont annoncées comme sélections TMDB en Belgique, pas comme un classement personnalisé de vos goûts.
2. **Reprise** : lire un peu un morceau ou un épisode, puis fermer le lecteur. Il doit apparaître dans **Reprendre le fil**, avec une barre de progression. Le menu **•••** permet de le marquer vu/écouté ou non vu/non écouté. Les deux actions le retirent de cette rangée ; « non lu » remet sa position à zéro pour un prochain départ au début.
3. **Recherche** : depuis Films, taper « Iron Man ». Le menu sous la barre montre uniquement des films et indique **Chez vous** ou **À découvrir**. Depuis Musique, il cherche des albums/artistes en ligne ainsi que les albums et morceaux locaux. Sur l'accueil, plusieurs univers peuvent apparaître. Flèches et Entrée permettent de choisir un résultat ; Échap ferme le menu.
4. **Films et séries** : cliquer une affiche. La grande fiche réunit les informations disponibles, les actions Lire/Reprendre, Ma liste/Favori, la distribution cliquable et les suggestions. Cliquer un acteur ouvre sa filmographie. Les bandes-annonces ouvrent YouTube dans un nouvel onglet. Les informations dépendent de ce que le fournisseur renseigne.
5. **Musique** : ouvrir Grand Garçon. Cliquer le titre d'un morceau ouvre sa page ; le bouton ▶ lance directement sa lecture. Cliquer un artiste ouvre ses projets locaux, puis permet d'explorer MusicBrainz. Une fiche MusicBrainz dispose de crédits d'artistes cliquables et d'autres éditions/projets du même artiste.
6. **Playlists** : ouvrir une playlist dans son grand écran. Les flèches ↑/↓ réorganisent les pistes ; × retire une piste de la playlist. Cliquer sa pochette pour choisir une image **JPEG ou PNG, maximum 8 Mo et 16 millions de pixels**. Les changements sont conservés sur le serveur, y compris après redémarrage.

## Compléter un album et voir les morceaux manquants

Dans la page d'un album local, cliquer **Compléter la tracklist**. MusicBrainz propose des éditions : vérifier l'artiste, la date et le pays, puis cliquer **Associer cette édition**. C'est une confirmation de correspondance, pas un transfert de fichiers.

Les pistes de cette édition sont comparées à vos fichiers par titre et numéro de disque/piste. Les pistes retrouvées sont lisibles ; les autres sont grisées avec **Demander**. Des fichiers locaux sans correspondance certaine restent accessibles dans une section séparée. Vos tags et pochettes intégrées sont conservés. Le choix d'une édition se change depuis la même page.

Les boutons **Demander** et **Demander le téléchargement** sont présents pour établir le parcours, mais expliquent qu'aucune demande n'est encore envoyée. Sonarr/Radarr et l'acquisition musicale ne sont pas raccordés.

## Ce qui dépend encore des données et des prochaines étapes

- Les dates d'ajout sont enregistrées à partir de cette version. Pour les anciens fichiers, l'historique d'ajout n'est pas reconstitué artificiellement.
- MusicBrainz est un fournisseur gratuit sans clé, avec débit limité et cache. Il peut manquer une sortie, une édition ou une pochette ; Mosaune ne les invente pas. Les recherches en ligne transmettent au fournisseur le titre ou l'identifiant nécessaire ; les fichiers restent locaux.
- La rangée **À l'horizon, côté musique** utilise jusqu'à trois artistes identifiés d'albums associés ou favoris. Elle montre des éditions futures annoncées, qui peuvent être des rééditions. Elle reste absente sans résultat. Le suivi de tous les artistes et la découverte d'artistes similaires selon vos goûts viendront avec les abonnements et recommandations musicales.
- Les résultats musicaux en ligne sont des **éditions d'albums**, pas encore une recherche autonome de tous les singles et morceaux. Les morceaux de votre collection sont recherchables.
- Les portraits des personnes utilisent actuellement leurs initiales. La lecture des bandes-annonces intégrée dans Mosaune reste à venir ; les liens ouvrent la vidéo référencée par TMDB.
- Listes et historique restent mono-utilisateur. Les profils multiples, les demandes réelles de téléchargement, les notifications et le nouveau lecteur vidéo ne font pas partie de cette refonte.

Le serveur, les contrats fournisseur et les règles de classement sont testés. La vérification visuelle automatisée n'a pas pu joindre le serveur local ; disposition, clavier, petites fenêtres et ressenti restent à confirmer avec ce parcours.
