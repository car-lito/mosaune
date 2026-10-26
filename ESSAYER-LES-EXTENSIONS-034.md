# Choisir un lecteur d’informations musicales — 0.0.34

Cette étape ouvre les lecteurs de tags tiers, installables depuis un fichier ou un dépôt GitHub. Ils partagent désormais le même cycle de vie que les reconnaisseurs : aperçu, confirmation, activation, choix explicite, mise à jour, restauration et retrait.

## Votre installation actuelle

Le lecteur officiel garde son état actif, désactivé ou désinstallé lors de la mise à jour. Il reste sélectionné comme auparavant. La migration sauvegarde le catalogue. Aucun média ni donnée personnelle n’est supprimé.

Dans **Réglages → Extensions**, ouvrez **Informations des fichiers musicaux**. La fiche précise maintenant si ce lecteur est utilisé pour les scans. Plusieurs lecteurs peuvent être installés, mais un seul est choisi. Activer un lecteur ne remplace jamais votre choix.

## Tester une alternative sans développer

Le lecteur ID3v1 fourni comme exemple lit seulement les titres, artistes et albums enregistrés dans ce format ancien de MP3. Il est volontairement limité : pas de FLAC, d’ID3v2 ni de pochette. Gardez le lecteur officiel pour votre vraie collection.

1. Dans **À télécharger → Gérer les dépôts GitHub**, ajouter le dépôt **https://github.com/car-lito/mosaune-extensions**, branche **audiotags-preview**.
2. Ouvrir **Lecteur ID3v1 — exemple**, vérifier les accès, puis confirmer l’installation.
3. Sur sa fiche, cliquer **Activer**, puis **Utiliser pour les scans**.
4. Scanner un nouveau MP3 portant des tags ID3v1 dans une bibliothèque de test. Les informations sont produites par ce programme externe. Les fichiers inchangés ne sont pas réanalysés.
5. Choisir une autre version dans le catalogue pour tester une mise à jour ; la fiche permet ensuite de restaurer la précédente sans joindre le dépôt. Les deux versions de démonstration exécutent le même code.
6. Revenir sur **Informations des fichiers musicaux**, l’activer si nécessaire et cliquer **Utiliser pour les scans** pour retrouver le lecteur officiel.

La désactivation ou le retrait du lecteur choisi arrête ses analyses non enregistrées. Les nouveaux fichiers gardent les informations tirées de leur nom ; les tags et pochettes déjà conservés restent visibles. Un autre lecteur ne prend pas automatiquement le relais.

## Pour les contributeurs

Sources de l’exemple : [extension-kit/examples/audiotags](extension-kit/examples/audiotags). Le contrat de paquet schéma 2 est commun aux reconnaisseurs et aux lecteurs de tags. La contribution est media.audio-tags.v1, cible music ; la fiche du dépôt utilise le type audio-tags. Mosaune 0.0.34 minimum.

L’installation confirmée autorise uniquement la lecture bornée du fichier que le serveur est en train d’analyser, avec le service media.file.read.v1 et son opération range. Aucun chemin choisi par le module, accès réseau ou secret. Toute autre opération, dépendance ou capacité est refusée. Changer de rôle pendant une mise à jour est refusé aussi. Le moteur et les limites du lecteur officiel s’appliquent aux lecteurs tiers.

La construction habituelle produit build/audiotags-repository avec un index GitHub et deux archives prêtes à publier. Les noms historiques de table et d’API recognition-packages sont conservés pour compatibilité ; le stockage et les opérations sont partagés, sans second installateur musical.

## Ce qui reste à faire

Les grands modules média, leurs pages et les connecteurs ne sont pas encore entièrement extraits. Les lecteurs audio/vidéo personnalisés viennent après. Aucun déploiement NAS à cette étape.

Validation finale : [tests Windows/Linux et conteneurs réussis](https://github.com/car-lito/mosaune/actions/runs/40223492293). Image ghcr.io/car-lito/mosaune:0.0.34-preview publique et vérifiée. Installation, sélection, mise à jour et restauration testées dans le navigateur sur une installation isolée.
