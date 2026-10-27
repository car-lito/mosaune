# En-têtes — media.detail-header.v1

Ce contrat compose les textes du haut des fiches Films, Séries et Musique. Il est indépendant de media.detail-sections.v1 et media.detail-list.v1 : leurs programmes et contrats existants ne changent pas. Il ne remplace pas les affiches, les liens d’artistes, les commandes, les listes, les suggestions ni les lecteurs.

## Entrée et résultat

- describe avec {} renvoie {"version":1}. project reçoit Query (header.go).
- Le navigateur fournit uniquement kind, workId, ids et trackId facultatif. Le serveur contrôle la visibilité, le type, les doublons et l’association des fichiers à l’œuvre. trackId doit appartenir aux fichiers autorisés de cette fiche musicale. Les titres soumis par le client sont refusés.
- Faits enregistrés : kind, track, storedWork, title, name (libellé du fichier), groupTitle, album, series, year, runtime en minutes, genres, rating sur 10, voteCount, provider, tagline, edition, localCount. Aucun chemin, URL, compte, secret ou contenu de fichier. Aucun appel de fournisseur.
- Pour une fiche locale sans œuvre associée, groupTitle provient de la projection du catalogue choisi, calculée sur les fichiers visibles sélectionnés. Sans ce module, les métadonnées enregistrées restent disponibles. Un morceau utilise ses propres métadonnées même si une œuvre album est associée.
- Result contient title (obligatoire), eyebrow, facts [{key,text,note}], tagline, footnote. facts est un tableau obligatoire, éventuellement vide. Les clés sont uniques ; leur ordre est l’ordre d’affichage. L’extension peut masquer les champs facultatifs, changer les libellés et ajouter des informations textuelles à partir des faits fournis.
- Le rendu utilise h1, eyebrow, content-facts, tagline et folder-help, donc les variables des thèmes existants. Les textes sont échappés. Les emplacements sont fixes dans cette version : aucune classe CSS arbitraire, URL, instruction HTML ou JavaScript ; ce n’est pas encore un langage de mise en page libre. Une évolution du contrat sera nécessaire pour de nouveaux composants.

## Cycle de vie et limites

Slot detail-header partagé, programme officiel mosaune.media.header, schéma de base 25. Paquet schéma 2 avec contribution unique media.detail-header.v1, cible media, même capacité unique, aucun service/dépendance. Mosaune minimum 0.0.38. Type de dépôt : detail-header.

Installation inactive, activation puis sélection explicite ; mise à jour et restauration communes. Désactivation, retrait, mode de secours ou résultat invalide : en-tête enrichi absent, avertissement et titre simple provenant de la carte déjà autorisée. Aucune autre extension n’est sélectionnée automatiquement. Les commandes, fichiers, affiches, métadonnées et progressions restent conservés. La migration installe ce nouveau module sans réactiver les modules précédemment désactivés ; il n’est pas réinstallé après un retrait.

Entrée 16 Kio JSON ; sortie 24 Kio ; 16 informations ; titres et textes longs 2 Kio ; texte d’une information 8 Kio ; note 500 octets ; 32 genres de 200 octets. UTF-8 sans caractères de contrôle, types et bornes numériques contrôlés. Jusqu’à 1000 identifiants locaux pour résoudre une fiche. Aucun tronquage silencieux.

Même moteur WASI que les autres projections : 64 Mio de mémoire linéaire, 2 secondes par invocation, budget 15 secondes une fois le créneau de présentation obtenu, moteur fermé après usage. Les projections de fiches sont sérialisées. La désactivation ou le remplacement annule les appels et refuse une validation tardive. Aucun cache ni VM supplémentaire au repos. Le navigateur annule les anciennes requêtes quand la fiche change ; un autre onglet déjà ouvert se met à jour à sa réouverture.

## Exemple indépendant

[Dans le cadre](../../../examples/detailheader) utilise seulement le SDK et ce contrat public. Il conserve le titre fourni, affiche une accroche différente et retire les informations secondaires. Son archive build/example-detailheader.mosaune.zip est construite avec le projet et jointe aux artefacts GitHub. Les tests exécutent réellement les deux programmes, puis vérifient sélection, mise à jour, restauration, retrait, migration et permissions.
