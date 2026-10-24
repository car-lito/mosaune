# Exemple de dépôt de reconnaissance Mosaune

Ce dossier peut être publié à la racine d’un dépôt GitHub public. Il contient seulement un catalogue et deux paquets de démonstration, sans compte, secret ou média.

Dans Mosaune **0.0.32 ou plus**, ouvrez **Réglages → Extensions → À télécharger → Gérer les dépôts GitHub**. Ajoutez l’URL du dépôt et sa branche si nécessaire. Filtrez par **Reconnaissance des médias**, puis ouvrez la carte de l’exemple.

Choisissez d’abord **1.0.0**, vérifiez le paquet et confirmez l’installation. Sa fiche permet ensuite de l’activer et de le choisir pour les scans Films. Testez avec un dossier de copies sans importance : les titres obtenus commencent par « Exemple — ». Le programme reconnaît seulement les `.mkv` et `.mp4`.

Revenez au catalogue pour installer **1.1.0**. Les deux versions ont volontairement le même comportement ; elles permettent d’essayer le cycle de mise à jour et la restauration de la version précédente. Le choix du module et son activation sont conservés.

L’identité de l’éditeur n’est pas authentifiée. Le hash du catalogue garantit seulement que le fichier téléchargé correspond à la fiche. Aucune activation, sélection ou mise à jour n’est automatique. Retirer le dépôt conserve les paquets installés ; les retirer se fait depuis leur fiche.

Pour revenir au comportement habituel, activez puis choisissez **Reconnaissance des films**, le module Mosaune fourni avec le serveur. N’utilisez pas cet exemple pour rescanner votre collection principale.

Les versions antérieures à 0.0.32 ne connaissent pas le type `recognition` dans un catalogue : publiez cet exemple dans un dépôt ou une branche séparés de leurs catalogues. Les archives elles-mêmes sont importables localement à partir de 0.0.31.

Sources, contrat et fabrication : https://github.com/car-lito/mosaune/tree/main/extension-kit/examples/recognition
