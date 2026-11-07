# Vers un Mosaune entièrement extensible

État du 7 novembre 2026 · étape 0.0.65.

**Environ 85 % du découpage visé est réalisé.** C’est un ordre de grandeur pour Films, Séries et Musique. Les lecteurs personnalisés, Android/TV et le déploiement NAS viennent ensuite.

La référence est exigeante : une fonction doit pouvoir être installée, désactivée, retirée et remplacée sans modifier le noyau, avec des droits contrôlés et des données communes conservées. Déplacer du code dans un fichier ne suffit pas. Les pourcentages ci-dessous sont des évaluations manuelles, arrondies ; les cases cochées désignent des parcours effectivement implémentés et testés.

Ce pourcentage est indicatif, sans pondération chiffrée par chantier. La validation du noyau seul et des familles indépendantes reste une condition de fin à part entière.

## Regroupement confirmé après la 0.0.45

La cible est désormais **quelques grandes extensions visibles, composées de sous-composants extensibles**. Films, Séries et Musique regroupent leurs comportements et écrans ; leurs composants restent remplaçables ou cumulables selon un contrat explicite. Les fournisseurs, connecteurs et thèmes restent des ajouts distincts lorsqu’ils ont un usage propre. Les dépendances techniques ne multiplient pas les cartes par défaut.

[Architecture et ordre de migration](ARCHITECTURE-EXTENSIONS-GROUPEES.md). Priorité à un vrai cycle de vie par paquet principal, puis à sa présentation. La 0.0.46 implémente le regroupement des composants extraits et des écrans, avec opérations transactionnelles et réglages avancés. La 0.0.47 ajoute les archives locales et leurs mises à jour coordonnées. La 0.0.48 ajoute le catalogue distant, ses exemples et le même aperçu d’installation. La 0.0.49 ajoute le retour coordonné des composants mis à jour, sans rétrograder les partagés ni les remplaçants choisis. La 0.0.50 extrait l’indexation de la recherche locale dans les catalogues existants. La 0.0.51 généralise les déclarations des fonctions de recherche et de sélection en ligne, sans terminer la distribution des adaptateurs. Leur indépendance complète reste ouverte ; l’estimation globale reste approximativement 70 %, la recherche distante et les autres services restant intégrés.

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
- [x] Indexation des résultats de recherche locale (groupes/fichiers, mots-clés, priorités et présentation) dans les mêmes composants.
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
- [x] 0.0.47 : archives de composants et écran, import local et mise à jour avec conservation des remplacements choisis et réutilisation des composants partagés.
- [x] Distribution distante et retour coordonné à la version précédente (0.0.48/49).
- [ ] Contraintes inter-familles et changements de composition.
- [x] 0.0.46 : présenter les composants techniques fournis dans les réglages avancés de leur famille et les filtrer dans le catalogue.
- [x] 0.0.46 : manifestes de composition versionnés, opérations communes avec l’écran et choix des remplacements conservés.
- [ ] Ajouter les dépendances de lecture partagées lorsque les lecteurs seront construits.

Les ensembles 0.0.40 sont des recettes de gestion des composants existants. Ils ne sont pas encore les extensions médias complètes. Depuis la 0.0.46, le regroupement retire aussi l’écran choisi et sa rubrique ; il ne retire pas les fournisseurs ni le lecteur intégré. Les composants communs sont conservés même si les trois ensembles sont retirés ; leur retrait individuel reste possible.

### Interface et fournisseurs

- [x] **0.0.61 :** routes HTTP, paramètres d’API, délais et tailles de réponse Radarr/Sonarr déclarés par le composant partagé. L’hôte exécute des opérations nommées sur le serveur configuré, sans exposer la clé au module. Ancien composant sans transport : mise à jour requise, aucun repli natif. Sérialisation des objets distants et formulaires encore intégrés.

- [x] **0.0.60 :** interprétation du suivi Radarr/Sonarr dans le même composant partagé : progression, saisons, blocages et états de disponibilité. Les fichiers lisibles restent vérifiés par le serveur ; aucun repli natif si le composant ne sait pas suivre. Adaptateurs HTTP et formulaires encore intégrés.

- [x] **0.0.57 :** sources réseau, limites et réglages du catalogue TMDB distribués par son paquet signé ; langue et pays configurables sans changer le serveur.

