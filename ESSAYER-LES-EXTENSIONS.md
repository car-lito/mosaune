# Essayer les extensions — Mosaune 0.0.14

> Guide historique de la 0.0.14. Pour la version actuelle, suivre [ESSAYER-LES-THEMES.md](ESSAYER-LES-THEMES.md) : Ambre 2.0.0, paramètres inclus, typographie et arrondis, limite portée à 256 Kio.

Cette étape ajoute de vrais paquets de thèmes : ils s’installent depuis un fichier, sans modifier ni recompiler le programme. Les fournisseurs, connecteurs et nouveaux types de médias ne sont pas encore des paquets installables. Ils continuent de fonctionner comme avant.

## 1. Mettre à jour

1. Arrête le serveur avec **Ctrl+C** dans sa fenêtre, puis garde une copie du dossier **data**.
2. Double-clique sur **DEMARRER-WINDOWS.cmd**, puis recharge Mosaune. Vérifie **0.0.14** dans la barre latérale.
3. Connecte-toi avec ton compte existant. Tu n’as pas à recréer ton administrateur.

Le catalogue passe au format 10. Mosaune crée aussi une sauvegarde SQLite `mosaune-before-extensions-…db` dans data avant cette migration. Pour revenir à une ancienne version, utilise la copie du dossier data faite avant la mise à jour, serveur arrêté. Ne lance pas un ancien exécutable sur la base mise à jour.

## 2. Installer ton premier paquet

Dans **Réglages → Extensions**, avec ton compte administrateur :

1. Clique sur **Télécharger le thème d’exemple Ambre**. Le fichier est fourni par ton propre serveur ; aucun compte GitHub ni accès Internet n’est nécessaire.
2. Dans **Paquet à vérifier**, sélectionne ce fichier ZIP. Tu peux aussi utiliser `examples/ambre/atelier-ambre-1.0.0.mosaune.zip` dans le dossier du projet.
3. Vérifie le nom, la version et les accès demandés, puis clique sur **Installer ce paquet**.
4. Dans sa carte, clique sur **Activer**, puis **Utiliser pour moi**.
5. Retourne à **Accueil**, Films ou Musique pour voir la palette Ambre.

Les paramètres et la connexion gardent volontairement leur apparence native. Le thème modifie les couleurs de l’interface de consultation ; il ne change pas encore la disposition des pages, les icônes ni les lecteurs.

Pour voir sa variante sombre, choisis **Sombre** dans **Réglages → Mon compte**, puis retourne à l’accueil. Le réglage **Suivre l’appareil** fonctionne aussi.

## 3. L’autoriser pour un autre compte

1. Dans la carte Ambre, ouvre **Accès des utilisateurs**.
2. Coche un utilisateur existant, puis **Enregistrer les accès**. Les administrateurs ont toujours accès aux paquets.
3. Connecte-toi avec cet utilisateur dans une fenêtre privée. Dans ses **Réglages → Extensions**, choisis **Utiliser pour moi**.

Chaque compte garde son choix. Installer ou choisir un thème pour toi ne l’impose pas aux autres. Une extension nouvellement installée n’est pas autorisée automatiquement pour les comptes ordinaires.

## 4. Désactivation, retrait et mise à jour

- **Désactiver** : les comptes reviennent à l’apparence Mosaune. Le paquet, les permissions et les préférences sont conservés. En le réactivant, les comptes qui l’avaient choisi le retrouvent.
- **Rétablir pour moi** : seul ton compte revient à l’apparence Mosaune ; les autres gardent leur choix.
- **Désinstaller** : un aperçu explique ce qui sera retiré. Après confirmation, le paquet, ses autorisations et ses choix sont supprimés. Les bibliothèques, métadonnées, affiches, médias et playlists sont conservés. Une réinstallation nécessite de redonner les autorisations.
- **Mettre à jour** : importe un paquet avec le même identifiant et le même éditeur déclaré, mais une version supérieure. L’écran montre les deux versions. L’installation remplace le paquet en une opération et le désactive ; réactive-le après vérification. Les permissions et préférences restent conservées. Pour ce format, les versions identiques et les rétrogradations sont refusées.

Le changement se voit immédiatement dans l’onglet où tu le fais. Les autres onglets ouverts le prennent en compte sous environ 30 secondes lorsqu’ils sont visibles, ou quand tu reviens dans la fenêtre. Une lecture en cours n’est pas relancée par le mécanisme de thèmes.

## 5. Mode de secours

Si tu souhaites démarrer sans appliquer aucun paquet externe, ferme le serveur normal puis double-clique sur **DEMARRER-SECOURS-WINDOWS.cmd**. Le mode de secours garde les paquets et les préférences mais empêche leur activation. Les comptes et leurs permissions restent obligatoires.

Pour le quitter, ferme cette fenêtre et relance **DEMARRER-WINDOWS.cmd**. Tu peux également désinstaller un thème depuis les paramètres en mode de secours.

## Ce que cette étape prépare

Un paquet possède un identifiant, une version, une compatibilité, une capacité requise, des permissions et un état. Son installation et son usage sont deux droits différents. Le stockage reste géré par Mosaune.

Les thèmes de cette première version ne contiennent que des couleurs, vérifiées pour le format et le contraste des principales paires de texte/fond. Ils n’exécutent aucun code et n’accèdent ni aux fichiers, ni au réseau, ni au catalogue. Limites : 64 Ko par paquet, 32 paquets installés.

L’import local ne certifie pas l’identité de l’auteur. Le nom d’éditeur est déclaratif ; une empreinte SHA-256 détecte un contenu différent, mais ne remplace pas une signature. Même notre exemple est présenté comme un import local, pas comme un paquet authentifié par une boutique.

Les dépôts GitHub, signatures, langues, menus supplémentaires et extensions exécutables restent les prochaines étapes. Les modules intégrés listés en bas de l’écran ne sont pas présentés comme désinstallables.

Pour créer ton propre paquet : [CREER-UNE-EXTENSION.md](CREER-UNE-EXTENSION.md).
