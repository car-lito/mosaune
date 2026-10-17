# Installer et actualiser TMDB — Mosaune 0.0.19

Cette version permet d’installer un vrai programme de fournisseur depuis un paquet ZIP, puis de le remplacer sans mettre à jour Mosaune. Le premier fournisseur pris en charge est TMDB.

## Mettre Mosaune à jour une première fois

1. Arrête le serveur avec **Ctrl+C**.
2. Copie le dossier **data** pour garder un point de retour.
3. Lance **DEMARRER-WINDOWS.cmd**, recharge la page et vérifie **v0.0.19**.

La base passe au format 12 pour conserver les paquets et leur version de retour. Mosaune crée aussi une sauvegarde SQLite avant la migration. Pour revenir à un ancien serveur, utilise la copie de data faite avant cette mise à jour.

## Installer le premier paquet

1. Avec ton compte administrateur, ouvre **Réglages → Extensions → À télécharger**.
2. Dans **Installer un fournisseur**, télécharge **TMDB 0.4.0**.
3. Sélectionne ce ZIP dans **Paquet fournisseur à vérifier**.
4. L’aperçu indique la version, la compatibilité, la signature reconnue et les accès. Clique sur **Installer ce fournisseur**.
5. La fiche TMDB s’ouvre et affiche **Paquet installé · 0.4.0**. Active le module si tu souhaites l’utiliser.

Le premier paquet est installé désactivé. Ta connexion TMDB est réutilisée : pas besoin de recopier ta clé, ni de rescanner ta collection. Les recherches et fiches détaillées passent par le programme importé quand le module est actif.

## Essayer une mise à jour sans remplacer Mosaune

1. Dans la fiche TMDB, clique sur **Installer une mise à jour**.
2. Télécharge **la mise à jour 0.4.1**, puis sélectionne ce nouveau ZIP.
3. L’aperçu affiche **0.4.0 → 0.4.1**. Clique sur **Installer cette mise à jour**.
4. La fiche doit afficher **Paquet installé · 0.4.1**, tandis que le serveur reste en **0.0.19**.

Les deux paquets fournis ont les mêmes fonctions médias ; leur programme annonce une version différente au contrôle de démarrage. Ils permettent de vérifier le mécanisme complet avant de distribuer de nouvelles fonctions par ce moyen.

Mosaune vérifie le programme avant le remplacement. S’il ne démarre pas correctement, le paquet actuel reste en place. Une mise à jour réussie conserve l’état actif ou désactivé du module et garde une seule version de retour.

## Restaurer, désactiver ou désinstaller

- **Restaurer la version 0.4.0** : confirme la restauration. Elle est testée avant le remplacement ; la 0.4.1 devient ensuite la version de retour.
- **Désactiver le module** : le paquet reste installé, ses travaux en cours sont arrêtés et le fournisseur intégré reprend la main.
- **Désinstaller le fournisseur** : après confirmation, le paquet et sa version de retour sont supprimés. Les fiches, affiches, corrections et progressions restent dans ta bibliothèque. La clé reste disponible pour le fournisseur intégré qui la partage.

Ces opérations se font sans redémarrer le serveur. Le mode de secours permet toujours de démarrer sans exécuter le paquet installé ; il autorise sa désinstallation mais bloque l’installation et la restauration jusqu’au redémarrage normal.

## Périmètre de cette première livraison

Le pilote accepte les paquets TMDB signés par la **clé de développement Mosaune** reconnue par ce serveur. L’import de thèmes continue de fonctionner dans sa section habituelle. L’ajout de clés ou de fournisseurs communautaires, la boutique GitHub et les modules média installables restent à réaliser.

Le même paquet WebAssembly fonctionne avec les versions Windows et Linux de Mosaune, y compris la compilation Linux ARM64. L’exécution sur ton Pi reste à valider.
