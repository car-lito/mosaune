# Rubriques des fiches par extension — 0.0.37

Le résumé, les informations complémentaires, les portraits et les bandes-annonces des fiches Films/Séries sont désormais composés par l’extension **Informations des fiches**. Le thème continue de déterminer leur apparence.

## Essai simple

1. Ouvrez la fiche d’un film ou d’une série déjà identifié : retrouvez son résumé, ses informations, ses acteurs et ses bandes-annonces lorsqu’ils sont disponibles.
2. Dans Réglages → Extensions → Informations des fiches, désactivez le module.
3. Rouvrez la fiche : ces rubriques disparaissent avec un message explicatif. Le titre, les commandes et les fichiers restent accessibles. Les saisons sont gérées séparément par Fiches des séries.
4. Réactivez le module et rouvrez la fiche : les rubriques reviennent sans nouvelle identification ni scan.

## Autre présentation

Importez l’archive build/example-detailsections.mosaune.zip depuis les artefacts GitHub Actions, activez **Fiche épurée — exemple**, puis choisissez **Utiliser pour les fiches**. Le résumé apparaît seul dans une rubrique « L’essentiel » en pleine largeur. Pour retrouver la présentation habituelle, sélectionnez **Informations des fiches**. Activer un module ne remplace pas automatiquement le choix actuel.

[Contrat pour les contributeurs](extension-kit/capabilities/detailsections/v1/README.md) · [Exemple](extension-kit/examples/detailsections)

Cette étape ne termine pas l’extensionisation : en-têtes, commandes, suggestions, listes musicales et autres comportements restent à extraire. Les lecteurs personnalisés puis le déploiement sur le NAS viendront après.

Validation finale : [tests Windows/Linux et conteneurs réussis](https://github.com/car-lito/mosaune/actions/runs/40507496331). Image ghcr.io/car-lito/mosaune:0.0.37-preview publique et vérifiée. La désactivation et le remplacement ont été testés dans le navigateur sur une installation isolée.
