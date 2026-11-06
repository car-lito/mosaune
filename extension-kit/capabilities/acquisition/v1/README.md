# Préparation et suivi des demandes

Le rôle `acquisition.plan.v1` cible `video` et nécessite Mosaune 0.0.59. Une seule implémentation est sélectionnée pour Films et Séries. Le module fourni est dans `modules/acquisition/source` et fait partie des archives de familles 10.0/10.1 (Mosaune 0.0.61).

Le programme répond à `describe` avec une version de contrat égale à 1. L’opération `plan` reçoit le connecteur, l’identifiant distant, la présence d’un fichier, le périmètre des saisons et le type de série. Elle renvoie un `Plan`. Les structures et leur validation sont définies dans `acquisition.go`.

Un plan indique la création, la mise à jour ou l’absence d’écriture, les saisons à surveiller et les recherches à déclencher. Le serveur vérifie que les choix respectent la demande confirmée. Il applique le plan à la ressource complète pour conserver les champs distants inconnus. Les dossiers et profils existants sont protégés. Les clés, chemins et réponses HTTP complètes restent dans le serveur.

Le fichier `policy.json` fourni contient `searchMode`. La valeur `auto` déclenche la recherche habituelle et `none` active uniquement la surveillance. Pour créer un remplaçant, modifiez le programme ou cette politique, compilez le module pour `wasip1/wasm` et empaquetez-le avec `cmd/package-module`. Le manifeste doit déclarer le rôle ci-dessus avec la cible `video`, un identifiant propre et aucune dépendance ou service.

L’installation utilise le cycle de vie commun des programmes. L’activation et la sélection sont explicites. Le retrait conserve les demandes enregistrées. Une confirmation préparée avant le remplacement doit être vérifiée à nouveau. La désactivation annule les connexions actives et bloque les nouvelles connexions du connecteur. Le plan ne rejoue aucune opération après une erreur d’envoi.

Le type de paquet de dépôt est `acquisition`. Les archives de familles peuvent inclure ce composant sous l’emplacement `acquisition` avec `shared` activé. La sérialisation des objets des API, les formulaires et leur configuration restent à extraire.

## Suivi, depuis Mosaune 0.0.60

Le module 0.2.0 conserve le même rôle et ajoute `tracking-info` (réponse `{"Version":1}`) et `track`. Ce dernier reçoit `Tracking` et renvoie `TrackingResult`, définis dans `tracking.go`. Les observations contiennent uniquement des identifiants internes au service, saisons, épisodes, tailles et états : aucun titre, chemin, clé, adresse ou identifiant de téléchargement. Le serveur lit au plus 20 pages de 100 éléments et 5 000 épisodes, avec une entrée Wasm plafonnée à 1 Mio, 64 Mio de mémoire et huit secondes par appel. Ce délai couvre le cas maximal dans l’interpréteur sur les machines plus lentes ; les limites de taille et de mémoire restent fixes.

Le programme décide des états et messages, de la pertinence des entrées de file et du calcul de progression. `waitingMessage` dans `policy.json` permet un exemple simple de personnalisation. Les compteurs sont vérifiés contre les observations du périmètre confirmé ; `available` nécessite réellement tous les fichiers locaux lisibles. Le serveur peut appeler `track` une seconde fois après le contrôle des fichiers. Le module ne déclenche aucun téléchargement ni scan.

Déclarez `minCore: "0.0.60"` pour un paquet qui fournit le suivi : ces opérations seront vérifiées à l’import. Les anciens paquets restent compatibles avec l’envoi, mais l’absence de `tracking-info` suspend le suivi avec une explication, sans repli sur une implémentation native. La mise à jour du module officiel ne modifie ni une sélection tierce, ni une désactivation, ni un retrait.

## Transport, depuis Mosaune 0.0.61

Le module 0.3.0 ajoute `describe-transport`, qui reçoit `{"connector":"radarr"}` ou `sonarr` et retourne `Transport` (`transport.go`). Le fichier `modules/acquisition/source/transport.json` est un exemple complet. `minCore: "0.0.61"` exige deux profils valides à l’import. Un ancien programme sans cette opération ne provoque aucun appel réseau ni repli natif.

Chaque profil déclare un préfixe relatif, un délai de 1 à 12 secondes, une réponse de 1 octet à 8 Mio et les routes nommées : status, profiles, roots, find, lookup, queue, create, update, search ; Sonarr ajoute episodes, episodefiles, monitor. Les méthodes sont fixes par opération (GET, POST ou PUT). Les chemins et les paramètres peuvent être modifiés dans le périmètre du serveur configuré. Seuls les identifiants et numéros de page fournis par Mosaune entrent dans les gabarits `{id}` et `{page}`. Les paramètres sont encodés séparément ; aucun chemin absolu, pourcentage, traversée, en-tête ou autorité réseau n’est accepté.

Le serveur conserve l’adresse, son éventuel sous-chemin, la clé et les objets à envoyer. Le programme ne reçoit ni ces objets ni les secrets. Il ne peut transformer une lecture en écriture, ajouter des opérations ou suivre une redirection. Le profil est conservé uniquement pendant la session en cours et invalidé avec le choix du composant. L’import et la sélection d’un programme lui confient le routage de ces opérations sur le service configuré ; ils ne donnent aucun accès direct au réseau.

Pour tester un autre préfixe d’API, changez `prefix` dans les deux profils, recompilez le programme et importez un paquet sous votre propre identifiant. Conservez les formats JSON compatibles avec Radarr/Sonarr : leur sérialisation reste actuellement dans le serveur. Les tests du dépôt couvrent routes remplacées après redémarrage, sous-chemin conservé, limites effectives, désactivation et absence de substitution silencieuse.
