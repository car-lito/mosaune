# Essayer le dépôt officiel — Mosaune 0.0.21

Le dépôt est en ligne : https://github.com/car-lito/mosaune-extensions. Il contient les extensions Ambre et TMDB et leurs fiches de catalogue. Le serveur Mosaune et son futur conteneur CasaOS auront leur propre dépôt.

1. Fermez votre serveur Mosaune actuel et gardez une copie du dossier `data`. Lancez `DEMARRER-WINDOWS.cmd` dans ce dossier : il démarre la 0.0.21 avec vos données habituelles. Aucun nouveau changement du format de base depuis la 0.0.20.
2. Connectez-vous en administrateur, puis ouvrez **Réglages → Extensions → À télécharger**.
3. Cliquez sur **Charger le catalogue officiel**. Si vous l’aviez déjà ajouté, le bouton indique **Actualiser le catalogue officiel**. Pas besoin de compte GitHub ni de clé GitHub.
4. Dans le filtre **Source**, choisissez **Mosaune — extensions officielles · car-lito/mosaune-extensions**. Les autres paquets « Inclus avec Mosaune » sont les exemples embarqués, également disponibles sans Internet.
5. Ouvrez **Ambre** ou **TMDB**, puis **Vérifier ce paquet**. Mosaune télécharge le fichier depuis GitHub et vérifie sa compatibilité et son intégrité, ainsi que la signature pour TMDB.
6. Installez-le puis activez-le depuis sa fiche. Pour Ambre, **Utiliser pour moi** applique le thème à votre compte. Un paquet déjà installé à la même version ne demande pas de réinstallation.

Pour tester une mise à jour TMDB sur un serveur où aucun paquet TMDB n’est installé : choisissez 0.4.0, installez-la, puis revenez choisir 0.4.1. Les fonctions médias restent identiques : ces versions servent à vérifier installation, mise à jour et restauration. Ne désinstallez pas une version fonctionnelle juste pour effectuer ce test.

Le catalogue garde sa dernière copie en cas de panne GitHub. Les futurs paquets ne s’installent pas automatiquement. La source officielle reste visible, mais aucun contact avec GitHub ne se fait simplement en ouvrant les paramètres.