- [x] **0.0.56 :** transport HTTP commun pour TMDB, MusicBrainz et leurs images, cache borné, destinations contrôlées et attente annulable. Les adaptateurs et configurations restent intégrés ; cette étape ne termine pas leur indépendance.

- [x] **0.0.55 :** recherche des pochettes, replis, portraits et crédits pilotée par le même composant Musique ; étapes et accès réseau bornés, cache partagé conservé.

- [x] **0.0.54 :** préparation des recherches MusicBrainz (albums, association, artistes, discographies, prochaines sorties) dans le même composant Wasm ; le noyau valide puis exécute le plan borné.
- [x] Contrat de thème avec variables, variantes et composants communs.
- [x] Cartes d’extensions, fiches de réglages, recherche et présentation extensible des résultats.
- [x] Rendu des fiches à partir de contributions contrôlées, avec les thèmes.
- [x] Pages déclaratives installables : menus principaux/personnels et rubriques de paramètres, textes, informations, raccourcis et collections ; import et dépôts, droits par compte, activation/retrait.
- [x] Menus existants servis par le même registre de descriptions, sans liste de rubriques codée dans le composant principal.
- [x] Accueil, Films, Séries, Musique, Ma liste et Favoris : composition distribuée en paquets d’interface, retrait des menus, remplacement externe et réinstallation.
- [ ] Relier ces interfaces aux familles médias complètes ; extraire profils et formulaires, généraliser traductions et compositions avancées.
- [x] Paquet TMDB pilote et désactivation réelle de ses accès en ligne.
- [x] TMDB 0.5.0 : découverte, suggestions, sélections, profils et affiches alternatives dans le programme installé ; anciens paquets refusés pour ces fonctions sans substitution silencieuse.
- [x] Supprimer le moteur TMDB natif dupliqué : module obligatoire, pause et désinstallation persistantes sans substitution automatique.
- [x] Distribuer le module TMDB fourni comme un paquet signé installé ; aucun retour implicite après retrait ni activation sans réinstallation.
- [x] Lire les recherches et sélections depuis le manifeste signé TMDB installé, avec cache, retour arrière et retrait cohérents (0.0.52).
- [ ] Achever l’indépendance de TMDB : adaptateur du transport, configuration, identifiants Sonarr et interface fournisseur générique.
- [x] Extraire l’interprétation des réponses MusicBrainz (albums, éditions, pistes et artistes) dans un composant Wasm de Musique, remplaçable et désactivable.
- [ ] Achever MusicBrainz : adaptateur du transport réseau, autorisation de nouvelles sources, configuration et distribution comme fournisseur indépendant.
- [x] 0.0.59 — préparation des ajouts, saisons et recherches Radarr/Sonarr dans un composant partagé installable et remplaçable. Activation et retrait contrôlent les échanges, y compris le suivi.
- [x] 0.0.60–0.0.62 — suivi, routes HTTP et formulaires Radarr/Sonarr pilotés par le composant partagé.
- [x] 0.0.63 — formats des objets API Radarr/Sonarr déclarés par le composant partagé, lecture/écriture et commandes.
- [x] 0.0.64 — programme d’envoi et préparation des écritures fournis par le composant partagé ; exécuteur hôte borné et validation avant tout envoi.
- [x] 0.0.65 — plan de collecte du suivi installé, limites contrôlées, projection privée dans le contrat commun et absence de repli natif.
- [ ] Achever le stockage de configuration, les prévisualisations, le rapprochement local et l’indépendance des connecteurs.

- [x] **0.0.62 :** formulaires déclaratifs, guides, ordre des champs, limites de saisie et rejet des formulaires périmés. Même composant partagé, sans nouvelle extension visible.

### Condition de fin du découpage

- [ ] Démarrer avec seulement le noyau : comptes, permissions, stockage commun et gestionnaire d’extensions.
- [ ] Ajouter uniquement Musique puis uniquement Films ou Séries, avec leurs dépendances.
- [ ] Retirer/remplacer chaque famille sans casser les autres ni perdre les données communes.
- [ ] Reproduire ces parcours sur Windows, Linux amd64 et ARM64.
- [ ] Fournir un exemple complet de famille média extérieure au noyau et un guide pour contributeurs.

**Ordre après le découpage :** lecteurs personnalisés vidéo et musique, puis installation neuve sur CasaOS. Aucun déploiement sur le NAS personnel à cette étape.
