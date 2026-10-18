# Essayer la recherche — 0.0.25

Lancez la nouvelle version comme d’habitude, puis rechargez la page de Mosaune. Votre bibliothèque et vos réglages sont conservés.

1. Depuis l’accueil, recherchez un mot présent dans un film et dans votre musique. Les pochettes musicales sont carrées ; les affiches de films et de séries restent verticales.
2. Regardez sous chaque titre : le type précède les autres informations, par exemple « Film · 2024 », « Album · Artiste » ou « Morceau · Artiste ».
3. Avec MusicBrainz disponible, recherchez une sortie musicale connue. Lorsque le fournisseur indique son type, Mosaune affiche Album, Single ou EP. S’il ne le précise pas, le résultat en ligne porte la mention « Sortie musicale » : un album d’un seul morceau n’est pas automatiquement un single.
4. Ouvrez un résultat pour vérifier que la navigation habituelle fonctionne. « Chez vous » et « À découvrir » continuent d’indiquer sa disponibilité, indépendamment de son type.

Pour une ancienne fiche, une nouvelle recherche en ligne peut compléter le type manquant sans remplacer ses informations détaillées.

## Pour les créateurs d’extensions

Le [contrat de présentation](extension-kit/presentation/v1/README.md) permet de fournir un libellé libre, jusqu’à trois mots-clés et les proportions de l’aperçu. Il propose une forme rectangulaire ou ronde et un cadrage qui remplit l’aperçu ou conserve l’image entière. Les couleurs, polices et arrondis des rectangles restent définis par le thème.

Par exemple, une extension pourra présenter « Jeu · Console » avec une jaquette verticale, ou « Artiste » avec une image ronde. Ce format est prêt ; il ne crée pas à lui seul les sources de recherche pour les jeux, acteurs ou saisons. Cette version améliore les résultats que Mosaune sait déjà rechercher.
