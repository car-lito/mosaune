# Mosaune 0.0.5 — préparer son écoute

**Disposition mise à jour en 0.0.7 :** les playlists ont leur onglet et une grande fiche avec pochette personnalisable ; la reprise musicale à la seconde est désormais disponible. Pour le parcours actuel, lire ESSAYER-LA-REFONTE.md. Le texte ci-dessous décrit l’atelier 0.0.5.

Arrête Mosaune dans sa fenêtre avec **Ctrl+C**, puis relance **DEMARRER-WINDOWS.cmd** et recharge la page. Sur une grande fenêtre, la barre latérale doit afficher **v0.0.5**. Tes dossiers, affiches, réglages TMDB et progressions vidéo restent dans le même catalogue.

Avant ce premier lancement, serveur arrêté, copie le dossier **data** ailleurs pour garder un point de retour. Cette version met à jour le format du catalogue ; les anciens exécutables ne pourront pas ouvrir le catalogue mis à jour.

## La file d’écoute

Une **file** est la liste des morceaux qui vont jouer maintenant. Une **playlist** est une sélection que tu gardes pour plus tard. Lire une playlist crée une nouvelle file : réorganiser celle-ci ne modifie donc pas la playlist enregistrée.

1. Ouvre **Musique**, puis ton album **Grand Garçon**. **Lire** prépare l’album dans son ordre de pistes. Le MP3 et le FLAC restent deux entrées distinctes.
2. **À la file** ajoute un morceau à la fin, même si une autre musique joue déjà. **Album à la file** ajoute tout l’album. Un même morceau peut apparaître plusieurs fois.
3. Ouvre **File d’écoute**, en haut de la page ou dans le lecteur. Les flèches ↑/↓ déplacent une piste et × la retire de la file. Aucun fichier n’est supprimé. Retirer le morceau en cours passe à un autre morceau disponible ; vider la file arrête l’écoute musicale.
4. **Mélanger la suite** réorganise uniquement les morceaux après celui en cours. L’ordre affiché est l’ordre réellement joué. Ce bouton effectue un mélange à chaque clic ; ce n’est pas un mode permanent pour les futurs ajouts.
5. Le bouton de répétition alterne **Répétition : non**, **Répéter la file**, **Répéter ce morceau**. Le bouton suivant passe au prochain morceau même avec la répétition d’un morceau activée. Précédent revient à la piste précédente ; au début de la file, il recommence le morceau courant.
6. Change d’espace : l’écoute continue. Recharge la page : la lecture s’arrête, mais les morceaux issus de la bibliothèque restent dans **File d’écoute**. Clique **Reprendre la file** pour reprendre au début du morceau sélectionné. La seconde précise d’écoute musicale n’est pas encore conservée.

La file est conservée dans ce navigateur, pour cette adresse et ce port de Mosaune. Elle n’est pas synchronisée entre appareils ou onglets. Les fichiers choisis uniquement pour une session disparaissent au rechargement. Un morceau absent de la bibliothèque est signalé comme indisponible et ignoré lors du passage automatique au suivant.

## Les playlists

1. Dans un album, **Playlist** ajoute une piste à une playlist existante ou à une nouvelle. **Album dans une playlist** ajoute tout l’album.
2. Tu peux aussi choisir **Créer une playlist** dans la file, ou **Nouvelle playlist** dans l’espace Musique pour commencer une liste vide.
3. Donne un nom, par exemple « Les soirs tranquilles », puis **Enregistrer**.
4. Dans **Vos playlists**, sélectionne son nom. Tu peux la lire, ajouter son contenu à la file, la renommer, déplacer ses pistes et en retirer.
5. Ferme et relance Mosaune : la playlist doit toujours être là. Elle est enregistrée dans le catalogue du serveur, contrairement à la file du navigateur.

Les playlists ne conservent que les médias d’un dossier ajouté à la bibliothèque. La fenêtre d’enregistrement indique si des fichiers temporaires sont exclus. Les morceaux momentanément absents restent dans la playlist, avec leur nom, sans empêcher la lecture des autres. Les flèches et retraits enregistrent immédiatement la modification ; renommer demande un clic sur **Renommer**.

Supprimer une playlist demande une confirmation dans l’interface. Cela supprime seulement la liste, jamais les morceaux. Si une autre fenêtre a modifié la même playlist, Mosaune refuse l’écrasement et propose **Recharger les playlists**.

## Limites de cet atelier

Maximum 200 pistes par file ou playlist et 100 playlists. Noms de 80 caractères maximum. Bibliothèque encore mono-utilisateur. Lecture via les contrôles audio du navigateur, sans lecture sans coupure garantie, fondu enchaîné ni normalisation du volume. Pas encore de favoris, de synchronisation des files entre appareils ni de commandes Android/TV.

## Petit parcours de validation

- [ ] Lire le MP3 puis passer au FLAC sans arrêter la lecture en naviguant.
- [ ] Ajouter plusieurs pistes, les déplacer, retirer une piste et vérifier l’ordre joué.
- [ ] Aller près de la fin avec la barre du lecteur : vérifier l’enchaînement, puis les deux modes de répétition.
- [ ] Créer une playlist, la renommer et modifier son ordre.
- [ ] Recharger : retrouver la file sans démarrage automatique, puis **Reprendre la file**.
- [ ] Redémarrer le serveur : retrouver la playlist, les affiches et les progressions vidéo.

Les tests automatiques de logique et du serveur sont passés. Ce parcours sert à vérifier ensemble les nouveaux boutons et la lecture dans ton navigateur réel.
