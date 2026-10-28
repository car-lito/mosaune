# Comportements des fiches — media.detail-experience.v1

Un seul contrat extrait ensemble les commandes, liens d’artistes, listes de fichiers et pistes, rapprochement des pistes locales/distantes, format de couverture et présentation des suggestions. Le noyau reçoit une description, pas du JavaScript. Le module officiel contient les règles Films/Séries/Musique ; le rendu partagé n’effectue plus le rapprochement des pistes ni le choix des boutons selon le média.

## Contrat

- describe avec {} renvoie {"version":1}. project reçoit Query, défini dans experience.go.
- L’entrée HTTP contient seulement kind, workId, ids et trackId facultatif. L’hôte résout les données stockées et contrôle la visibilité de chaque fichier et son association à l’œuvre. Aucun chemin, contenu de fichier, clé ou URL n’entre dans le programme.
- Query : fichiers visibles et progression, titres/artistes/numérotation, pistes de l’édition, personnes par indices, contexte d’œuvre et fichier ciblé. Les personnes complètes restent côté hôte ; seuls leurs noms entrent dans le programme.
- Result : cover (square/portrait), actions, people (indices), sections ordonnées avec lignes et actions, episodeList (délégation à la présentation des saisons existante), albumLink et suggestions (activation, titre, description).
- Une action est une intention : operation, label, fileId éventuel, primary. Opérations : play, open, inspect, mark, poster, match, enqueue, playlist, watchlist, favorite, request, unavailable. Une référence de fichier doit appartenir à la fiche autorisée. Les commandes d’une ligne ne peuvent viser un autre fichier. Aucune URL, méthode HTTP ou instruction libre.
- Les libellés ne donnent aucun droit. Le rendu contrôle les permissions ; les services existants les vérifient aussi lors de l’action. Le module ne peut pas lancer seul une lecture, une modification ou un téléchargement : il faut une interaction de l’utilisateur.
- Les fichiers doivent tous apparaître exactement une fois dans les sections, ou être délégués à la liste de séries ; un contexte de morceau exige au minimum ce morceau. Les pistes absentes n’ont pas de fileId. Les doublons et références étrangères sont rejetés. En cas d’ambiguïté de titre/numéro, le module officiel conserve les fichiers dans « Autres fichiers » plutôt que de les associer au hasard.
- Les textes sont échappés. Le renderer emploie les composants et variables de thème existants : content-actions, primary/secondary, artist-links, album-tracks, track-title, episode-actions. Ordre, libellés et sections sont libres dans les bornes du contrat. Ajouter de nouvelles opérations ou composants exige une évolution du contrat ; aucun script arbitraire n’est accepté.

## Installation et limites

Mosaune 0.0.39 minimum, schéma de base 26, slot detail-experience, officiel mosaune.media.experience. Paquet exécutable schéma 2, une contribution/capacité media.detail-experience.v1, cible media, aucun service/dépendance. Type de dépôt detail-experience. Installation inactive, activation et sélection distinctes, mise à jour/restauration/retrait communs.

Désactiver le module supprime ses commandes, listes enrichies et suggestions, annule les projections en cours et refuse leurs réponses tardives. Une liste simple des fichiers visibles reste utilisable ; aucune autre extension ne prend sa place. Les comptes, métadonnées, progressions et fichiers sont conservés. La migration n’active pas les anciens modules désactivés et ne réinstalle pas un module retiré après migration.

1000 fichiers, 1000 pistes, 1000 personnes ; entrée JSON 512 Kio, sortie 768 Kio, 8 sections et 2000 lignes. Enveloppes de transport bornées à 1 Mio pour cette seule capacité ; les autres modules gardent leurs budgets habituels. Le SDK fournit ServeSized pour recevoir cette enveloppe sans retirer les limites. Moteur WASI à 64 Mio, 5 secondes maximum par invocation pour cette capacité (2 secondes restent la valeur par défaut des autres projections), projections sérialisées et moteur fermé après usage. Ni cache permanent ni processus supplémentaire au repos.

Les fournisseurs de suggestions et les services exécutant les actions restent des adaptateurs hôtes existants : ce lot extrait leur sélection et présentation dans la fiche, pas MusicBrainz, Radarr/Sonarr ou les moteurs de lecture. Les familles installables d’un bloc et les contributions de navigation restent à terminer.

## Exemple indépendant

[Lecture directe](../../../examples/detailexperience) utilise uniquement ce contrat et le SDK, sans importer les règles officielles. Il remplace toute la présentation par une liste compacte, choisit une couverture carrée et supprime les suggestions. Archive : build/example-detailexperience.mosaune.zip, produite par scripts/project.mjs et jointe aux artefacts GitHub. Les tests utilisent un dépôt simulé pour vérifier téléchargement, import, sélection, mises à jour et retrait.
