# Mosaune 0.0.12 — vos affiches et vos paramètres

Cette livraison correspond à la première étape du plan avant NAS : affiches de films/séries et paramètres organisés. Les comptes, permissions, thèmes, langues et extensions installables viendront dans les étapes suivantes.

## Installer

1. Arrêter l’ancien serveur avec **Ctrl+C** dans sa fenêtre.
2. Copier le dossier **data**, serveur arrêté, pour garder une sauvegarde.
3. Relancer **DEMARRER-WINDOWS.cmd**, recharger la page et vérifier **v0.0.12** dans la barre latérale.

Le catalogue passe au format 8. Dossiers, progressions, playlists, choix de métadonnées, état de l’identification et demandes sont conservés. Un retour à une ancienne version nécessite sa sauvegarde de catalogue. Aucun nouveau scan du NAS n’est nécessaire pour utiliser cette mise à jour.

## Changer une affiche

Ouvrir la grande fiche d’un film ou d’une série, puis cliquer sur **Changer l’affiche**, près de **Corriger l’association**.

- **Affiches proposées** : sélectionner une image dans la grille. Les images marquées français passent en premier, puis celles sans langue, puis les autres langues. La langue de l’application reste le français dans cette version.
- **Votre image** : cliquer sur **Importer une image** et choisir un JPEG ou PNG. L’aperçu apparaît avant enregistrement.
- Cliquer sur **Enregistrer l’affiche** pour appliquer le choix, ou **Annuler** pour ne rien changer.
- Pour retirer un choix personnalisé, ouvrir à nouveau l’éditeur, choisir **Rétablir l’affiche automatique**, puis enregistrer.

Les propositions viennent actuellement de TMDB et demandent une fiche identifiée et une clé configurée. Si le fournisseur est indisponible ou ne propose rien, l’import reste possible. Un film local non identifié accepte aussi une image importée.

Le choix s’applique à l’œuvre identifiée : plusieurs fichiers du même film et les épisodes d’une même série partagent son affiche. Une image importée avant identification reste attachée au fichier et continue de s’afficher après son identification. Les affiches individuelles des saisons et les pochettes d’albums ne sont pas modifiables dans cet écran pour l’instant.

## Ce qui est conservé

Les choix sont séparés des données automatiques : un scan, une actualisation du fournisseur ou un redémarrage ne les remplace pas. Les images choisies sont conservées localement, indépendamment du cache des fournisseurs. Les fichiers vidéo ne sont pas modifiés.

Les images acceptées sont limitées à 8 Mo et 16 millions de pixels, puis converties en JPEG avec un grand côté de 1 200 pixels maximum. Le stockage permanent des choix est plafonné à 256 Mo ; Mosaune explique si la limite est atteinte et ne supprime pas silencieusement d’autres choix. Une image partagée entre plusieurs choix n’est nettoyée qu’après le retrait de sa dernière utilisation.

## Les paramètres

L’entrée **Réglages** ouvre désormais une page avec quatre rubriques :

| Rubrique | Contenu |
|---|---|
| Bibliothèques | Dossiers, ajout, actualisation et barre du scan des fichiers |
| Métadonnées | Configuration des fournisseurs, guide TMDB, progression des fiches et pause/reprise |
| Téléchargements | Réglages Radarr/Sonarr et correspondance des dossiers |
| Serveur | Version, modules actifs, rappel de sauvegarde et ouverture de fichiers pour un essai ponctuel |

Les formulaires restent montés quand on change de rubrique pour conserver les valeurs saisies. Cette navigation prépare des rubriques supplémentaires ; elle n’installe pas encore d’extensions et n’introduit pas de comptes administrateur/utilisateurs.

## Essai conseillé

1. Sur un film identifié, sélectionner une autre affiche et vérifier sa fiche puis la rangée correspondante sur l’accueil.
2. Importer un JPEG ou PNG, enregistrer, puis recharger le navigateur : le choix doit rester visible.
3. Redémarrer Mosaune ou actualiser le dossier : le choix doit toujours être conservé.
4. Revenir à l’affiche automatique et vérifier le résultat.
5. Parcourir les quatre rubriques de paramètres ; le suivi de l’identification est maintenant dans **Métadonnées**.

Les tests de développement vérifient stockage, migrations, accès, conservation et fournisseurs simulés. Le rendu visuel et les alternatives de votre compte TMDB restent à valider dans votre navigateur. Aucun média, catalogue ou secret personnel n’a été utilisé pour ces tests.
