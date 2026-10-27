# Fiches de séries par extension — 0.0.36

Cette étape confie les saisons et la liste des épisodes à une véritable extension. Les fiches gardent leurs composants et leur thème actuels. L’extension choisit les saisons visibles, les libellés et l’ordre des épisodes.

## Essai simple

1. Ouvrez une série : les saisons et les épisodes sont présentés comme auparavant. Une saison référencée mais absente du serveur reste visible lorsqu’elle annonce des épisodes.
2. Dans Réglages → Extensions → Fiches des séries, désactivez le module.
3. Rouvrez la série : les fichiers restent accessibles dans une liste simple, accompagnée d’une explication. Leur progression et leurs informations restent conservées.
4. Réactivez le module : les saisons reviennent sans nouveau scan.

## Remplacer la présentation

L’archive build/example-detaillist.mosaune.zip est produite avec les fichiers de test GitHub Actions. Importez-la depuis Extensions → À télécharger → Importer un module d’analyse, activez « Épisodes sans saisons — exemple », puis choisissez « Utiliser pour les fiches ». Les épisodes s’affichent en liste simple avec le titre « Épisodes — vue simple ». Activer ne suffit pas à remplacer votre choix actuel.

Pour retrouver les saisons, sélectionnez de nouveau Fiches des séries. Le remplacement ne déplace aucun fichier et ne modifie pas les métadonnées.

[Documentation du contrat](extension-kit/capabilities/detaillist/v1/README.md) · [Exemple contributeur](extension-kit/examples/detaillist)

## Limites de cette tranche

Le haut des fiches, les acteurs, les suggestions, les fiches Films/Musique et les autres comportements restent à extraire. Ce module ne représente donc pas encore toute l’extension Séries. Une fiche trop volumineuse pour le contrat conserve une liste simple, sans cacher d’épisodes ; la pagination viendra ensuite. Les lecteurs personnalisés et le déploiement NAS restent les étapes suivantes.

Validation finale : [tests Windows/Linux et conteneurs réussis](https://github.com/car-lito/mosaune/actions/runs/40399371864). Image ghcr.io/car-lito/mosaune:0.0.36-preview publique et vérifiée. Le parcours de désactivation et réactivation des saisons a été testé dans le navigateur sur une installation isolée.
