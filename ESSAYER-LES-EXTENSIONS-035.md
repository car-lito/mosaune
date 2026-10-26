# Organisation des catalogues par extensions — 0.0.35

Cette version retire de l’interface les règles qui regroupaient les morceaux en albums et les épisodes en séries. Trois véritables programmes les fournissent maintenant : Catalogue des films, Catalogue des séries et Catalogue musical. Ils sont installables, désactivables, désinstallables et remplaçables avec le mécanisme commun.

## Un test simple

1. Lancez cette version puis ouvrez Réglages → Extensions → Catalogue musical.
2. Désactivez le module, puis ouvrez Musique : les fichiers apparaissent individuellement et un message explique pourquoi.
3. Réactivez-le : les albums se reforment sans scanner à nouveau les dossiers.
4. Le même principe s’applique au Catalogue des séries. Les titres, épisodes, affiches et progressions sont conservés.

La migration installe et sélectionne ces trois nouveaux composants une seule fois pour garder l’affichage habituel. Par la suite, un redémarrage respecte vos désactivations et retraits. Une sauvegarde de la base précède la migration.

## Essayer un remplacement

Le paquet build/example-catalogue.mosaune.zip, également inclus dans les fichiers produits par GitHub Actions, constitue un exemple volontairement simple : il affiche les morceaux séparément avec la mention « Exemple contributeur ».

Dans Extensions → À télécharger → Importer un module d’analyse, vérifiez ce fichier, confirmez l’installation, activez-le puis choisissez « Utiliser pour le catalogue ». Activer seul ne remplace jamais votre module actuel. Pour revenir aux albums, choisissez à nouveau Catalogue musical. Il n’y a aucun transfert ni modification de vos fichiers.

Les créateurs peuvent distribuer ces modules via les dépôts GitHub déjà pris en charge. [Contrat et documentation](extension-kit/capabilities/catalogue/v1/README.md), [exemple indépendant](extension-kit/examples/catalogue).

## Ce que cette version ne termine pas

Les pages médias, la présentation des saisons, les associations aux œuvres des fournisseurs et les connecteurs restent à extraire. Les lecteurs audio/vidéo personnalisés viennent ensuite. Aucun déploiement sur le NAS à cette étape.

Validation finale : [tests Windows/Linux et conteneurs réussis](https://github.com/car-lito/mosaune/actions/runs/40347498282). Image ghcr.io/car-lito/mosaune:0.0.35-preview publique et vérifiée. Désactivation et réactivation testées dans le navigateur sur une installation isolée.
