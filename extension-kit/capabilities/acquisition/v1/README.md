# Préparation et suivi des demandes

Le rôle `acquisition.plan.v1` cible `video` et nécessite Mosaune 0.0.59. Une seule implémentation est sélectionnée pour Films et Séries. Le module fourni est dans `modules/acquisition/source` et fait partie des archives de familles 13.0/13.1 (Mosaune 0.0.64).

Le programme répond à `describe` avec une version de contrat égale à 1. L’opération `plan` reçoit le connecteur, l’identifiant distant, la présence d’un fichier, le périmètre des saisons et le type de série. Elle renvoie un `Plan`. Les structures et leur validation sont définies dans `acquisition.go`.

Un plan indique la création, la mise à jour ou l’absence d’écriture, les saisons à surveiller et les recherches à déclencher. Le serveur vérifie que les choix respectent la demande confirmée. Il applique le plan à la ressource complète pour conserver les champs distants inconnus. Les dossiers et profils existants sont protégés. Les clés, chemins et réponses HTTP complètes restent dans le serveur.

Le fichier `policy.json` fourni contient `searchMode`. La valeur `auto` déclenche la recherche habituelle et `none` active uniquement la surveillance. Pour créer un remplaçant, modifiez le programme ou cette politique, compilez le module pour `wasip1/wasm` et empaquetez-le avec `cmd/package-module`. Le manifeste doit déclarer le rôle ci-dessus avec la cible `video`, un identifiant propre et aucune dépendance ou service.

L’installation utilise le cycle de vie commun des programmes. L’activation et la sélection sont explicites. Le retrait conserve les demandes enregistrées. Une confirmation préparée avant le remplacement doit être vérifiée à nouveau. La désactivation annule les connexions actives et bloque les nouvelles connexions du connecteur. Le plan ne rejoue aucune opération après une erreur d’envoi.

Le type de paquet de dépôt est `acquisition`. Les archives de familles peuvent inclure ce composant sous l’emplacement `acquisition` avec `shared` activé. Le stockage de configuration et la coordination métier restent à extraire.

## Suivi, depuis Mosaune 0.0.60

Le module 0.2.0 conserve le même rôle et ajoute `tracking-info` (réponse `{"Version":1}`) et `track`. Ce dernier reçoit `Tracking` et renvoie `TrackingResult`, définis dans `tracking.go`. Les observations contiennent uniquement des identifiants internes au service, saisons, épisodes, tailles et états : aucun titre, chemin, clé, adresse ou identifiant de téléchargement. Le serveur lit au plus 20 pages de 100 éléments et 5 000 épisodes, avec une entrée Wasm plafonnée à 1 Mio, 64 Mio de mémoire et huit secondes par appel. Ce délai couvre le cas maximal dans l’interpréteur sur les machines plus lentes ; les limites de taille et de mémoire restent fixes.

Le programme décide des états et messages, de la pertinence des entrées de file et du calcul de progression. `waitingMessage` dans `policy.json` permet un exemple simple de personnalisation. Les compteurs sont vérifiés contre les observations du périmètre confirmé ; `available` nécessite réellement tous les fichiers locaux lisibles. Le serveur peut appeler `track` une seconde fois après le contrôle des fichiers. Le module ne déclenche aucun téléchargement ni scan.

Déclarez `minCore: "0.0.60"` pour un paquet qui fournit le suivi : ces opérations seront vérifiées à l’import. Les anciens paquets restent compatibles avec l’envoi, mais l’absence de `tracking-info` suspend le suivi avec une explication, sans repli sur une implémentation native. La mise à jour du module officiel ne modifie ni une sélection tierce, ni une désactivation, ni un retrait.

## Transport, depuis Mosaune 0.0.61

Le module 0.3.0 ajoute `describe-transport`, qui reçoit `{"connector":"radarr"}` ou `sonarr` et retourne `Transport` (`transport.go`). Le fichier `modules/acquisition/source/transport.json` est un exemple complet. `minCore: "0.0.61"` exige deux profils valides à l’import. Un ancien programme sans cette opération ne provoque aucun appel réseau ni repli natif.

Chaque profil déclare un préfixe relatif, un délai de 1 à 12 secondes, une réponse de 1 octet à 8 Mio et les routes nommées : status, profiles, roots, find, lookup, queue, create, update, search ; Sonarr ajoute episodes, episodefiles, monitor. Les méthodes sont fixes par opération (GET, POST ou PUT). Les chemins et les paramètres peuvent être modifiés dans le périmètre du serveur configuré. Seuls les identifiants et numéros de page fournis par Mosaune entrent dans les gabarits `{id}` et `{page}`. Les paramètres sont encodés séparément ; aucun chemin absolu, pourcentage, traversée, en-tête ou autorité réseau n’est accepté.

