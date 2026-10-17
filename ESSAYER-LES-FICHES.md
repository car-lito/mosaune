# Photos, saisons et associations — Mosaune 0.0.8

> Depuis 0.0.9, les demandes vidéo sont raccordées à Radarr/Sonarr. Voir CONFIGURER-LES-DEMANDES.md ; les mentions « bientôt » de ce guide 0.0.8 restent applicables à la musique.

## Installer la mise à jour locale

Arrêter Mosaune avec **Ctrl+C**, conserver une copie du dossier **data**, puis relancer **DEMARRER-WINDOWS.cmd** et recharger la page. Vérifier **v0.0.8** dans la barre latérale. Si nécessaire, utiliser **Ctrl+F5**.

La base reste au format v5 de la 0.0.7. Aucune réimportation n'est nécessaire. Les anciennes fiches en ligne se complètent à leur ouverture ; les fichiers musicaux et leurs tags restent intacts. Le serveur personnel n'a pas été arrêté ni son catalogue modifié pendant le développement.

## Six choses à essayer

1. **Lire un album depuis Musique** : sur ordinateur, survoler une pochette de votre collection et cliquer sur **▶**. L'album démarre dans l'ordre des disques et pistes, depuis le début, sans ouvrir sa fiche. Le bouton est aussi accessible au clavier. Sur écran tactile, ouvrir la fiche comme auparavant. Un album absent du serveur n'a pas ce bouton.
2. **Corriger « Le Chant des sirènes »** : ouvrir l'album, puis **Corriger l'association**, près de « Les morceaux ». Chercher le titre et **Orelsan**. Les deux champs sont modifiables ; supprimer un filtre ou raccourcir un titre si nécessaire. Choisir une proposition pour voir sa tracklist, puis **Associer cet album**. La couverture en ligne est utilisée si elle existe, avec repli sur la pochette intégrée aux fichiers. Le nom et les tags des fichiers ne sont pas réécrits.
3. **Choisir une édition précise** : dans le même panneau, ouvrir « Utiliser un lien ou un identifiant MusicBrainz » et coller le lien `/release/…` de l'édition, ou son UUID. Les résultats de recherche ordinaires sont paginés. Une autre association remplace le lien de l'album ; les favoris de l'ancienne œuvre restent conservés.
4. **Voir les personnes** : ouvrir une fiche de film/série identifiée ; la distribution affiche les portraits TMDB disponibles. Cliquer sur une personne pour sa grande photo et ses projets. Pour un artiste musical issu de vos tags, ouvrir son profil : Mosaune cherche son identité MusicBrainz ; une correspondance exacte unique ouvre son profil, plusieurs résultats proposent un choix. La photo provient du lien Wikidata de cet artiste, avec crédits et lien Wikimedia Commons sous le portrait.
5. **Parcourir les saisons** : ouvrir une série identifiée. La rangée « Les saisons » montre les affiches renseignées et le nombre d'épisodes présents. Cliquer une saison filtre vos épisodes ; **Tous mes épisodes** retire le filtre. Les saisons absentes restent consultables, avec une demande préparée qui n'envoie encore aucun téléchargement.
6. **Bandes-annonces** : ouvrir une fiche film/série. Les vidéos françaises sont proposées en premier ; une mention « VF » ou « version française » dans leur titre est prioritaire sur une vidéo française simplement sous-titrée. Sans bande-annonce dans la sélection localisée, Mosaune recherche d'autres langues. Chaque lien affiche sa miniature disponible et ouvre toujours YouTube dans un nouvel onglet.

## Images manquantes et limites

- Pour un album externe, Mosaune essaie la pochette de l'édition, puis celle du groupe d'éditions du même album sur Cover Art Archive. Il ne remplace pas une édition par un autre album d'un artiste au nom semblable.
- Les portraits de chanteurs nécessitent un lien MusicBrainz → Wikidata et une photo Wikimedia Commons en JPEG/PNG. Sans photo disponible, les initiales restent affichées. Aucun portrait artificiel n'est inventé.
- Certains albums, saisons ou bandes-annonces n'ont aucune image renseignée. Un visuel de remplacement reste visible. Les erreurs d'image sont mémorisées brièvement (dix minutes côté serveur), pour éviter de solliciter le fournisseur à chaque passage.
- Les crédits/licences des photos Commons sont accessibles depuis la page de l'artiste. Ces photos sont normalisées pour l'affichage ; les fichiers originaux ne sont pas distribués avec Mosaune.
- L'interface est actuellement en français : sa langue de référence est `fr-FR`. Un sélecteur multilingue n'est pas encore livré. La langue et les mentions VF proviennent du fournisseur ; elles ne résultent pas d'une analyse de la piste audio.
- Les identifications vidéo depuis « Corriger l'association » continuent d'agir sur le fichier choisi, comme le précise le panneau. L'association d'un album, elle, relie les fichiers regroupés sous le même album et artiste d'album.
- Aucun nouveau lecteur vidéo, traitement de codec ou raccordement Sonarr/Radarr n'est ajouté.

Les contrats, les données et l'exécutable sont vérifiés automatiquement. Le navigateur de test ne parvient toujours pas à ouvrir le serveur local ; les clics, le survol et le rendu visuel de cette version restent à confirmer dans votre navigateur.
