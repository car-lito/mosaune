# Manifestes de composition — schéma 1

Les trois répertoires media-films, media-series et media-music contiennent les manifestes compilés dans cette étape. Ce ne sont pas des archives externes importables. Leurs identifiants sont stables ; version désigne la composition, pas une mise à jour automatique des programmes.

Champs : schema=1, id, name, version, kind (films/series/music), components (slot, composition, shared), screen (identifiant du paquet d’écran fourni).

Chaque slot correspond à un contrat déjà pris en charge par le registre. Seul composition=exclusive est accepté pour le moment. Le moteur utilise le programme explicitement sélectionné ; le programme fourni ne remplace pas silencieusement une sélection externe disparue. Les composants partagés sont déduits des appartenances croisées et conservés au retrait. shared décrit cette appartenance pour la présentation.

Le choix explicite d’un écran actif est mémorisé séparément ; désactivation et désinstallation n’effacent pas ce choix. Les paquets avec plusieurs pages sont vérifiés contre tous les propriétaires actifs ; leur retrait groupé est bloqué pour ne pas affecter une autre rubrique.

API d’administration : GET /api/v1/media-bundles ; POST /api/v1/media-bundles/{kind}/preview puis /apply. Envoyer operation (prepare/disable/remove) et revision ; apply exige aussi checksum retourné par preview. L’empreinte couvre le plan, les versions d’écran, leurs états et les autorisations. Révision ou empreinte périmée : 409, aucun changement.

Les changements de programmes, écran et enregistrement du regroupement partagent une transaction. Les appels en cours sont annulés uniquement après validation. La migration schéma 30 adopte l’état existant sans installer ni activer un programme. Les données médias sont conservées.

Restent à définir : archives, signatures/éditeurs de familles, résolution des versions et mise à jour coordonnée, contributions cumulables, propriétés des données privées et extraction des services métiers restants. Ne pas publier ce schéma comme un SDK universel d’extensions médias.
