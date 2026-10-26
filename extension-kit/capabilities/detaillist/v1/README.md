# Liste de fiche média v1

La capacité media.detail-list.v1 fournit une liste de fichiers ordonnée et des groupes optionnels. Première cible : la liste d’épisodes et les saisons des fiches Séries. Ce contrat ne remplace pas encore le haut de fiche, le casting, les suggestions, les pages films/musique ni leurs actions.

## Entrées et sorties

- describe avec un objet vide retourne {"version":1}.
- project reçoit Query : entries (id, title, season, episode) et seasons (number, name, episodes).
- Le serveur lit les données enregistrées et filtre les permissions avant l’appel. Aucun chemin, URL, secret, compte ou contenu de fichier n’est transmis. Les textes transmis sont des copies nettoyées et bornées ; les valeurs stockées ne changent pas.
- Result fournit title/all/files/help/empty, groups et rows. Les textes sont affichés comme texte, jamais comme HTML ou JavaScript. Le module choisit l’ordre.
- Chaque groupe contient key/title/total ; artwork est un index optionnel dans les saisons fournies. Le serveur conserve les références d’images et le client utilise les mêmes composants thémables.
- requestSeason est optionnel. Il doit désigner une saison connue dans les entrées ; il propose une action de demande via les contrôles habituels du serveur. Il ne lance aucun téléchargement à lui seul.
- Chaque ligne contient id/title/ordinal/group. Tous les identifiants reçus doivent apparaître exactement une fois. Aucun identifiant supplémentaire, doublon ou groupe inconnu n’est accepté. Le client ne résout que les fichiers actuellement visibles.

## Limites

Au maximum 1 000 fichiers, 128 saisons/groupes, 48 Kio de JSON d’entrée et 192 Kio de sortie. Textes de sortie de 1 Kio maximum, sans caractères de contrôle. Une fiche dépassant ces limites conserve la liste simple et affiche un message ; pas de troncature silencieuse des épisodes. La pagination des très grandes fiches reste à ajouter.

Moteur WASI commun : 64 Mio de mémoire linéaire (pas la RAM totale du serveur), 2 secondes par invocation, budget global 15 secondes. Une présentation calculée à la fois par catalogue ; moteur fermé après usage, pas de VM ni de cache de présentation au repos. L’arrêt, la sélection ou le retrait annulent les appels et leurs réponses tardives sont refusées avant publication. Les vues déjà ouvertes récupèrent l’état au prochain chargement ; la fenêtre qui modifie le module peut rafraîchir sa vue via l’événement Mosaune.

## Cycle de vie et exemple

Paquet exécutable schéma2, une contribution media.detail-list.v1 cible series, une seule capacité, aucun service ni dépendance. Type de catalogue GitHub : detail-list. Mosaune0.0.36 minimum. Installation inactive ; activation et sélection explicite séparées. Import local et dépôts passent par le même moteur que les autres modules, avec mise à jour, restauration et mode secours.

L’exemple examples/detaillist dépend uniquement du contrat public et du SDK. Il affiche les épisodes sans saisons, avec ses propres libellés. Archive générée : build/example-detaillist.mosaune.zip. Les règles officielles sont dans modules/detailseries, compilées uniquement dans modules/detail-series/recognizer.wasm ; elles ne sont pas liées au serveur.

La désactivation ou le retrait rend la liste individuelle accessible. Métadonnées, pochettes, progression et fichiers conservés. Aucun recours silencieux à la présentation officielle. La migration23 installe et sélectionne le nouveau module une seule fois ; les choix des autres modules ne changent pas.