Le serveur conserve l’adresse, son éventuel sous-chemin, la clé et les objets à envoyer. Le programme ne reçoit ni ces objets ni les secrets. Il ne peut transformer une lecture en écriture, ajouter des opérations ou suivre une redirection. Le profil est conservé uniquement pendant la session en cours et invalidé avec le choix du composant. L’import et la sélection d’un programme lui confient le routage de ces opérations sur le service configuré ; ils ne donnent aucun accès direct au réseau.

Pour tester un autre préfixe d’API, changez `prefix` dans les deux profils, recompilez le programme et importez un paquet sous votre propre identifiant. Conservez les formats JSON compatibles avec Radarr/Sonarr : leur sérialisation reste actuellement dans le serveur. Les tests du dépôt couvrent routes remplacées après redémarrage, sous-chemin conservé, limites effectives, désactivation et absence de substitution silencieuse.

## Formulaires, depuis Mosaune 0.0.62

Le composant 0.4.0 ajoute `describe-settings`, avec l’entrée publique `{"connector":"radarr"}` (ou sonarr). La réponse Settings de settings.go déclare titre, instructions et cinq contrôles liés aux sources nommées url, key, profile, root, library. Voir modules/acquisition/source/settings.json. Chaque contrôle apparaît exactement une fois ; les contrôles de connexion précèdent ceux de destination, dans l’ordre fourni pour chaque étape. Les limites d’adresse et de clé sont définies par le composant dans les plafonds de sécurité de l’hôte (2048 et 200 caractères). La sélection des options est toujours vérifiée auprès du service et des bibliothèques autorisées.

Le programme ne reçoit aucune valeur saisie, clé, adresse, chemin ou option privée. Les libellés restent du texte échappé, les styles ceux du thème. L’hôte fournit les entrées URL/mot de passe et les listes dynamiques : un binding ne permet pas de lire un autre stockage ou d’ajouter un accès réseau. Ce contrat spécialisé réutilise le cycle de vie des demandes ; il ne remplace pas le contrat général configuration des autres extensions. Des contrôles supplémentaires nécessiteront une évolution versionnée du contrat.

GET /api/v1/acquisition/settings retourne form et formRevision par connecteur, ou formError si indisponible. Les clients renvoient formRevision (sans form ni formError) avec POST test et PUT settings. L’empreinte couvre le programme sélectionné, le connecteur et le formulaire. Un formulaire périmé ou absent est refusé avant le réseau. Une désactivation interrompt la session ; la pause reste possible. Les formulaires ne sont pas stockés avec les clés. Un paquet minCore 0.0.62 doit fournir deux formulaires valides ; les anciens paquets exigent une mise à jour pour modifier les réglages, sans formulaire natif de secours. La migration42 conserve retraits, désactivations et remplaçants. Familles11.0/11.1.

## Formats JSON, depuis Mosaune 0.0.63

Le composant 0.5.0 fournit describe-wire, avec la même entrée publique connector. Il retourne WireFormat (wire.go) : champs JSON par objet et noms des commandes de recherche. Voir modules/acquisition/source/wire.json. Un paquet minCore 0.0.63 doit déclarer des formats valides pour les deux connecteurs. L’ancien paquet sans ce contrat ne peut pas utiliser de format natif de secours : aucun échange réseau n’est lancé.

Les objets status/profile/root/resource/file/queue/row, ainsi que season/episode/monitor pour Sonarr, forment le vocabulaire public des demandes. Leur structure et leurs valeurs sémantiques restent définies par ce contrat versionné ; l’auteur choisit le nom distant de chaque champ. externalId représente l’identifiant du fournisseur, resourceId celui de la ressource dans une file. Les objets imbriqués addOptions, movieFile, seasons et records sont traduits récursivement. Les noms de recherche movie/series/season correspondent aux commandes déclarées. Aucun champ nécessaire ne peut être omis, fusionné avec un autre ou défini deux fois. Ce contrat couvre les API compatibles avec ces structures, pas un langage de transformation arbitraire ; une autre structure nécessitera une nouvelle version du contrat.

