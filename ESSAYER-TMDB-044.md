# Mosaune 0.0.44 — un seul moteur TMDB

La recherche, les fiches, les suggestions, les sélections, les profils et les affiches alternatives passent désormais toujours par le module WASI TMDB. L’ancienne implémentation intégrée est supprimée. Les pages gardent leur présentation.

## Mise à jour

Arrêtez Mosaune puis relancez le lanceur habituel. La migration de la bibliothèque vers le schéma 28 conserve médias, corrections et connexion. Elle remplace une fois l’ancien choix « moteur intégré » par le module. Elle ne réactive ni un fournisseur désactivé ni un accès réseau refusé. Un paquet déjà installé est conservé, sans mise à jour forcée.

Si vous avez encore TMDB 0.4.x, installez **0.5.0** depuis Réglages → Extensions → À télécharger pour retrouver toutes les fonctions du catalogue.

## Vérifier simplement

1. Ouvrez Réglages → Extensions → TMDB. « Module de traitement » remplace l’ancien choix entre deux moteurs.
2. Avec TMDB et le module activés, utilisez votre recherche ou une fiche comme avant.
3. Cliquez « Mettre le module en pause » : les consultations TMDB s’arrêtent, les fiches et images conservées restent accessibles. La carte indique « Module en pause ».
4. Redémarrez Mosaune : la pause reste en place. Vous pouvez réactiver le module explicitement.
5. Si vous désinstallez un paquet TMDB, ses consultations restent arrêtées après redémarrage. Vous pouvez installer un autre paquet ou choisir explicitement « Activer le module fourni ». Aucun retour automatique ne se produit.

En mode de secours, aucun appel TMDB n’est repris par un autre moteur. Les fichiers locaux et les données déjà conservées restent disponibles. Les arrêts en cours d’identification laissent les fichiers en attente, sans erreur permanente.

## Ce qu’il reste à extraire

Cette étape supprime le double moteur, mais ne rend pas tout le fournisseur indépendant. Le programme WASI fourni est encore embarqué, son adaptateur réseau, le transport des images, la correspondance TMDB/TVDB pour Sonarr et certains réglages restent côté serveur. La distribution et le cycle de vie de tous les fournisseurs doivent encore être généralisés. Les lecteurs et le NAS viennent après le découpage.
