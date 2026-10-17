Depuis 0.0.12, les dossiers sont dans **Réglages → Bibliothèques**, et le suivi des fiches dans **Réglages → Métadonnées**. Le reste du fonctionnement décrit ci-dessous est conservé.

# Mosaune 0.0.12 — suivre votre bibliothèque

## Installer la version

1. Arrêtez l’ancien serveur Mosaune avec **Ctrl+C** dans sa fenêtre.
2. Copiez le dossier **data** du projet pour garder une sauvegarde avant la mise à jour.
3. Lancez **DEMARRER-WINDOWS.cmd**, rechargez la page et vérifiez **v0.0.12** dans la barre latérale.

Le format du catalogue passe de 6/7 à 8. Les dossiers, associations, progressions, playlists, clés et demandes sont conservés. Un ancien exécutable nécessite la sauvegarde de son ancien catalogue pour revenir en arrière.

## Vos 987 films déjà ajoutés

Il n’est pas nécessaire de retirer le dossier, de le rajouter ou de rescanner le NAS pour reprendre les fiches en attente. Si TMDB est configuré, leur identification démarre automatiquement avec le serveur. Les anciennes associations et les fiches déjà signalées « à vérifier » sont conservées.

Dans **Réglages → Affiches et fiches**, chaque bibliothèque affiche une barre et quatre compteurs :

- **Identifiées** : une association a été enregistrée.
- **À vérifier** : le résultat n’est pas suffisamment sûr ; ouvrez la fiche du média et corrigez l’association.
- **En attente** : ces fichiers doivent encore être identifiés.
- **En erreur** : le fournisseur n’a pas pu répondre correctement ; le détail est dépliable.

Le pourcentage correspond aux fichiers identifiés ou à vérifier, rapportés au total. Il n’avance pas artificiellement lorsque le fournisseur échoue. Pour les séries, les compteurs représentent des fichiers d’épisodes.

**Mettre en pause** laisse finir la fiche déjà commencée, puis arrête les suivantes. **Reprendre l’identification** poursuit les fiches restantes. La pause est conservée après redémarrage. Vous pouvez naviguer ou fermer le navigateur ; le serveur doit rester ouvert pour continuer.

En cas d’erreur, Mosaune espace ses appels et effectue au plus trois essais par fichier. **Relancer les erreurs** remet ces essais à zéro, sans refaire les fiches réussies et sans sortir d’une pause choisie. Si le bouton signale qu’une fiche se termine, réessayez un instant plus tard. Une affiche absente n’est pas forcément une erreur de fiche : les données du fournisseur peuvent être incomplètes.

## La barre du scan des fichiers

Dans **Réglages → Vos dossiers**, cliquez sur **Actualiser** pour rechercher les changements sur le disque, ou ajoutez un nouveau dossier.

La barre distingue deux étapes :

1. **Découverte du dossier** : compteur des fichiers trouvés, barre animée sans pourcentage, car le total n’est pas encore connu.
2. **Analyse** : nombre analysé / total, avec pourcentage, puis confirmation de fin après l’enregistrement.

Cette barre couvre films, séries et musique. Gardez l’onglet ouvert pendant cette analyse des fichiers. Elle reste liée à la requête de scan ; une interruption conserve le catalogue précédent et demande de relancer le scan. Seule l’identification des fiches reprend automatiquement au redémarrage.

Un gros fichier vidéo n’est pas chargé en mémoire pour calculer cette progression. Le scan ne modifie pas les médias. Les affiches sont récupérées ensuite, avec leur propre barre. La progression du scan en cours n’est pas enregistrée sur disque ; le suivi des fiches, lui, l’est.

## Vérifier une arrivée Radarr/Sonarr

Pour le film déjà demandé, aucune nouvelle demande n’est nécessaire.

1. Dans **Demandes**, cliquez sur **Actualiser le suivi**. Le suivi automatique vérifie aussi les services environ chaque minute.
2. Après l’import par Radarr/Sonarr, Mosaune recherche le fichier dans le dossier associé (au plus une analyse de ce dossier toutes les cinq minutes).
3. Si le chemin d’import correspond exactement à ce dossier, l’œuvre connue passe en priorité dans l’identification. Une pause de l’identification reste respectée.
4. Le suivi suivant peut afficher **Disponible** lorsque le fichier est identifié et lisible. La notification apparaît dans Mosaune ouvert ; la fiche donne alors accès à la lecture.

Le mapping doit conserver la même arborescence sous la racine distante et sous le dossier Mosaune. Par exemple `/movies/Film/film.mkv` devient `N:\films\Film\film.mkv` lorsque les racines associées sont `/movies` et `N:\films`. Un chemin incompatible ne déclenche pas une association par simple nom de fichier ; la recherche habituelle reste utilisée. Les associations manuelles et les correspondances à vérifier ne sont pas remplacées. Les fichiers multi-épisodes restent à vérifier.

Les tests de développement couvrent ces étapes avec de faux services et des fichiers de test. L’import, la notification et la lecture sur votre NAS restent à confirmer. Les codecs et le lecteur vidéo n’ont pas changé dans cette version.
