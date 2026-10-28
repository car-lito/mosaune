# Gérer les composants médias ensemble — 0.0.40

Dans **Réglages → Extensions → Films**, **Séries** ou **Musique**, ouvrez « Gérer les composants ensemble ».

1. **Préparer les composants** propose d’installer ceux qui manquent et d’activer ceux qui sont arrêtés. Une extension de remplacement déjà choisie reste choisie.
2. **Désactiver les composants propres** arrête les composants propres à cet ensemble.
3. **Retirer les composants propres** désinstalle les programmes choisis propres à l’ensemble. Les alternatives non sélectionnées restent présentes.
4. L’aperçu indique les installations, activations, choix et retraits, ainsi que les composants partagés conservés. Rien ne change avant **Confirmer les changements**.

Si tout est prêt, la préparation indique qu’aucun changement n’est nécessaire. Si une extension externe choisie a été désinstallée, réimportez-la ou choisissez explicitement une autre extension dans sa fiche : Mosaune ne la remplace pas automatiquement.

Un échec annule tout le lot. Si un autre onglet modifie les extensions, il faut refaire l’aperçu. Les médias, métadonnées, affiches et progressions restent conservés. Après une désactivation, une ancienne analyse ne peut pas enregistrer tardivement ses résultats, même si l’ensemble est réactivé immédiatement.

## Ce que cette étape ne fait pas encore

Ces ensembles regroupent les composants déjà extraits ; ils ne retirent pas encore toute la rubrique Films, Séries ou Musique. Les lecteurs, certains écrans et les services encore intégrés restent présents. Les fiches communes ne sont pas retirées automatiquement, même si plus aucun ensemble n’en a besoin. Leur gestion individuelle reste accessible.

[Checklist et estimation d’avancement](AVANCEMENT-EXTENSIONS.md).

## Pour les contributeurs

Les recettes sont dans `media-sets.json` : identifiant, nom et liste de rôles. Les dépendances communes sont déduites des rôles utilisés par plusieurs recettes. Aucun programme supplémentaire ni service permanent n’est lancé par un ensemble. Ces recettes officielles sont embarquées ; l’import d’une recette externe n’est pas encore disponible.

Le serveur construit le plan depuis les paquets et choix conservés, puis le recalcule sous verrou et dans une transaction avant application. Les changements utilisent le même installateur que les actions individuelles. La révision et l’empreinte de l’aperçu empêchent une confirmation d’appliquer un autre plan. Les appels en cours ne sont annulés qu’après réussite de la transaction. Les endpoints `/api/v1/media-sets` sont réservés aux administrateurs.

## Correction de la validation précédente

La CI 0.0.39 a réussi sous Linux et échoué sous Windows : traitement d’une fiche de 1 000 pistes trop long, puis limite de douze minutes de la suite atteinte. Aucune image 0.0.39 n’a été publiée par ce workflow.

Le décodage des faits évite désormais une réécriture JSON complète dans le module. La seule capacité de comportements des fiches dispose d’un plafond de dix secondes par appel ; ses limites de mémoire et de taille restent inchangées. La suite serveur dispose de vingt minutes, indépendamment des limites d’exécution des extensions. La publication 0.0.40 dépend de la nouvelle CI ; ne pas confondre des tests locaux réussis avec une image déjà publiée.

## Vérifications locales

158 tests serveur et 29 tests interface réussis, tests et analyse Go du kit réussis, Svelte sans erreur ni avertissement. Les dix essais HTTP initialement bloqués par les permissions de l’environnement ont été relancés avec succès après renouvellement. Trois essais restent conditionnels (liens symboliques Windows, médias personnels facultatifs, sources publiques facultatives). Windows amd64, Linux amd64 et Linux ARM64 construits. Parcours de désactivation/réactivation et aperçu de retrait validés sur une collection fictive, sans toucher au serveur personnel. La CI GitHub vérifiera de nouveau les sources publiées.
