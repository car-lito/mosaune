# Revenir à la version précédente d’une famille — 0.0.49

Cette version ajoute une action dans **Réglages → Extensions → Films, Séries ou Musique**. Elle concerne les familles installées par archive ou depuis le catalogue.

1. Arrêtez votre ancien serveur, conservez une copie du dossier data, puis lancez DEMARRER-WINDOWS.cmd. Recharger la page seule ne met pas le serveur à jour.
2. Installez une première version de famille puis sa mise à jour depuis le catalogue. Pour un essai isolé, les exemples officiels 2.0.0 puis 2.1.0 conviennent.
3. Ouvrez la carte de la famille. **Revenir à la version …** apparaît lorsqu’un point de restauration existe.
4. Consultez les composants qui seront restaurés et ceux qui restent inchangés, puis confirmez.

Les fichiers, fiches, progressions, réglages, permissions et choix actif/désactivé sont conservés. Les composants partagés et les remplaçants choisis ne sont pas rétrogradés. Le point de restauration est enregistré sur disque et reste disponible après redémarrage.

Une modification séparée d’un programme encore choisi peut empêcher le retour : Mosaune explique le conflit et ne modifie rien. Un composant réinstallé pendant la mise à jour, dont l’ancienne version était déjà absente, bloque aussi sa restauration. Un changement effectué depuis l’aperçu impose de le refaire.

Un seul retour est conservé par famille, puis consommé après confirmation. Réinstallez la nouvelle version depuis le catalogue si vous changez d’avis. Retirer la famille retire aussi son historique ; désactiver la famille le conserve.

**Les anciennes mises à jour ne possèdent pas cet historique.** Il commence avec les mises à jour effectuées sous 0.0.49. Si vous avez déjà 2.1.0, ne désinstallez pas votre famille juste pour tester : attendez une prochaine mise à jour ou utilisez une installation d’essai distincte.

Le retour porte sur les programmes et l’écran effectivement mis à jour, pas sur une sauvegarde complète du serveur. Certaines fonctions métier et les lecteurs restent intégrés : cette étape ne rend pas encore Mosaune entièrement extensible. Aucun déploiement NAS nécessaire.
