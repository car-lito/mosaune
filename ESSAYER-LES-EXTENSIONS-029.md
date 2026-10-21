# Étape 0.0.29 — première extraction des modules média

Cette étape change surtout l'intérieur de Mosaune. Les règles qui reconnaissent les formats des films, séries et musiques, ainsi que les titres et numéros d'épisodes tirés des noms, s'exécutent dans trois programmes séparés. Elles ne sont plus exécutées comme du code intégré au serveur.

Vous n'avez aucun réglage à modifier. Au prochain scan, la progression distingue la reconnaissance des noms de l'enregistrement dans la bibliothèque. Les lecteurs, les fiches et les identifications en ligne restent les mêmes.

## Vérification facultative

1. Dans une installation de test, ajoutez un petit dossier de films ou épisodes. Un nom comme `Ma Série/Saison 01/S01E02.mkv` doit conserver son numéro de saison et d'épisode. Les fichiers texte du même dossier sont ignorés.
2. Ajoutez de la musique : les MP3/FLAC restent reconnus ; leurs tags et pochettes intégrées restent prioritaires.
3. Actualisez un dossier déjà connu : les identifiants, corrections et progressions des fichiers inchangés doivent être conservés.

Ne relancez pas toute votre collection uniquement pour tester cette évolution. Les vérifications automatiques utilisent des dossiers fictifs, sans modifier vos médias.

## Ce qui reste avant le NAS

Les trois programmes de reconnaissance sont encore livrés avec Mosaune, sans bouton de désinstallation autonome. Le code des écrans, des tags audio, des catalogues spécialisés et des connecteurs reste à extraire. La boutique ne sait pas encore installer une nouvelle famille de médias. Cette version est une étape réelle de découpage, **pas la fin du système d'extensions**.

Le critère avant NAS reste : noyau commun, fonctions média en extensions installables/retirables, puis lecteurs personnalisés remplaçables et thémables. [Contrat pour les contributeurs](extension-kit/capabilities/recognition/v1/README.md).
