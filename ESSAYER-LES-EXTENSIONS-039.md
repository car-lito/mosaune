# 0.0.39 — un lot complet de comportements des fiches

Ce lot regroupe plusieurs extractions : commandes des fiches, artistes musicaux, format de couverture, listes de fichiers, pistes d’album disponibles/manquantes, rapprochement des titres et présentation des suggestions. Le noyau conserve le rendu thémable et l’exécution contrôlée des actions.

## Essai court

1. Ouvrir un album : vérifier artistes, lecture, file, playlist, pistes disponibles et manquantes. Les titres restent cliquables.
2. Ouvrir un morceau : la file/playlist ne reçoit que ce morceau ; le retour à l’album reste présent. Les artistes de la piste sont privilégiés quand l’édition les fournit.
3. Ouvrir un film puis une série : commandes, reprise, fichiers et saisons restent accessibles. Les suggestions dépendent toujours du fournisseur activé/configuré.
4. Réglages → Extensions → Comportements des fiches : désactiver, puis rouvrir une fiche. Les commandes et listes spécialisées disparaissent, avec un avertissement et une liste simple de fichiers. Les données restent conservées. Réactiver pour retrouver la présentation.
5. Pour vérifier un vrai remplacement, importer build/example-detailexperience.mosaune.zip, activer puis sélectionner « Lecture directe ». La fiche devient compacte et les suggestions disparaissent. Sélectionner le module officiel pour revenir.

Aucune demande de téléchargement n’est envoyée par ces essais sans clic explicite. Les demandes musicales restent annoncées comme non raccordées. Le NAS personnel n’est ni installé ni modifié.

## Ce qui reste avant un noyau complet

Les familles Films/Séries/Musique ne s’installent pas encore comme des paquets complets avec toutes leurs dépendances. Navigation et pages spécialisées, fournisseurs/connecteurs restants et choix des adaptateurs d’actions doivent encore sortir du noyau. Le socle commun de comptes, permissions, stockage et chargement d’extensions y restera volontairement. Les lecteurs personnalisés viennent ensuite.

## Vérifications

Contrat testé avec références étrangères, opérations inconnues, lignes manquantes, limites et droits de visibilité. Règles testées avec accents Unicode, pistes absentes/ambiguës, morceau isolé, reprise et album de 1000 pistes. Exécution WASI, exemple externe, sélection, annulation, migration, mise à jour/restauration, retrait/réinstallation et conservation testés. 154 tests serveur et 29 tests web réussis, tests du kit et contrôles Svelte sans erreur ni avertissement. Le test WASI de 1 000 pistes passe avec un budget de 5 secondes réservé à cette capacité ; les autres gardent leur limite habituelle. Construction Windows/Linux amd64 et Linux ARM64. Parcours navigateur validés sur une bibliothèque fictive, dont ajout de l’album à la file et lecture d’un WAV.

[Contrat auteur](extension-kit/capabilities/detailexperience/v1/README.md).

Sources publiées : [commit c773df5](https://github.com/car-lito/mosaune/commit/c773df512d0d3a57040570749f2f8c6f30401417). [Vérifications Windows/Linux et conteneurs](https://github.com/car-lito/mosaune/actions/runs/40728242107) lancées ; l’image distribuée ne sera disponible qu’après leur réussite. Aucun essai sur le Raspberry Pi personnel.