Le serveur applique cette déclaration aux données dans les deux sens. Le programme ne reçoit pas les réponses, adresses, clés, chemins ni objets à écrire. Les champs inconnus sont conservés en valeur, y compris les entiers supérieurs à la précision d’un flottant. L’ordre et les espaces du JSON ne sont pas conservés. Les clés dupliquées, collisions entre champs connus et inconnus (y compris la casse), formes incompatibles et commandes non déclarées sont refusées. Plafonds : 8 Mio, profondeur48 et200000 valeurs, en plus de la limite réseau de chaque connecteur. Ces règles n’autorisent aucun nouvel appel : le plan confirmé, les méthodes et les identifiants restent contrôlés par l’hôte. Choisir un composant lui confie l’adaptation à l’API du service configuré, comme pour les routes.

Le format est chargé avec le transport une fois par session et invalidé avec son composant. La migration43 met à jour uniquement le composant fourni intact, conserve les états et respecte les remplaçants. Familles12.0/12.1 ; les archives antérieures restent immuables. Les tests remplacent les champs d’identification, de surveillance et la commande de recherche d’une API fictive, redémarrent le catalogue, puis vérifient la conservation de la destination, du profil et d’un grand entier inconnu.

Il reste de la coordination métier et du stockage de configuration dans le serveur. Cette étape ne constitue pas encore un connecteur autonome complet.

## Programme d’envoi, depuis Mosaune 0.0.64

Le composant 0.6.0 ajoute workflow-info (Version1), et submission-workflow, qui reçoit Query et renvoie Workflow. L’entrée reste limitée aux identifiants numériques, saisons et indicateurs déjà utilisés par plan ; aucune clé, adresse, destination ou ressource complète n’est envoyée au programme. L’opération plan reste disponible pour les clients précédents.

Workflow contient le plan et jusqu’à trois actions ordonnées create/update, monitor, search. Le composant construit les champs à écrire, les champs à retirer à la création et les références privées. Valeurs de type Value : soit literal JSON, soit ref symbolique. Références disponibles : destination.profile, destination.root, resource.id, resource.ids, requested.season et selected.episodes. L’hôte vérifie que chaque référence est utilisée au bon endroit ; fournir un identifiant littéral, changer le chemin/profil d’une ressource existante, modifier une saison non demandée, ignorer une étape requise ou ajouter une action est refusé avant le premier échange.

La validation n’est pas un programme de secours : aucun scénario n’est exécuté sans réponse du composant installé. L’exécuteur résout les valeurs privées, préserve les champs inconnus et s’arrête au premier échec. La surveillance filtre les épisodes sur les saisons du plan, refuse les identifiants invalides/dupliqués et reste limitée à5000épisodes. La préparation copie la ressource et ses saisons sans muter les objets de départ. Le composant peut choisir de ne pas lancer de recherche via sa politique ; il ne peut pas remplacer un plan confirmé par une opération plus large.

Un paquet minCore0.0.64 doit déclarer workflow-info version1 ; chaque réponse submission-workflow est validée intégralement lors de l’envoi. Un ancien paquet peut être conservé mais doit être mis à jour avant un nouvel envoi. Le suivi et la pause gardent leurs contrats distincts. Migration44 conserve désactivations/retraits/remplaçants ; familles13.0/13.1, archives précédentes inchangées. Le SDK contient les règles d’autorisation et les utilitaires de copie/résolution ; la construction du programme reste dans modules/acquisition/source/workflow.go.

Le stockage des réglages, la préparation des observations de suivi et certaines étapes de prévisualisation restent dans le serveur. Les connecteurs ne sont pas encore des familles entièrement autonomes.


### Plan de collecte (serveur 0.0.65, composant fourni 0.7.0)

`observation-plan(ObservationQuery) → ObservationPlan` reçoit uniquement le connecteur et l’identifiant distant. Version1 : une ressource absente (ID0) ne déclenche aucune lecture ; un film exige la file ; une série exige épisodes, saisons et file. Le composant choisit l’ordre des lectures et les limites (épisodes1–5000, file1–20pages de100). Ces exigences évitent de déclarer une disponibilité sur des preuves omises. Le manifeste fourni embarque ces choix dans `source/observations.json` ; un remplaçant peut les calculer dans son programme.

L’hôte valide le plan avant exécution. Les helpers `Begin`, `ObserveEpisodes` et `ObserveQueue` tournent côté hôte : ils projettent les objets canoniques vers `Tracking`, contrôlent types, identifiants, doublons et tailles. Les clés API, chemins, titres et identifiants de téléchargement restent privés. Une file dépassant la limite ou une page intermédiaire incomplète produit une erreur, pas un résultat partiel présenté comme complet. Une file évoluant pendant la pagination ne constitue pas un instantané atomique.

