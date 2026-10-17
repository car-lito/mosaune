# Les extensions en cartes — Mosaune 0.0.17

## Mettre à jour

1. Arrêtez votre serveur avec **Ctrl+C** dans sa fenêtre.
2. Copiez le dossier **data** pour garder une sauvegarde.
3. Double-cliquez sur **DEMARRER-WINDOWS.cmd**, puis rechargez Mosaune. La version affichée doit être **0.0.17**.

Vos comptes, bibliothèques, corrections et préférences sont conservés. La base reste au format 11 utilisé par la 0.0.16.

## Ce que vous pouvez essayer

1. Ouvrez **Réglages → Extensions** avec votre compte administrateur.
2. Sur ordinateur, faites défiler la colonne des catégories : le contenu de droite reste en place. Faites ensuite défiler le contenu : la colonne garde sa position. Sur petit écran, les catégories forment une barre horizontale.
3. Dans **Sur le serveur**, les extensions sont présentées en cartes. La recherche filtre les noms et descriptions.
4. Cliquez sur une carte pour ouvrir ses réglages en grand. **← Toutes les extensions** vous ramène aux cartes.
5. Le commutateur d’un thème permet de l’activer ou de le désactiver sans ouvrir sa fiche. Sa fiche permet aussi de le choisir pour votre compte, de gérer ses utilisateurs autorisés ou de le désinstaller.
6. **À télécharger** regroupe l’import des paquets et les exemples téléchargeables. Vous pouvez récupérer Ambre 2 puis importer son fichier ZIP, vérifier l’aperçu, installer, activer et choisir le thème pour vous.

Films, Séries, Musique, MusicBrainz, Radarr et Sonarr restent intégrés au programme pour le moment. Leurs cartes ouvrent une présentation et un accès à leurs réglages actuels. Elles n’ont pas de faux commutateur : leur extraction en paquets autonomes reste à faire. La boutique et les dépôts GitHub ne sont pas encore connectés.

## Le moteur TMDB progresse aussi

La carte **TMDB** permet d’activer le module expérimental. Il prend maintenant en charge les recherches et les fiches détaillées : acteurs, genres, saisons, épisodes et bandes-annonces avec priorité à la langue demandée et repli disponible.

La connexion TMDB existante est réutilisée. L’activation ne relance pas toute votre collection ; les fiches déjà enregistrées restent disponibles. Pour provoquer un nouvel appel de détails, corrigez l’association d’un média qui en a besoin. Voir [le guide TMDB](ESSAYER-TMDB-EXTENSION.md).

Les images sont toujours chargées par le serveur et les suggestions spécialisées restent intégrées. Le module ne reçoit pas votre clé et n’écrit pas directement dans le catalogue. Il est fourni avec Mosaune ; l’import de fournisseurs exécutables communautaires n’est pas encore ouvert.

Les nouveaux boutons et onglets utilisent le contrat de thèmes commun. Les anciens thèmes héritent des nouvelles valeurs manquantes ; un auteur peut personnaliser séparément les commutateurs et les onglets.
