Pour ajouter des pages et des menus sans compiler de programme : [contrat ui.pages.v1 et exemple Salon personnel](ESSAYER-LES-PAGES-041.md).

# Créer un thème pour Mosaune — contrat v2

Client web Mosaune 0.0.15. Un thème est un fichier JSON déclaratif : il fournit des valeurs, les composants Mosaune les utilisent. Le thème natif utilise le même mécanisme. JSON permet une validation stricte avec les outils déjà présents ; aucun moteur YAML ni CSS libre n’est nécessaire.

## Démarrer

Dans Réglages → Extensions, téléchargez le modèle JSON complet et le catalogue des réglages. Ils existent aussi dans [examples/ambre-v2/extension.json](examples/ambre-v2/extension.json) et [theme/contract.json](theme/contract.json).

1. Copiez le modèle dans un dossier de travail ; modifiez id, name, publisher et description pour créer votre propre thème. Gardez api 2, schema 1, family interface, requires ["ui.theme.v2"], permissions [] et minCore 0.0.15.
2. Personnalisez les valeurs décrites ci-dessous. Il n’est pas nécessaire de modifier le serveur ou les composants.
3. Compressez uniquement **extension.json**, à la racine du ZIP, puis importez le ZIP dans Extensions.
4. Installez, activez et utilisez le thème pour votre compte. Pour une mise à jour, conservez id et publisher, augmentez version (par exemple 1.0.0 → 1.0.1), puis réactivez après installation.

L’identifiant doit contenir au moins deux segments en minuscules ASCII, séparés par un point (ex. atelier.cinema), et ne pas utiliser le préfixe réservé mosaune. Le nom et l’éditeur déclaré sont limités à 80 octets, l’identifiant à 80 et la description à 600. Les versions sont trois entiers sans zéro initial. Le ZIP et son contenu décompressé sont chacun limités à 256 Kio. Une seule entrée régulière extension.json est acceptée ; aucun fichier n’est extrait. Les champs, familles et capacités inconnus sont refusés. Maximum 32 paquets installés.

## Les cinq parties du thème

| Partie | Rôle |
|---|---|
| palette | Couleurs nommées, avec light et dark pour chacune |
| fonts | Familles de police réutilisables, par ordre de préférence |
| radii | Arrondis réutilisables pour affiches, boutons, badges, champs… |
| tokens | Valeur de chaque réglage de composant, éventuellement référence à une valeur ci-dessus |
| sections | Exceptions facultatives par rubrique ; toutes les autres valeurs restent communes |

Le modèle complet contient 996 réglages. Un paquet peut n’en fournir qu’une partie : Mosaune complète les valeurs absentes avec son thème natif. Pour maîtriser toute l’apparence, partez du modèle complet.

Extrait illustratif de la partie theme (à intégrer au manifeste complet) :

```json
{
  "palette": {
    "menu": { "light": "#352619", "dark": "#21170f" },
    "menu-musique": { "light": "#302343", "dark": "#171020" }
  },
  "fonts": {
    "body": ["Segoe UI", "Arial", "sans-serif"],
    "title": ["Georgia", "serif"]
  },
  "radii": { "poster": "14px", "primary": "10px", "badge": "6px" },
  "tokens": {
    "sidebar.background": "$palette.menu",
    "app.font.family": "$fonts.body",
    "button.primary.border.radius": "$radii.primary",
    "button.shared.font.size": "16px"
  },
  "sections": {
    "music": { "sidebar.background": "$palette.menu-musique" }
  }
}
```

Ici le menu reste brun dans Films, Séries et Réglages, et devient violet dans Musique. Les autres propriétés du menu restent inchangées. Sans sections, toute l’application partage les mêmes réglages. Rubriques acceptées : home, films, series, music, settings, watchlist, favorites, playlists, requests. La couleur claire/sombre est résolue automatiquement à partir de la palette ; le mode reste une préférence du compte.

## Retrouver un élément

Le catalogue associe chaque nom de réglage à son type et à ses utilisations : composant, sélecteur et propriété. Cherchez par exemple sidebar pour le menu, button.primary pour les actions principales, tile.status pour les indicateurs de disponibilité. Les états survol/sélection et les adaptations selon la taille d’écran ont leurs propres entrées. Les noms sont exacts ; les inconnus sont rejetés pour signaler les fautes de frappe.

On peut utiliser une couleur directe #RRGGBB ou #RRGGBBAA, ou $palette.nom. Une police utilise $fonts.nom et un arrondi peut utiliser $radii.nom. Les références ne pointent pas vers d’autres tokens : pas de chaîne ou de cycle implicite.

Les polices sont des familles déjà disponibles sur l’appareil, avec des replis ; le paquet ne télécharge pas de police. Les tailles se donnent en px ou rem, ou en clamp(min, valeur, max) avec vw possible dans clamp. Les tailles restent identiques entre modes clair/sombre ; leurs couleurs changent via la palette. Les variantes de taille par rubrique sont possibles via sections.

## Validation et limites

Les valeurs sont validées par type. Arrondis : 0–100px,0–6rem ou0–50%. Tailles : 4–128px ou0,25–8rem ; vw 0–10 dans clamp. Graisses 100–900, interligne 0,8–3. Les deux réglages du mot décoratif spotlight.word disposent de bornes adaptées ; ils ne permettent pas de rendre les commandes invisibles. Les contrastes principaux texte/fond doivent atteindre 4,5 en clair et sombre, y compris dans les exceptions. Cela ne remplace pas une vérification visuelle et d’accessibilité complète.

Les thèmes alimentent les couleurs, typographies, arrondis et opacités des composants. Ils ne changent pas leur ordre ou leur structure, ne chargent pas de ressource distante, n’injectent ni JavaScript ni CSS et n’accèdent pas aux données des médias. Les contrôles natifs internes du lecteur navigateur restent sous son contrôle ; Mosaune peut fournir la préférence claire/sombre, pas redessiner chaque bouton interne. Les futures dispositions, contributions de menus, traductions et applications natives demanderont des contrats dédiés.

## Compatibilité et cycle de vie

Les anciens paquets api 1/ui.theme.v1 restent acceptés ; leurs neuf couleurs sont adaptées aux nouveaux réglages. Les nouvelles personnalisations utilisent api 2/ui.theme.v2. Le numéro schema de l’enveloppe reste 1.

Installer ou mettre à jour conserve le paquet désactivé jusqu’à activation explicite. La mise à jour garde droits d’usage et préférences ; désactiver les conserve aussi. Désinstaller retire ces références, sans modifier la bibliothèque. Les versions identiques ou inférieures sont refusées. L’empreinte lie l’installation au paquet prévisualisé ; elle ne certifie pas l’identité de l’auteur. L’éditeur reste déclaratif.

Les paramètres suivent le thème du compte. La connexion reste native avant identification ; le mode de secours permet toujours de gérer les paquets sans les appliquer. Les autres onglets visibles actualisent le choix sous 30 secondes ou au retour de fenêtre. Testez clair/sombre, chaque rubrique, les petits écrans, la désactivation et un second compte autorisé.

## Pour les développeurs de Mosaune

theme/default.json contient les valeurs natives, theme/contract.json la liste typée. Ajouter un élément visuel implique de déclarer son réglage et de le consommer dans le composant, avec une valeur par défaut. scripts/build-theme.mjs génère la feuille native au démarrage du build Vite ; theme.go produit les variables des paquets avec le même contrat. Le test Go compare les deux rendus. web/src/theme.test.mjs contrôle les styles CSS/Svelte pour éviter de réintroduire des couleurs, polices ou arrondis codés directement dans les composants.
