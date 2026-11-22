# Manifestes de composition — schéma 1

Les trois répertoires media-films, media-series et media-music contiennent les manifestes compilés dans cette étape. Ces manifestes de composition ne sont pas eux-mêmes des archives. Depuis la 0.0.47, un format distinct permet de distribuer leurs composants : voir [FAMILY-ARCHIVES.md](FAMILY-ARCHIVES.md). Leurs identifiants sont stables ; version désigne la composition, pas une mise à jour automatique des programmes.

Champs : schema=1, id, name, version, kind (films/series/music), components (slot, composition, shared), screen (identifiant du paquet d’écran fourni).

Chaque slot correspond à un contrat déjà pris en charge par le registre. Seul composition=exclusive est accepté pour le moment. Le moteur utilise le programme explicitement sélectionné ; le programme fourni ne remplace pas silencieusement une sélection externe disparue. Les composants partagés sont déduits des appartenances croisées et conservés au retrait. shared décrit cette appartenance pour la présentation.

Le choix explicite d’un écran actif est mémorisé séparément ; désactivation et désinstallation n’effacent pas ce choix. Les paquets avec plusieurs pages sont vérifiés contre tous les propriétaires actifs ; leur retrait groupé est bloqué pour ne pas affecter une autre rubrique.

API d’administration : GET /api/v1/media-bundles ; POST /api/v1/media-bundles/{kind}/preview puis /apply. Envoyer operation (prepare/disable/remove) et revision ; apply exige aussi checksum retourné par preview. L’empreinte couvre le plan, les versions d’écran, leurs états et les autorisations. Révision ou empreinte périmée : 409, aucun changement.

Les changements de programmes, écran et enregistrement du regroupement partagent une transaction. Les appels en cours sont annulés uniquement après validation. La migration schéma 30 adopte l’état existant sans installer ni activer un programme. Les données médias sont conservées.

Import et mise à jour coordonnée par fichier sont disponibles en 0.0.47. Restent à définir : distribution distante, signatures/éditeurs de familles, résolution de contraintes inter-familles, restauration globale, contributions cumulables, propriétés des données privées et extraction des services métiers restants. Ne pas publier ce schéma comme un SDK universel d’extensions médias.

## Installation initiale — 0.0.76

Le démarrage classique installe Films, Séries, Musique et l’accueil comme auparavant. Sur un **nouveau dossier de données**, choisir les familles avec **-initial-media music**, **-initial-media films,series** ou **-initial-media none**. La variable MOSAUNE_INITIAL_MEDIA accepte les mêmes valeurs ; l’option en ligne de commande est prioritaire. Les valeurs inconnues, doublons et éléments vides sont refusés. **-initial-home=false** omet le paquet d’accueil ; l’accueil est indépendant des trois familles.

Ces options définissent seulement la première installation. Une fois le catalogue créé, utiliser les réglages Extensions pour ajouter, désactiver ou retirer des familles. Changer les options au redémarrage ne remplace pas les choix enregistrés. **-modules** reste un filtre d’exécution distinct, pas une sélection d’installation. Les fournisseurs sont composés séparément : **-initial-media none** ne retire pas TMDB.

Le point d’entrée interne openCatalogWithComposition reçoit les fabriques de fournisseurs et mediaBootstrapSelection. Avant les migrations, media_bootstrap mémorise les familles et l’accueil. Les migrations historiques reportent uniquement leur installation initiale, en conservant la création des tables ; leur parcours pour une base existante reste inchangé. Après les migrations, finishMediaBootstrap utilise planBundle et applyMediaBundlePlanTx, également utilisés pour l’administration normale.

Toutes les familles sélectionnées, leurs programmes partagés, écrans et marqueur de réussite sont validés dans une seule transaction. Une erreur conserve la sélection pour reprise, sans installation partielle. Un redémarrage avec d’autres valeurs par défaut n’ajoute rien. Une sélection enregistrée invalide provoque une erreur explicite. Les droits d’écran sont accordés par le même mécanisme que les paquets fournis ; les comptes créés ensuite ne reçoivent des droits que sur les écrans réellement installés.

Tests : chacune des trois familles seule, aucune, composition classique, partage des programmes, installation et retrait ultérieurs, panne sur la deuxième famille et reprise, conservation des pauses/retraits historiques et refus des compositions malformées. Le code fourni, les adaptateurs et les tables restent présents dans le serveur ; le noyau entièrement minimal reste à achever.
