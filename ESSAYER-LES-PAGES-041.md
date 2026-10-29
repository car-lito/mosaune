# Pages et menus d’extensions — 0.0.41

## Essayer sans programmer

1. Ouvrez **Réglages → Extensions → À télécharger**.
2. Dans le catalogue, ouvrez **Salon personnel**, inclus avec Mosaune. Vérifiez le paquet puis installez-le.
3. Activez-le dans sa fiche : **Mon salon** apparaît dans le menu principal et dans les paramètres.
4. Ouvrez Mon salon, puis **Le carnet du salon**. Les pages et les raccourcis proviennent du paquet. La collection utilise les albums auxquels votre compte a accès.
5. Changez de thème : ces pages utilisent les mêmes composants et variables que l’application.
6. Désactivez le paquet : ses menus disparaissent. Une page déjà ouverte devient indisponible à l’actualisation des extensions (immédiate dans cet onglet, au retour ou sous 30 secondes dans un autre). Les albums et les comptes restent conservés.
7. Facultatif : autorisez un deuxième compte depuis **Accès des utilisateurs**. Sans cet accès, il ne peut ni voir les menus ni charger les pages directement.

L’exemple est également téléchargeable depuis la section d’import local. Aucun compte de test ni média personnel n’est inclus dans le paquet.

## Construire ses propres pages

Copiez [le manifeste de l’exemple](examples/reading-room/extension.json), modifiez-le puis placez **uniquement `extension.json` à la racine d’un ZIP**. Aucun outil de compilation n’est nécessaire. Changez l’identifiant et le nom pour votre propre extension ; augmentez la version pour mettre à jour un paquet existant.

Contrat : `schema: 1`, `family: "interface"`, `api: 3`, `requires: ["ui.pages.v1"]`, `permissions: []`. L’API 3 désigne un document de pages ; les thèmes gardent leurs API 1 et 2. Le ZIP est limité à 256 Kio, avec au maximum 16 pages et 32 blocs par page.

Chaque page déclare :

- `id` : identifiant local stable, lettres minuscules, chiffres et tirets, débutant par une lettre, 48 caractères maximum.
- `label`, `description` : titre et texte d’introduction.
- `placement` : `main` (menu principal), `personal` (listes personnelles), `settings` (paramètres), `hidden` (accessible depuis un raccourci).
- `icon` : une icône du vocabulaire commun (`home`, `films`, `series`, `music`, `bookmark`, `heart`, `queue`, `settings`, `server`, `search`).
- `admin: true` : page réservée aux administrateurs. Sans ce champ, les droits d’accès au paquet restent nécessaires.
- `kinds` : facultatif ; au moins l’un des types demandés doit être actif sur le serveur et autorisé pour le compte.
- `blocks` : composition de la page, dans l’ordre souhaité.

Blocs disponibles :

| Type | Champs | Comportement |
|---|---|---|
| `text` | `title`, `text` | Texte simple ; aucun HTML exécuté. |
| `facts` | `title`, `facts: [{label, value}]` | Informations présentées en liste descriptive. |
| `link` | `title`, `text`, `target` | Raccourci vers `page:identifiant`, une rubrique connue (`films`, `music`, `favorites`…) ou `settings:account`, etc. |
| `collection` | `title`, `view` | Collection locale (`films`, `series`, `music`, `home`) ou liste enregistrée (`watchlist`, `favorites`), avec les cartes et commandes existantes. |

Les liens vers des pages interdites et les collections d’un type non autorisé sont retirés côté serveur. Les API habituelles restent responsables des droits sur les fichiers. Le paquet ne reçoit ni chemin de fichier, ni clé API, ni accès direct au stockage. Il ne peut pas remplacer la page de connexion ou masquer les réglages de récupération. Les identifiants des pages sont séparés par propriétaire pour éviter les collisions entre extensions.

Dans un dépôt GitHub, annoncez ce format avec `kind: "pages"`. Le téléchargement, l’empreinte, l’aperçu et l’installation utilisent les mêmes parcours que les thèmes. Une mise à jour conserve les autorisations mais reste désactivée jusqu’à sa validation explicite. La désinstallation retire le document et ses autorisations, sans toucher aux collections communes.

## Ce que cette étape termine, et ce qui reste

La navigation principale et celle des paramètres utilisent désormais un registre de descriptions côté serveur. Les pages externes fonctionnent réellement après installation, sans ajouter une branche dans le routeur central et sans nouveau processus au repos.

Les écrans médias historiques utilisent encore leurs composants intégrés : leur déclaration dans la navigation ne les transforme pas en familles installables. Ce premier contrat de pages compose quatre types de blocs ; il ne couvre pas encore les formulaires personnalisés persistants, les traductions, les mises en page totalement libres ou les nouveaux services exécutables. Ces capacités devront rejoindre le contrat commun, avec leur propre validation, et non être ajoutées comme exceptions par extension.

Les thèmes restent responsables des couleurs, polices et arrondis. Une extension de pages choisit la composition, pas du CSS arbitraire. Aucun déploiement sur le NAS ni nouveau lecteur dans cette étape.

## Correction de la validation Docker précédente

Les tests Windows et Linux de 0.0.40 avaient réussi. Le scénario Docker échouait car il préparait l’ensemble Musique (ce qui réactivait volontairement ses composants), puis attendait que les tags audio soient toujours désactivés après redémarrage. Le scénario vérifie maintenant cette réactivation, désactive explicitement les tags, puis contrôle leur état et la conservation des pochettes après redémarrage.
