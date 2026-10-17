# Le dépôt GitHub de Mosaune

Adresse prévue : https://github.com/car-lito/mosaune

Le dépôt du serveur est distinct de [mosaune-extensions](https://github.com/car-lito/mosaune-extensions), qui distribue les paquets d'extensions.

## Les trois endroits utiles

1. **Code** : les fichiers du projet. Un « commit » est une étape enregistrée de leur évolution. Le bouton **History** permet de retrouver les modifications.
2. **Actions** : les vérifications automatiques. Un cercle jaune signifie « en cours », une coche verte « réussi », une croix rouge « une vérification a échoué ». Ouvrir l'exécution pour voir le contrôle concerné.
3. **Issues** : les problèmes et idées à suivre. Éviter d'y mettre des clés API, données de compte ou captures contenant des informations personnelles.

## Récupérer une version de test

Dans **Actions**, ouvrir une exécution réussie de « Vérifier et construire Mosaune », puis télécharger l'archive `mosaune-test-…` dans **Artifacts**. Il faut être connecté à GitHub. L'archive contient les versions Windows et Linux ainsi que les empreintes permettant de vérifier les fichiers.

Pour les essais actuels sous Windows, conserver le dossier `data` existant et faire une sauvegarde avant de changer d'exécutable. Ne pas lancer deux serveurs simultanément sur la même base.

## Ce qui vient ensuite

Le code publié ne met pas à jour automatiquement ton serveur. Les prochaines étapes sont l'accès réseau protégé, les images Docker et l'installation CasaOS. La procédure CasaOS devra correspondre à la version installée sur le NAS. Aucun déploiement sur le NAS n'est réalisé par les vérifications GitHub de cette première étape.
