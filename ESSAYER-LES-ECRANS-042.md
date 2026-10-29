# Écrans remplaçables — 0.0.42

Les pages Accueil, Films, Séries, Musique, Ma liste et Favoris sont maintenant fournies par quatre paquets d’interface. Leur composition n’est plus écrite dans un écran Dashboard du noyau : les paquets déclarent les rangées, leur ordre, leur présentation et les sélections à demander. Le rendu réutilise les composants et les variables des thèmes.

## Essai simple

1. Ouvrez **Réglages → Extensions → Sur le serveur**. Les quatre cartes **Écrans · Accueil et listes**, **Écrans · Films**, **Écrans · Séries**, **Écrans · Musique** sont actives au premier passage dans cette version.
2. Désactivez **Écrans · Films**. Le menu Films disparaît. La recherche, les fichiers et les fiches restent disponibles : c’est le paquet d’interface qui a été désactivé.
3. Dans **À télécharger**, cherchez **Cinéma compact**, installez-le puis activez-le. La nouvelle rubrique **Cinéma** affiche les films en grille, puis les favoris, sans sélections en ligne.
4. Pour revenir à la présentation habituelle, désactivez Cinéma compact et réactivez Écrans · Films.

Si vous tentez d’activer les deux en même temps, Mosaune vous indique le conflit et ne change rien. Un seul paquet actif peut posséder une rubrique. Les administrateurs ont accès aux pages ; les autres comptes doivent avoir accès au paquet et au type de média. Les accès des paquets fournis sont accordés lors de la migration et à la création d’un compte ; ils restent modifiables par l’administrateur.

Un paquet désinstallé reste absent après redémarrage. Retrouvez les écrans fournis dans À télécharger pour les réinstaller, puis activez-les et redonnez les accès aux comptes souhaités. Les réglages et la gestion des extensions restent accessibles même si tous les écrans sont désactivés. Il n’y a pas de copie cachée de l’ancien accueil qui reprend leur place.

## Ce que ce lot sépare — et ce qui reste

- Séparés : menus et composition des six écrans, reprise, mise en avant, ajouts récents, favoris, liste personnelle et rangées de découverte.
- Toujours communs : rendu des cartes, contrôles d’accès, accès aux données, stockage, styles et commandes de lecture.
- Encore à extraire : services de recherche/découverte, profils, gestion des playlists, formulaires et adaptateurs des commandes ; la logique fournisseur reste intégrée pour certains services.

Ces paquets sont des **interfaces médias**, pas encore les familles Films/Séries/Musique complètes. Couper Écrans · Films ne coupe donc pas TMDB ni Radarr. Le retrait d’un ensemble de composants ne retire pas automatiquement son écran à cette étape. Aucun nouveau lecteur ni déploiement NAS dans ce lot.

## Pour les créateurs

Les exemples réels se trouvent dans extensions/ui-films/extension.json et examples/cinema-compact/extension.json. Un paquet reste un ZIP contenant uniquement extension.json à la racine. Il passe par les mêmes routes d’import, validation, mise à jour, permissions et retrait que les autres pages. Les paquets fournis ne bénéficient d’aucun contournement pour leur activation.

Le contrat reste API 3, ui.pages.v1 ; les champs ci-dessous exigent minCore 0.0.42 ou plus :

| Champ | Usage |
|---|---|
| pages[].screen | Rubrique remplacée : home, films, series, music, watchlist ou favorites. Facultatif pour une page supplémentaire. |
| pages[].label | Libellé dans le menu. |
| pages[].heading | Titre de la page ; label est utilisé s’il est absent. |
| pages[].kinds | Types autorisés. Un écran films, series ou music doit déclarer exactement son type. |
| blocks[].type = feed | Rangée alimentée par les données accessibles au compte. |
| source | recent, resume, favorites, watchlist ou remote. |
| view | home (tous les types autorisés), films, series ou music. |
| display | shelf (par défaut), grid ou spotlight. |
| title / text | Titre et sous-titre de la rangée. |
| provider / shelf | Pour remote seulement : service et sélection déclarés ci-dessous. |

Sélections actuellement disponibles : tmdb / films / now_playing ou upcoming ; tmdb / series / popular ; musicbrainz / music / upcoming. La dernière consulte au maximum trois artistes issus des albums identifiés locaux ou favoris. Le fournisseur doit être activé et configuré. Il ne s’agit pas d’une autorisation d’appeler une URL arbitraire. Le contrat fournisseur sera généralisé dans la suite du découpage.

Maximum quatre sélections en ligne distinctes par page ; les sélections dupliquées sont refusées. Les appels sont annulés lorsque la page est quittée ou que son fournisseur devient indisponible dans l’interface. Les limites et autorisations serveur restent applicables. Les blocs texte, faits, liens et collections de la version 0.0.41 restent utilisables avant, entre ou après les rangées. Les liens vers une rubrique désactivée sont retirés du document renvoyé.

Un paquet ne peut pas remplacer les pages de sécurité ou les paramètres du noyau. Les textes sont échappés ; aucun HTML, script, chemin de fichier ou SQL fourni par une extension n’est exécuté.

## Mise à jour

Schéma de données 27. La mise à jour réalise la sauvegarde de migration habituelle avant de créer les quatre paquets fournis. Ce provisionnement se produit une seule fois ; il ne réactive pas les paquets aux démarrages suivants. Les médias et métadonnées ne sont pas déplacés ni supprimés.
