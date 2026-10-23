# 0.0.30 — gérer les modules de reconnaissance

Dans **Paramètres → Extensions → Sur le serveur**, trois nouvelles cartes contrôlent la reconnaissance des films, des séries et de la musique. Ces modules identifient les formats et les noms pendant les scans ; ce ne sont pas encore les extensions complètes des trois familles de médias.

## Essai sur un petit dossier

1. Ouvrez **Reconnaissance des films** et désactivez-la. L’actualisation d’un dossier de films doit expliquer que le module est indisponible. Les médias déjà enregistrés restent dans la collection et lisibles.
2. Réactivez-la : l’actualisation fonctionne de nouveau. Les scans de musique et de séries restent indépendants.
3. Vous pouvez aussi désinstaller ce module depuis sa fiche. La confirmation précise que le catalogue est conservé. Après un redémarrage, le module reste absent.
4. Retrouvez-le dans **À télécharger → Modules fournis avec Mosaune** : il se réinstalle sans téléchargement réseau, puis demande une activation explicite.

L’installation ou la mise à niveau initiale conserve le fonctionnement actuel en installant les trois modules. Ce choix initial n’est effectué qu’une fois : un module retiré n’est jamais recréé au démarrage suivant. La migration sauvegarde le catalogue existant.

La désactivation interrompt les analyses qui n’ont pas commencé leur enregistrement. Si l’écriture du catalogue a déjà commencé, elle se termine avant la modification d’état. Aucun ancien résultat annulé ne peut être enregistré après réactivation. Une erreur de retrait laisse l’état précédent intact.

## Limites de cette étape

- Les paquets installés et leur état utilisent le registre commun des extensions. Leur programme installé est supprimé lors du retrait ; la copie de distribution, incluse dans le serveur, permet la réinstallation hors ligne. Le fichier de base de données ne rétrécit pas nécessairement immédiatement après suppression.
- Seules les trois versions officielles fournies sont installables par ce parcours. L’import de reconnaisseurs communautaires et le choix entre plusieurs implémentations restent à faire.
- Désactiver la reconnaissance ne désactive pas les pages Films/Musique, les fournisseurs de métadonnées ni les lecteurs. Ces fonctions doivent encore être extraites ; leurs cartes « intégrées » restent clairement distinctes.
- Le mode de secours suspend aussi ces reconnaisseurs. Aucun programme de reconnaissance ne reste chargé au repos.
- Le noyau conserve les fichiers et les métadonnées communes. Les modules reçoivent uniquement des noms relatifs ; ils n’ont accès ni aux fichiers, ni au réseau, ni aux clés API.

Aucun déploiement sur le NAS à cette étape. Il n’est pas nécessaire de rescanner toute la collection pour essayer ces commandes.
