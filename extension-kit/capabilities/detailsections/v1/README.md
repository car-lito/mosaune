# Rubriques de fiche — media.detail-sections.v1

Ce contrat extrait la présentation descriptive des fiches. Il ne remplace pas leur en-tête, leurs commandes, les listes de fichiers, les suggestions, les profils de personnes ni les lecteurs. Le programme officiel présente les films et séries ; il ne fournit aucune rubrique musicale pour le moment. Le contrat peut recevoir un autre kind sans changer le moteur de rendu, mais l’ajout de nouveaux types de média au serveur reste un chantier distinct.

## Données et composition

- describe avec {} renvoie {"version":1}. project reçoit Query, définie dans sections.go.
- Le serveur vérifie le type, la visibilité des fichiers et leur association à l’œuvre avant de préparer les faits enregistrés. Aucun appel de fournisseur n’est déclenché par cette route.
- Faits : kind, overview, releaseDate, status, countries, languages, cast, localCount, people (name/role), trailers (name/language). Aucun chemin, URL, identifiant de compte, secret ou contenu de fichier.
- Result contient blocks, un tableau obligatoire (vide autorisé). Chaque bloc possède une key unique, une region (main, aside, full), un kind et un title. L’ordre est conservé à l’intérieur de chaque région. Sur petit écran, la colonne secondaire suit la principale.
- paragraph : text ; facts : fields [{label,value}] ; people et trailers : references [{index,label}], avec text optionnel. Les indices pointent exclusivement vers les tableaux d’entrée. Aucun lien ni action arbitraire n’est accepté.
- Le rendu commun affiche les textes échappés et les composants du thème. Les portraits, liens de profils et vidéos proviennent des ressources conservées par le serveur. La navigation vers un profil utilise les contrôles habituels ; les liens vidéo n’acceptent que HTTPS, sans identifiants intégrés, et s’ouvrent dans un nouvel onglet protégé.
- Ces quatre composants constituent la première version du contrat, pas un langage de mise en page illimité. Ajouter un nouveau composant nécessitera une évolution du contrat et du moteur, avec compatibilité explicite. Aucun HTML/JavaScript arbitraire.

## Limites et cycle de vie

48 Kio d’entrée JSON, 96 Kio de sortie ; 24 blocs, 32 champs par bloc, 128 personnes, 32 vidéos. Résumés/textes longs : 16 Kio. Les références inconnues, doublons, régions/types inconnus et tableaux manquants sont refusés. Pas de troncature silencieuse : un dépassement affiche une explication, les fichiers restent accessibles et les données ne sont pas effacées.

WASI commun : 64 Mio de mémoire linéaire, 2 secondes par invocation, budget 15 secondes ; une présentation à la fois, moteur fermé après usage. Le retrait, changement de sélection ou arrêt annule les appels ; les résultats tardifs sont refusés avant publication. Pas de VM au repos ni de nouveau cache persistant. Une vue déjà chargée dans un autre onglet est actualisée à sa réouverture.

Paquet exécutable schéma 2, contribution unique media.detail-sections.v1 cible media, même capacité unique, aucun service/dépendance, Mosaune 0.0.37 minimum. Type de dépôt : detail-sections. Slot commun detail-information. Installation désactivée ; activation et sélection explicite séparées. Désactivation/retrait : rubriques masquées, aucune substitution automatique, fichiers/métadonnées/progressions conservés. Mise à jour et restauration via le moteur commun.

[Exemple indépendant](../../../examples/detailsections) : utilise seulement le contrat public et le SDK, affiche « L’essentiel » en pleine largeur. Archive build/example-detailsections.mosaune.zip produite par scripts/project.mjs et jointe aux artefacts GitHub.
