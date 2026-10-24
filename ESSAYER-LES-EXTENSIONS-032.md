# Mosaune 0.0.32 — installer un reconnaisseur depuis un dépôt

Cette étape raccorde les modules externes de reconnaissance au catalogue GitHub. Elle ne termine pas l’extraction des familles médias et ne lance aucun déploiement NAS.

## Ce qui change

Dans **Réglages → Extensions → À télécharger**, le filtre **Reconnaissance des médias** affiche ces programmes aux côtés des thèmes et fournisseurs. Leur fiche propose la vérification, l’installation et les versions disponibles. Après installation, Mosaune ouvre directement leurs réglages.

Un module nouvellement installé reste désactivé. Il faut l’activer, puis le choisir pour les scans du type de média concerné. Une mise à jour conserve ces deux choix et garde une version restaurable. Le catalogue indique la version installée, même si le module est désactivé.

Si l’état des extensions change après la vérification, la confirmation est refusée : utilisez de nouveau **Vérifier ce paquet**. Retirer un dépôt conserve ses modules installés et leur version de retour, qui reste restaurable hors ligne.

## Essayer avec le vrai dépôt GitHub

Redémarrez vous-même Mosaune avec la nouvelle version, puis rechargez la page et vérifiez **v0.0.32** dans le menu. Gardez une copie de votre dossier `data` avant tout changement de version ; cette étape reste au schéma de catalogue 19.

1. Connectez-vous comme administrateur et ouvrez **Réglages → Extensions → À télécharger → Gérer les dépôts GitHub**.
2. Adresse : `https://github.com/car-lito/mosaune-extensions`.
3. Branche ou référence : `recognition-preview`.
4. Ajoutez le dépôt. Le catalogue d’exemple apparaît ; aucun programme ne s’installe automatiquement.
5. Filtrez par **Reconnaissance des médias**, ouvrez sa carte et choisissez **1.0.0**.
6. Cliquez sur **Vérifier ce paquet**, lisez l’aperçu, puis **Installer ce paquet**. Sa fiche s’ouvre dans **Sur le serveur**.
7. Pour essayer les scans, activez-le et choisissez-le explicitement. Utilisez uniquement un dossier de test : le module reconnaît les MKV/MP4 et ajoute « Exemple — » à leurs titres.
8. Revenez au catalogue et installez **1.1.0**. La fiche permet ensuite de restaurer **1.0.0**. Les deux versions ont volontairement le même comportement pour tester ce cycle.
9. Pour les scans habituels, réactivez et choisissez **Reconnaissance des films**, le module Mosaune fourni avec le serveur.

Le [catalogue d’exemple public](https://github.com/car-lito/mosaune-extensions/tree/recognition-preview) est séparé de la branche principale : les anciennes versions ne connaissent pas encore ce type de paquet dans un index. Le catalogue officiel habituel reste inchangé.

## Pour les contributeurs

Les constructions produisent maintenant `build/recognition-repository/`, également présent dans les fichiers de test GitHub Actions. Ce dossier contient un index et deux archives, prêts à publier à la racine d’un dépôt public. Aucun fichier de votre installation personnelle n’y entre.

Une entrée d’index utilise `kind: "recognition"`. L’index reste au schéma 1 ; l’archive exécutable est au schéma 2. Les champs ID, nom, éditeur, description, version et version minimale doivent correspondre exactement au manifeste, et `sha256` est l’empreinte du ZIP entier. La construction calcule ces champs depuis le manifeste utilisé et l’archive produite.

Les limites et contrôles de l’import local s’appliquent aussi aux dépôts : 9 Mio d’archive, vérification statique avant confirmation, droits administrateur, contrôle de révision et validation du programme dans le moteur limité avant enregistrement. L’identité de l’éditeur n’est pas vérifiée ; ni le nom d’un dépôt ni une empreinte ne constituent une signature. Le pilote TMDB conserve sa vérification de signature distincte.

Les reconnaisseurs acceptés n’ont ni services ni dépendances et reçoivent seulement les noms relatifs pendant le scan. Ce n’est pas encore un mécanisme pour installer des catalogues spécialisés, pages, lecteurs ou nouveaux types de médias.

## Vérifications

Localement : 122 tests du serveur, 22 tests de l’interface, vérifications du kit, analyse statique et compilations Windows/Linux amd64/Linux arm64 réussies. Les tests du catalogue couvrent téléchargement, aperçu, installation, mise à jour, révision périmée, conservation de l’activation et du choix, retrait du dépôt, restauration hors ligne, suppression du module, fausse fiche, mauvaise famille, limites de taille et refus des accès non autorisés par le contrat.

Les résultats de publication et du parcours navigateur sont consignés après leur vérification dans l’historique et la documentation de déploiement. Aucun test sur le NAS personnel n’est inclus.

Publication finale vérifiée : [tests Windows/Linux et conteneurs réussis](https://github.com/car-lito/mosaune/actions/runs/40111611668), image publique `ghcr.io/car-lito/mosaune:0.0.32-preview` téléchargée et vérifiée. Le parcours navigateur a couvert le téléchargement depuis le dépôt public, l’installation, le choix, la mise à jour et la restauration sur une installation de test.
