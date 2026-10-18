# Présentation d’un résultat — v1

Ce contrat décrit l’aperçu et la ligne sous le titre, indépendamment du type de média. Le renderer web commun est `SearchResultContent.svelte`. Les noms de types et les mots-clés ne sont pas une liste fermée.

```json
{
  "version": 1,
  "label": "Jeu",
  "keywords": ["Console", "Nintendo Switch"],
  "preview": {
    "aspectRatio": 0.62,
    "shape": "rectangle",
    "fit": "contain"
  }
}
```

L’objet est facultatif, dans le champ `presentation` d’une fiche `Metadata`, d’un candidat `metadata.search.v1` ou d’une fiche `metadata.video-details.v1`. Le serveur 0.0.25 accepte et valide ces champs puis les conserve dans le catalogue partagé. Déclarer un noyau minimum 0.0.25 pour un nouveau paquet qui les émet : les anciens hôtes stricts peuvent les refuser. Les paquets existants sans ce champ restent compatibles.

Une future capacité de recherche d’artistes, de consoles ou de jeux pourra utiliser le même objet. **Ce contrat de présentation n’installe pas à lui seul un nouveau module média ni une nouvelle source de recherche.** Les capacités et les actions d’ouverture doivent aussi être prises en charge par l’hôte. Le pilote actuel reste limité aux modules Films, Séries et Musique.

| Champ | Règle |
|---|---|
| `version` | `1` |
| `label` | Type affiché, texte libre non vide, 40 caractères maximum |
| `keywords` | Facultatif, au plus trois textes non vides de 40 caractères chacun |
| `preview` | Facultatif ; conserve les proportions natives en son absence |
| `aspectRatio` | Largeur / hauteur entre 0,5 et 3 ; carré : 1, affiche : 2/3, paysage : 16/9 |
| `shape` | `rectangle` ou `circle` ; un cercle exige un ratio de 1 |
| `fit` | `cover` remplit en recadrant ; `contain` montre l’image entière |

Exemples : `Album`, `Single`, `EP`, `Saison`, `Épisode`, `Artiste`, `Acteur`, `Jeu`, `Console`. Les textes doivent être fournis dans la langue demandée par le client ; ce contrat n’ajoute pas encore un moteur de traduction. L’app native reste en français.

Les textes sont échappés, jamais interprétés comme du HTML. Aucun CSS, script ou URL ne peut être introduit par ce champ. L’image reste une ressource autorisée par l’hôte, séparée de sa présentation. Les couleurs, les polices et les arrondis rectangulaires proviennent du thème. Les proportions sont bornées pour préserver l’alignement de la liste ; une image absente ou en erreur garde un aperçu de même forme.

Le serveur refuse les présentations invalides avant persistance. Le client applique un repli pour les données anciennes ou inconnues. La disponibilité « Chez vous / À découvrir », les autorisations et l’action d’ouverture sont contrôlées par Mosaune, pas par les mots-clés de l’extension.

Pour la musique native, le libellé Album/Single/EP utilise le `primary-type` MusicBrainz, quand fourni. Une réponse sans type est présentée comme « Sortie musicale » ; un morceau local n’est jamais déclaré single simplement parce qu’il est seul. Référence : [API MusicBrainz](https://musicbrainz.org/doc/MusicBrainz_API).
