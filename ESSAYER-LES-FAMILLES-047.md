# Archives de familles — 0.0.47

Cette étape permet d’importer ensemble les composants déjà extraits d’une famille et son écran, puis de les mettre à jour avec un second fichier. Films, Séries et Musique restent les trois familles reconnues. Les programmes utilisent les mêmes contrats et limites que les extensions individuelles.

## Tester sans créer de paquet

1. Ouvrez **Réglages → Extensions → À télécharger → Importer un fichier depuis cet appareil**.
2. Dans **Installer une famille depuis un fichier**, ouvrez **Exemples pour les contributeurs**. Téléchargez Films **Version 1.0**.
3. Sélectionnez ce fichier dans **Archive de famille**. Lisez l’aperçu : il indique les installations, les composants réutilisés et les choix conservés.
4. Cliquez sur **Installer ce paquet de famille**. Mosaune ouvre la fiche Films. Les nouveaux programmes et le nouvel écran sont en pause ; choisissez **Installer / activer la famille**, puis confirmez.
5. Téléchargez maintenant **Version 1.1** et importez-la de la même manière. L’aperçu doit annoncer la mise à jour ; les états actif/en pause et les choix existants sont conservés.

Un nouvel écran externe n’obtient pas automatiquement l’accès pour les autres comptes : ouvrez **Composants et remplacements → Configurer l’écran → Accès des utilisateurs** pour l’accorder. Les autorisations d’un écran déjà installé restent identiques pendant sa mise à jour.

Les exemples réutilisent les fonctions Mosaune actuelles avec des identifiants distincts. Leur but est de rendre le parcours observable ; ce ne sont pas de nouveaux lecteurs ni des fonctionnalités supplémentaires. Ils peuvent servir de base à un contributeur.

## Règles importantes

- Un remplaçant explicitement choisi est conservé, même s’il est désactivé ou a été retiré. Cela inclut le retour volontaire à un composant d’origine après l’installation d’une archive.
- À la première installation, les composants exclusifs d’origine peuvent être remplacés par ceux du paquet, selon l’aperçu. L’écran fourni est alors mis en pause ; vos données sont conservées.
- Les composants partagés d’origine déjà installés sont réutilisés. Un composant partagé alternatif déjà choisi est également conservé. Ils ne sont pas réinstallés pour chaque famille.
- Une erreur annule l’ensemble des modifications. Un aperçu devenu périmé est refusé.
- Une même version doit garder exactement le même contenu. Une mise à jour conserve les identifiants et les rôles ; une version inférieure est refusée.
- Le retrait utilise les règles de la famille : composants exclusifs choisis et écran retirés, composants partagés et alternatives non sélectionnées conservés. Les informations de l’archive sont retirées ; le fichier d’origine permet de la réinstaller.

## Ce qui reste à faire

L’import se fait ici par fichier local, avec éditeur déclaré non vérifié. Le catalogue GitHub ne distribue pas encore ce nouveau type d’archive. Il n’y a pas encore de restauration globale d’une famille, de dépendances entre familles avec contraintes de versions, de contributions cumulables génériques ni de nouveau type de média librement ajouté. Les mises à jour individuelles et la restauration des programmes restent disponibles dans les réglages avancés.

L’extraction des services métier encore intégrés continue avant les lecteurs et le déploiement NAS. [Format pour les contributeurs](extensions/FAMILY-ARCHIVES.md).
