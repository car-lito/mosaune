# Reconnaissance locale v1

Capacité `media.recognition.v1`, indépendante des fournisseurs de métadonnées et des lecteurs. Les trois programmes officiels sont dans `extension-kit/modules/films`, `series` et `music`. Le serveur importe leurs exécutables Wasm et leurs descripteurs, jamais le paquet Go qui interprète les noms.

## Opérations

- `formats`, avec `{}` : renvoie `{"extensions":[".mkv",".mp4"]}`. Entre 1 et 64 suffixes uniques en minuscules. Ce sont des formats indexables, pas une garantie de décodage par le lecteur.
- `recognize`, avec `{"paths":["Example/Saison 01/S01E02.mkv"]}` : renvoie `{"items":[{"accepted":true,"title":"Épisode 2","series":"Example","season":1,"episode":2}]}`.

Un lot contient entre 1 et 32 noms relatifs UTF-8 (1024 octets maximum par nom). Aucun chemin absolu, `..`, antislash, deux-points ou octet nul. Une réponse contient exactement un résultat par entrée, dans le même ordre. Un fichier non accepté produit `{ "accepted": false }`, sans métadonnées. Le serveur vérifie la concordance avec les formats annoncés ainsi que les tailles et les nombres. L'extension ne choisit jamais un identifiant de fichier, un chemin de sortie, une bibliothèque ou une requête SQL.

Champs de reconnaissance facultatifs : `year`, `series`, `season`, `episode`, `episodeEnd`. Ce contrat décrit l'identification initiale par nom. Il ne prétend pas définir toutes les propriétés des futurs médias : de nouvelles capacités versionnées peuvent fournir d'autres modèles sans élargir arbitrairement ce message.

## Exécution et limites actuelles

Un moteur temporaire par scan, un module chargé, appels par lots. L'hôte choisit le mode interprété pour ces programmes courts : il évite la compilation native initiale qui dépassait le budget sous émulation ARM64. Les fournisseurs existants conservent leur mode habituel. Aucun dossier monté, environnement personnel, service réseau, secret ou base de données exposé au module. Le moteur est fermé en fin de scan. Limites hôte inchangées : mémoire linéaire 64 Mio, appel 2 secondes, préparation 15 secondes, entrée 64 Kio, sortie 256 Kio. Ce ne sont pas des mesures de RAM totale : préparation et moteur consomment aussi de la mémoire.

Les réponses sont validées avant la transaction d'indexation. Une erreur conserve la bibliothèque précédente. Le serveur continue à ouvrir les dossiers et à appliquer les permissions, les limites de lecture et les transactions. Les tags audio sont encore extraits par le serveur ; ils enrichissent le titre initial fourni par Musique.

Ces programmes sont **livrés avec le serveur**, compilés séparément par `scripts/project.mjs` et Docker. Ils ne sont pas encore des paquets installables ou supprimables dans la boutique. Le mode de secours des paquets externes ne les désactive pas. Aucune nouvelle permission utilisateur et aucune migration de catalogue dans cette étape.

## Prochaines extractions

Raccorder ces capacités aux paquets média, à leur cycle d'installation et à leurs dépendances ; extraire les tags et les comportements propres aux médias ; sortir leurs écrans du routage central. Les lecteurs viendront ensuite. Ne pas présenter la reconnaissance extraite comme un module Films/Séries/Musique entièrement extrait.
