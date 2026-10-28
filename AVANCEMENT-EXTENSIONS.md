# Vers un Mosaune entièrement extensible

État du 28 octobre 2026 · étape 0.0.40.

**Environ 60 % du découpage visé est réalisé.** Il s’agit d’une estimation d’architecture pour Films, Séries et Musique, pas du pourcentage de toutes les fonctionnalités de Mosaune ni du temps restant. Les lecteurs personnalisés, Android/TV et le déploiement NAS viennent ensuite.

La référence est exigeante : une fonction doit pouvoir être installée, désactivée, retirée et remplacée sans modifier le noyau, avec des droits contrôlés et des données communes conservées. Déplacer du code dans un fichier ne suffit pas. Les pourcentages ci-dessous sont des évaluations manuelles, arrondies ; les cases cochées désignent des parcours effectivement implémentés et testés.

| Chantier | Poids | Estimation réalisée | Contribution au total |
|---|---:|---:|---:|
| Moteur, contrats, registre et isolation | 20 % | 90 % | 18 points |
| Installation, dépendances et gestion des ensembles | 20 % | 60 % | 12 points |
| Extraction des comportements Films/Séries/Musique | 25 % | 72 % | 18 points |
| Thèmes et contributions d’interface | 15 % | 60 % | 9 points |
| Fournisseurs et connecteurs indépendants | 10 % | 30 % | 3 points |
| Validation finale d’un noyau vide et d’une installation complète par extensions | 10 % | 0 % | 0 point |
| **Total estimé** | **100 %** | | **60 points** |

## Checklist

### Socle des extensions

- [x] Exécution de modules WASI avec limites, arrêt des appels et contrôle des résultats.
- [x] Registre persistant, révisions et choix explicites de modules.
- [x] Contrats de réglages et d’autorisations par propriétaire ; première intégration TMDB.
- [x] Dépôt officiel, ajout de dépôts GitHub et import de paquets compatibles.
- [x] Installation, activation, sélection, mise à jour, restauration et retrait des familles exécutables déjà prises en charge.
- [ ] Généraliser réglages, migrations privées, permissions de services et gestion des secrets à toutes les familles.
- [ ] Terminer les règles de confiance des éditeurs et les budgets globaux pour ouvrir plus largement les extensions exécutables.

### Comportements médias

- [x] Reconnaissance des fichiers Films, Séries et Musique.
- [x] Lecture des informations et pochettes intégrées aux fichiers musicaux.
- [x] Regroupement et classement des catalogues.
- [x] Saisons et listes d’épisodes.
- [x] Résumés, informations, distribution et bandes-annonces des fiches.
- [x] Titres et informations de l’en-tête des fiches.
- [x] Commandes, artistes, listes de pistes/fichiers, rapprochement des pistes, format des pochettes et présentation des suggestions.
- [x] Exemples de remplaçants externes réellement installés et exécutés.
- [ ] Extraire les comportements encore intégrés : profils, services de recherche/découverte, association des métadonnées, playlists et adaptateurs des commandes.
- [ ] Livrer trois familles médias complètes, dont l’installation et le retrait commandent aussi leurs écrans et services.

### Gestion commune et dépendances

- [x] Résolution de dépendances par capacité dans le moteur et le registre.
- [x] **0.0.40 : préparer, désactiver ou retirer en lot les composants déjà extraits de Films, Séries et Musique.**
- [x] Aperçu détaillé, application transactionnelle, refus d’un aperçu périmé et conservation des remplaçants choisis.
- [x] Les composants de fiches partagés sont conservés lors du retrait d’un ensemble.
- [ ] Généraliser la distribution des dépendances et leurs contraintes de versions aux paquets médias complets.
- [ ] Ajouter les dépendances de lecture partagées lorsque les lecteurs seront construits.

Les ensembles 0.0.40 sont des recettes de gestion des composants existants. Ils ne sont pas encore les extensions médias complètes. Leur retrait ne masque pas toute une rubrique et ne retire ni fournisseur ni lecteur intégré. Les composants communs sont conservés même si les trois ensembles sont retirés ; leur retrait individuel reste possible.

### Interface et fournisseurs

- [x] Contrat de thème avec variables, variantes et composants communs.
- [x] Cartes d’extensions, fiches de réglages, recherche et présentation extensible des résultats.
- [x] Rendu des fiches à partir de contributions contrôlées, avec les thèmes.
- [ ] Navigation, pages, menus, réglages et traductions enregistrés par les extensions sans modifier les routes centrales.
- [x] Paquet TMDB pilote et désactivation réelle de ses accès en ligne.
- [ ] Achever l’indépendance de TMDB, supprimer sa substitution intégrée et généraliser l’interface fournisseur.
- [ ] Extraire MusicBrainz et ses services associés.
- [ ] Extraire Radarr et Sonarr derrière le même contrat de demandes/suivi.

### Condition de fin du découpage

- [ ] Démarrer avec seulement le noyau : comptes, permissions, stockage commun et gestionnaire d’extensions.
- [ ] Ajouter uniquement Musique puis uniquement Films ou Séries, avec leurs dépendances.
- [ ] Retirer/remplacer chaque famille sans casser les autres ni perdre les données communes.
- [ ] Reproduire ces parcours sur Windows, Linux amd64 et ARM64.
- [ ] Fournir un exemple complet de famille média extérieure au noyau et un guide pour contributeurs.

**Ordre après le découpage :** lecteurs personnalisés vidéo et musique, puis installation neuve sur CasaOS. Aucun déploiement sur le NAS personnel à cette étape.
