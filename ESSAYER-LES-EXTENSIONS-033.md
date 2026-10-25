# Informations musicales en extension — 0.0.33

Dans **Réglages → Extensions → Sur le serveur**, ouvrez **Informations des fichiers musicaux**.

Cette extension lit les titres, artistes, albums, numéros de pistes et pochettes déjà présents dans les morceaux. Elle est livrée et activée lors de la première installation ou de la migration depuis 0.0.32. Elle ne consulte aucun service en ligne.

## Essai rapide

1. Sur une bibliothèque de test, ajoutez un MP3 ou FLAC contenant des informations et une pochette. Lancez son scan : elles doivent apparaître.
2. Désactivez l’extension. Les fiches et pochettes déjà enregistrées restent visibles.
3. Ajoutez un **autre fichier** puis actualisez la bibliothèque : son nom sert de titre ; ses informations internes ne sont pas analysées.
4. Réactivez l’extension puis ajoutez un nouveau morceau : l’analyse des informations reprend. Les fichiers inchangés ne sont pas réanalysés automatiquement.
5. Pour tester la désinstallation, retirez l’extension depuis sa fiche. Sa carte passe dans **À télécharger**, où vous pouvez la réinstaller depuis la copie livrée avec Mosaune, même sans Internet. Réactivez-la ensuite.

Cette action ne supprime pas vos morceaux, les métadonnées incorporées au catalogue, les pochettes utilisées, ni les playlists. Un traitement non enregistré qui utilisait l’extension est annulé lorsqu’elle est désactivée ou retirée.

## Ce qui change pour les développeurs

Le parseur musical est un programme WASI séparé, stocké avec les paquets installés. Le noyau ne contient plus le parseur natif de tags. Le [contrat documenté](extension-kit/capabilities/audiotags/v1/README.md) permet de travailler sur ce composant indépendamment.

La lecture est limitée à un fichier ouvert par Mosaune, par blocs de 64 Kio, avec un maximum de 16 Mio effectivement lus. Les pochettes transitent une par une ; leurs petites versions sont temporairement préparées sur disque dans le dossier de données, avant la transaction du catalogue. Ce fichier temporaire est supprimé à la fin ou à l’annulation du scan. Une coupure brutale peut laisser un fichier `mosaune-artwork-*.tmp` inutilisé dans le dossier de données.

## Limites de cette étape

Le paquet officiel possède un vrai cycle installer/activer/désactiver/retirer. L’import de lecteurs de tags tiers et leur sélection ne sont pas encore ouverts : les dépôts acceptent toujours les rôles déjà pris en charge, notamment les reconnaisseurs de noms. Les interfaces spécialisées, règles de catalogue et connecteurs restants doivent encore être extraits avant les lecteurs et le premier essai NAS.

Aucun serveur personnel n’est modifié par les tests de développement. Pour essayer cette version, utilisez votre démarrage habituel après récupération de la mise à jour.

Validation finale : [tests Windows/Linux et conteneurs réussis](https://github.com/car-lito/mosaune/actions/runs/40196649971). Image ghcr.io/car-lito/mosaune:0.0.33-preview publique et vérifiée. Parcours de gestion de l’extension testé dans le navigateur sur une installation isolée.
