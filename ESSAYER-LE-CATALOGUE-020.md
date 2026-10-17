Depuis le 16 octobre 2026, le vrai dépôt est publié : https://github.com/car-lito/mosaune-extensions. La 0.0.20 permet de l’ajouter manuellement ; la 0.0.21 le propose directement. Voir [le nouveau guide](ESSAYER-LE-DEPOT-OFFICIEL-021.md).

# Mosaune 0.0.20 — le catalogue d’extensions

## Démarrer cette version

Fermez Mosaune, gardez une copie du dossier `data`, puis lancez `DEMARRER-WINDOWS.cmd`. Rechargez la page : le bas du menu doit afficher **v0.0.20**. Vos comptes, bibliothèques, thèmes et paquets installés restent conservés. Le catalogue passe au format 13 et crée aussi une sauvegarde SQLite avant la migration. Ne rouvrez pas cette base avec un ancien exécutable.

## Essayer sans compte GitHub

1. Connectez-vous comme administrateur, puis ouvrez **Réglages → Extensions → À télécharger**.
2. Les cartes **Ambre** et **TMDB** proviennent de la source **Inclus avec Mosaune**. Vous pouvez rechercher et filtrer par type ou par source.
3. Ouvrez une carte, choisissez sa version, puis **Vérifier ce paquet**. Mosaune charge le fichier et vérifie son contenu avant d’afficher ses accès.
4. Cliquez sur **Installer ce paquet**. Sa fiche s’ouvre dans **Sur le serveur**. Au premier import, le paquet est désactivé : activez-le si vous souhaitez l’utiliser. Pour un thème, choisissez ensuite **Utiliser pour moi**.

Vous n’avez plus besoin de télécharger le ZIP à la main puis de l’importer. L’import depuis votre ordinateur reste disponible dans **Importer un fichier depuis cet appareil**.

Si Ambre ou TMDB sont déjà installés, la carte l’indique : il n’y a aucune raison de les réinstaller. Pour essayer le cycle TMDB sur une installation sans paquet, choisissez d’abord **0.4.0**, puis revenez au catalogue pour installer **0.4.1**. Ces deux exemples ont les mêmes fonctions médias ; ils servent à tester la mise à jour. La fiche TMDB propose ensuite la restauration de la précédente version. Une mise à jour conserve son état actif ou désactivé.

## Ajouter un dépôt GitHub public

Dans **Gérer les dépôts GitHub**, indiquez une adresse de la forme `https://github.com/auteur/depot`. Laissez le champ de référence vide pour utiliser la branche par défaut. Si nécessaire, indiquez une branche, une étiquette de version ou un identifiant de commit.

Le dépôt doit être préparé pour Mosaune : il lui faut un fichier `mosaune-repository.json` à sa racine. Un dépôt GitHub quelconque ne devient pas automatiquement un catalogue.

**Ajouter le dépôt** lit sa liste, sans installer de paquet. Ses cartes apparaissent ensuite avec leur source. **Actualiser ce dépôt** récupère sa nouvelle liste et fait apparaître les mises à jour disponibles. Aucune installation ni actualisation réseau ne se lance automatiquement en arrière-plan.

Si GitHub est indisponible, la liste précédente reste consultable. Les paquets déjà installés continuent de fonctionner. **Retirer ce dépôt** supprime uniquement sa liste : pour enlever une extension, utilisez sa fiche dans **Sur le serveur**.

## Préparer notre premier dépôt, sans programmation

L’application propose **Télécharger un dépôt d’exemple (ZIP)** dans la gestion des dépôts. Il contient uniquement le catalogue d’exemple, ses trois paquets et un petit guide. Aucun compte, catalogue de médias ou secret de votre installation n’y figure.

Pour le publier lorsque vous serez prêt :

1. Extrayez ce ZIP dans un dossier vide.
2. Créez un dépôt public sur votre compte GitHub.
3. Utilisez l’envoi de fichiers de GitHub pour ajouter le fichier `mosaune-repository.json`, le `README.md` et le dossier `packages` **à la racine** du dépôt. N’envoyez pas le ZIP entier comme seul fichier.
4. Enregistrez ces fichiers, puis copiez l’adresse du dépôt dans Mosaune.

Nous n’avons pas encore publié de dépôt officiel en ligne : aucune adresse fictive n’est utilisée. Les exemples inclus restent disponibles en attendant. Nous pourrons faire ensemble la publication et la préparation CasaOS.

## Ce que cette étape accepte

- Des thèmes déclaratifs compatibles, y compris provenant d’un dépôt communautaire.
- Les paquets TMDB portant la signature reconnue par le projet.
- Des dépôts publics GitHub, jusqu’à 20 sources ajoutées et 128 versions de paquets par source.

Ajouter un dépôt n’autorise pas les programmes de son auteur à s’exécuter. Les autres fournisseurs, les clés d’éditeurs communautaires, les gros modules média et les mises à jour automatiques restent pour les étapes suivantes. Les dépôts privés et les autres hébergeurs ne sont pas encore pris en charge.

Le fichier téléchargé doit correspondre à l’empreinte et à la fiche annoncées. Une incohérence arrête l’opération. Les paquets suivent ensuite les mêmes vérifications et mécanismes d’installation que les imports locaux.

Une copie du [dépôt d’exemple](packages/mosaune-repository-example.zip) est également livrée dans le projet. Elle permet de poursuivre si le navigateur intégré refuse l’enregistrement du ZIP.
