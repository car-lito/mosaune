## Imports externes — 0.0.31

Le même contrat peut être fourni par un paquet local externe, installé et choisi explicitement pour un type de média. Mise à jour, restauration et retrait passent par le registre partagé. [Exemple contributeur](../../../examples/recognition/README.md) et [guide](../../../../ESSAYER-LES-EXTENSIONS-031.md). Les reconnaisseurs externes ne peuvent demander ni service ni dépendance dans cette version. Le catalogue de familles reste fixe ; les écrans ne sont pas encore extraits.

## Cycle des paquets officiels — 0.0.30

Le serveur résout désormais le programme installé et son état dans le registre partagé. Le scan ne charge jamais directement la copie de distribution. Installation/réinstallation officielle, activation, désactivation et retrait sont disponibles dans les paramètres ; les sessions annulées ne peuvent plus enregistrer de résultats. [Parcours et limites](../../../../ESSAYER-LES-EXTENSIONS-030.md). L’import externe et la sélection entre implémentations restent à généraliser.

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

Ces programmes sont **distribués avec le serveur**, compilés séparément par `scripts/project.mjs` et Docker. Depuis 0.0.30, leur copie installée et leur état sont gérés dans le registre commun et le stockage privé `recognition_packages` (schéma 18). Le retrait efface cette copie installée ; la copie de distribution permet une réinstallation hors ligne, désactivée par défaut. Le mode de secours suspend leur exécution. Leur gestion est réservée aux administrateurs. L’import de versions communautaires n’est pas encore proposé.

## Prochaines extractions

Généraliser les sources de paquets et la sélection entre implémentations ; extraire les tags et les comportements propres aux médias ; sortir leurs écrans du routage central. Les lecteurs viendront ensuite. Ne pas présenter la reconnaissance extraite comme un module Films/Séries/Musique entièrement extrait.
