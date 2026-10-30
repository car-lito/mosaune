# TMDB 0.5.0 dans Mosaune 0.0.43

Cette étape étend le module installé à la recherche de contenus à découvrir, aux suggestions, aux sélections de l’accueil, aux profils d’acteurs et aux affiches alternatives. Les fiches et la recherche d’association étaient déjà prises en charge. L’apparence des pages reste la même.

## Essayer

1. Dans Réglages → Extensions → À télécharger, ouvrez TMDB et choisissez la version **0.5.0** dans le catalogue officiel, après actualisation, ou dans « Inclus avec Mosaune ».
2. Vérifiez le paquet puis installez-le. Ouvrez la carte TMDB sur le serveur.
3. Dans **Moteur de traitement · avancé**, choisissez **Utiliser le moteur extension** si ce n’est pas déjà fait. TMDB doit être actif, sa connexion configurée et son accès en ligne autorisé.
4. Cherchez un film absent de votre bibliothèque, ouvrez ses suggestions, cliquez sur un acteur et consultez les affiches alternatives. Les rangées au cinéma/prochaines sorties passent aussi par le module.
5. Désactivez TMDB : les nouveaux appels sont arrêtés, vos fiches et affiches déjà conservées restent disponibles.

Si le catalogue distant ne montre que 0.4.x, le paquet fourni avec Mosaune et le lien « Télécharger TMDB 0.5.0 » dans l’import restent utilisables. Il n’est pas nécessaire de communiquer votre clé API.

## Compatibilité et limites

Une installation 0.4.x reste conservée : elle n’est pas remplacée silencieusement. Avec le moteur extension choisi, ses anciennes fonctions continuent ; pour le nouveau catalogue, Mosaune demande une mise à jour au lieu d’utiliser discrètement le moteur intégré. Une restauration vers 0.4.x rétablit donc cette limitation. Les paquets requérant le nouveau contrat ne peuvent pas s’installer sur Mosaune antérieur à 0.0.43.

Cette étape ne termine pas l’indépendance du fournisseur : le mode intégré historique, le transport des images, la correspondance d’identifiant Sonarr et certains réglages restent dans le serveur. La suppression de ces dépendances et la généralisation à MusicBrainz sont encore à faire. Aucun déploiement NAS et aucun nouveau lecteur ici.
