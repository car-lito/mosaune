# Images musicales — protocole v1, depuis Mosaune 0.0.55

Le même emplacement `music-results` prend en charge les recherches, leur interprétation et les images. Aucun service réseau, accès aux fichiers ou accès direct à la base n’est accordé au programme. Un nouveau paquet annonce `minCore: 0.0.55` et répond à `describe-visuals` avec `{"Version":1}`.

L’opération `visual` reçoit une `VisualQuery` :

- `Operation` : `artwork` ou `artist`.
- `Ref` : identifiant d’artiste, identifiant d’édition éventuellement suivi de `/identifiant-groupe`, ou `commons:nom.jpg` pour une image.
- `State` : état JSON opaque renvoyé par l’étape précédente, limité à 8 Kio. L’hôte ne dépend pas des noms d’étapes choisis par le créateur.
- `Status` et `Body` : statut et dernière réponse JSON, limitée à 512 Kio. Au premier appel, ces champs sont vides. Les octets d’une image ne traversent jamais le programme Wasm.

Le programme renvoie exactement l’un des résultats suivants dans `VisualStep` :

- `Fetch` : prochaine requête à exécuter, avec le nouvel état `State`.
- `Photo` : portrait identifié, accompagné de son fichier Commons, de son URL, de sa source et de ses crédits/licence. Uniquement pour `artist`.
- `Missing: true` : aucune image trouvée.

Les plans sont validés à la sortie du programme puis par le noyau. Les sources permises sont MusicBrainz (relations d’artiste et groupe d’une édition), Wikidata (propriété P18), Commons (informations d’image), Cover Art Archive (couvertures) et les serveurs d’images Wikimedia. Les domaines, chemins, paramètres et références sont vérifiés ; aucune URL arbitraire, méthode d’écriture ou clé API ne peut être fournie. Un plan d’image terminé avec HTTP 200 rend ses octets directement à l’hôte ; un 404 est renvoyé au programme pour qu’il choisisse son éventuel repli.

Le workflow est limité à six étapes et vingt secondes au total, en plus des délais du transport et du contexte appelant. Chaque appel Wasm garde son plafond de quatre secondes ; enveloppe de 1 Mio, sortie de 512 Kio et mémoire linéaire de 64 Mio inchangées. Une seule session couvre les étapes et les accès réseau. La désactivation annule cette session et les requêtes en cours. Le cache partagé appartient au serveur, jamais au fournisseur.

Les anciens paquets sans `describe-visuals` restent stockés mais ne récupèrent pas de nouvelles images. Les imports de nouveaux paquets doivent proposer les trois contrats `describe`, `describe-requests` et `describe-visuals`. La migration du composant fourni vérifie son identité réservée, l’absence d’import et l’intégrité du code enregistré ; elle respecte son activation et la sélection d’un autre composant.

L’implémentation fournie est dans `modules/musicresults/visuals.go`, uniquement compilée en Wasm. Elle gère les replis de pochettes, le parcours MusicBrainz → Wikidata → Commons et le nettoyage des crédits HTML. Les types et contrôles des plans se trouvent dans `visuals.go` du présent contrat. Ajouter une nouvelle source réseau exige encore un contrat et une autorisation de l’hôte : le workflow est remplaçable, le fournisseur complet n’est pas encore indépendant.