Un premier appel avec ID0 vérifie la disponibilité du plan avant la recherche distante. À l’import, les paquets annonçant minCore0.0.65 sont contrôlés pour les deux connecteurs, avec et sans ressource ; chaque résultat est également validé à l’exécution. Aucun plan natif de secours. Les contrôles de fichiers locaux, quotas de cadence, réglages persistants et prévisualisations restent encore dans le serveur ; cette étape ne rend pas le connecteur entièrement autonome.

### Réglages persistants — 0.0.66

L’adaptateur hôte `extension-kit/acquisitionconfig` utilise maintenant le stockage commun des extensions. Il est séparé des types compilés dans le Wasm : aucun accès SQL ni clé ne devient disponible au programme invité. Le protocole du composant reste compatible avec la version 0.7.0. Voir [le contrat de stockage](../../../acquisitionconfig/README.md).


### Préparation de l’aperçu — serveur 0.0.67 / composant 0.8.0

`preparation-plan(PreviewQuery{kind,season}) → PreviewPlan` est appelé avant la résolution d’identité et les échanges avec le service. Le programme ne reçoit ni chemins, ni titres, ni profils, ni clé. Le plan doit annoncer version1 et le connecteur compatible avec le média : radarr pour films, sonarr pour séries. Le protocole reste limité à ces deux familles de services.

`reads` contient exactement status, profiles, roots et find, sans doublons, avec status en premier. Les trois autres lectures peuvent changer d’ordre. `missing` vaut lookup ou reject : la deuxième politique refuse les ressources absentes sans lancer de recherche de fiche ni aucun ajout. Les textes missingMessage, seasonMessage et profilePrefix sont bornés, sans HTML exécuté. Le composant fourni déclare ces choix dans `source/preview.json` ; un remplaçant peut calculer un plan selon le média et la saison demandés.

Le serveur valide le plan puis exécute ses lectures avec le transport installé. `PreviewPlan.Project` tourne côté hôte : destination/profil existants pour une ressource connue, valeurs configurées pour une nouvelle ; saisons uniques, entières, bornées et limitées à la demande, triées pour un aperçu stable. Les clés et objets privés ne traversent pas le module. Une réponse de lookup contenant un identifiant interne déjà attribué est refusée pour éviter de confondre création et mise à jour.

Chaque plan est validé à l’exécution ; un paquet minCore0.0.67 est testé à l’import pour films, série entière et saison. Un ancien paquet peut être conservé mais ne prépare plus de demande sans cette opération. La confirmation est recalculée avant l’envoi et lie l’œuvre, le périmètre, les destinations, le programme et la révision des réglages. La bibliothèque locale, les permissions, la résolution d’identité du fournisseur et les identifiants secrets restent des responsabilités hôte. Les aperçus ne réalisent aucune écriture distante.

## Association des imports (cœur 0.0.68)

Le composant acquisition 0.9.0 ajoute deux opérations :

- `import-plan` reçoit PreviewQuery (kind/season) et fournit ImportPlan v1 : mode `verified` ou `deferred`, limit 1..5000, reads vide sauf `episodefiles` pour les séries en mode verified.
- `match-imports` reçoit ImportQuery : scope et candidats (slot, season, episode). Retour ImportResult avec les slots choisis. Le composant peut choisir un sous-ensemble ou différer l’association.

Les chemins, adresses, identifiants distants, clés et objets réseau restent dans l’hôte. ProjectImportCandidates est une réduction de preuves exécutée côté hôte, pas dans le module. Les fichiers/épisodes sont plafonnés à 5000 ; les identifiants de fichiers ou d’épisodes ambigus et fichiers multi-épisodes sont exclus. Les chemins sont vérifiés dans la destination exacte, les doublons de chemins exclus. Aucune sortie ne peut créer un candidat, choisir une autre saison ou répéter un slot. La session d’extension est annulable, et le protocole est vérifié à l’import des paquets minCore >=0.0.68 et à l’exécution. Aucun repli natif.

La famille conserve le même composant partagé, sans nouvelle carte visible. Les opérations physiques sur les fichiers et la vérification finale de leur disponibilité restent sous contrôle du serveur. Le contrat est encore limité aux objets canoniques films/séries ; il ne constitue pas un connecteur universel.
