# Roadmap active

Mise à jour : 26 novembre 2026, base 0.0.85. Ce document remplace les anciennes listes dispersées. **Ordre : E → P → N**, puis élargissement progressif. Pas de dates promises sans estimation du lot. Les objectifs peuvent évoluer ; toute décision contraire à [DECISIONS](product/DECISIONS.md) doit être explicite.

Les capacités déjà livrées sont dans [STATUS](STATUS.md). Une case ne se coche qu’avec code, documentation et preuve adaptée ; un changement de dossiers ou un pourcentage ne suffit pas.

## E — Terminer les extensions regroupées (priorité actuelle)

- [ ] **E1 — Configuration générique.** Remplacer les formulaires/guides spécifiques encore imposés par le serveur ou l’UI (notamment `MetadataSettings.svelte` et les adaptateurs de configuration). Vérifier secrets, permissions, migrations et erreurs avec un autre fournisseur. Fini quand il peut déclarer ses réglages sans modifier l’écran central.
- [ ] **E2 — Services médias restants.** Auditer profils, recherche/découverte, associations, playlists, commandes et orchestration. Déplacer les règles métier et garder seulement les primitives communes dans le noyau. Les contrats profils/association/images existent déjà ; éviter de recommencer ces extractions. Fini avec un remplacement réel, absence de repli caché et tests de retrait pendant une opération.
- [ ] **E3 — Distribution indépendante.** Achever les adaptateurs fournisseurs et connecteurs comme capacités distribuables, ou formaliser précisément les primitives hôtes nécessaires. Étendre confiance des éditeurs, autorisations, quotas/budgets globaux, stockage privé et migrations sans permettre le chargement natif arbitraire. Fini avec un paquet externe crédible utilisant son format/sa source sans modification du noyau.
- [ ] **E4 — Familles complètes.** Contraintes de versions inter-familles, changements de composition, dépendances partagées, éléments remplaçables/cumulables, installation partielle et récupération après échec. Retrait = nettoyage privé sûr, pas suppression des données partagées d’une autre famille.
- [ ] **E5 — Contributions UI complètes.** Relier les contrats de pages, menus, recherche et formulaires aux familles complètes ; généraliser langues/traductions et compositions nécessaires. Documenter ce qui peut être étendu et fournir des exemples sans JavaScript communautaire arbitraire.
- [ ] **E6 — Recette de sortie.** Depuis un catalogue vide : noyau seul ; Musique seule ; Films seuls ; Séries seules ; combinaisons ; ajout, remplacement, mise à jour, désactivation, retrait, restauration et redémarrage. Vérifier absence d’appels après désactivation, isolation des comptes, données conservées, conflits et pannes. Répéter Windows/Linux amd64/ARM64 et publier un exemple complet de famille extérieure au noyau avec composant remplaçant et contribution cumulable.

**Passage aux lecteurs :** E1–E6 démontrés ou écarts explicitement acceptés et documentés par le propriétaire. Ne pas déclarer « 100 % extensions » sur la seule présence d’interfaces Go.

## P — Lecteurs vidéo et musique personnalisés

- [ ] **P1 — Contrats et choix des moteurs.** Session commune (file, état, progression, commandes), capacités de plateforme, sélection explicite d’un lecteur alternatif. Choisir des bases open source après vérification des licences/maintenance. Un lecteur vidéo partagé Films/Séries ; lecteur audio fourni par Musique.
- [ ] **P2 — Vidéo.** Bandeau inférieur avec aperçu pendant navigation ; vue agrandie et plein écran ; lecture/pause, recherche temporelle, sauts, volume, pistes audio, sous-titres, erreurs utiles, clavier/télécommande. Thèmes appliqués partout.
- [ ] **P3 — Musique.** Bandeau permanent, pochettes, file/aléatoire/répétition, volume, progression ; clic pochette vers vue immersive, crédits et emplacements extensibles. Reprise sans interruption ni duplication de session.
- [ ] **P4 — Compatibilité mesurée.** Examiner fichiers/pistes ; lecture directe d’abord, remux/transcodage seulement si nécessaire et soutenable. Tester codecs, conteneurs, sous-titres forcés, synchronisation et changements de piste. Mesurer ressources sur petite machine. Ne pas confondre contrôles personnalisés et capacité de décodage.
- [ ] **P5 — Qualité progressive.** Lecture enchaînée, chapitres, détection/saut d’intro et générique, options audio avancées (gapless, normalisation, fondus selon faisabilité). Définir un socle P2/P3 acceptable avant d’attendre la totalité des raffinements.

Spécification détaillée : [PLAYERS](product/PLAYERS.md). Android peut partager les contrats sans partager le moteur du navigateur.

## N — Première installation NAS neuve et bêta

- [ ] **N1 — Parcours d’installation.** Import GitHub/CasaOS, volumes médias en lecture seule, données persistantes, HTTPS/certificats et premier admin ; guide novice complet. CasaOS 0.4.15 fait partie des cibles historiques à vérifier, pas une garantie universelle.
- [ ] **N2 — Recette matérielle.** Pi 5 ARM64, bibliothèque de plus de 1 000 films, reprise du scan après coupure, NAS indisponible, consommation au repos/en scan/en lecture. Définir puis publier budgets et limites observés.
- [ ] **N3 — Exploitation.** Sauvegarde/restauration, mise à jour d’image, retour arrière, compatibilité base/paquets, permissions des volumes, redémarrage et arrêt propre. Une rétrogradation de binaire ne doit pas être présentée comme un rollback sûr de base sans preuve.
- [ ] **N4 — Bêta communautaire.** Choisir licence, politique de contribution et de sécurité, canaux de retours, versions/releases, tests de démarrage vierge. Revue de confidentialité de chaque publication. Aucun déploiement personnel automatique.

## D — Découverte, suivi et demandes

- [ ] **D1 — Goûts et recommandations.** Favoris/historique comme signaux, suggestions inter-contenus, explication/source, préférences de langue/région et confidentialité. Ne pas présenter les seules sélections TMDB comme un moteur personnel complet.
- [ ] **D2 — Suivre des créateurs.** Chanteurs, acteurs, auteurs : nouveaux projets, sorties prévues, règles de demande automatique configurables, dédoublonnage et quotas.
- [ ] **D3 — Notifications.** Disponibilité réelle après import, erreurs de demandes et nouveaux projets ; préférences par compte, canaux extensibles, pas de bruit permanent.
- [ ] **D4 — Acquisition au-delà de la vidéo.** Gestionnaires compatibles musique/livres/autres, demande de pistes ou albums absents, contrats déclaratifs et permissions. Ne pas ajouter un téléchargeur torrent au noyau.

## C — Clients et mobilité

- [ ] **C1 — Android mobile.** Connexion, navigation et lecture ; téléchargements depuis le serveur, gestion du stockage hors ligne, expiration/droits, synchronisation des progressions et erreurs réseau.
- [ ] **C2 — Android TV.** Navigation au focus/télécommande, connexion et profils Mosaune ; priorité aux essais Chromecast APK ; validation ensuite sur autres appareils.
- [ ] **C3 — Intégrations système TV.** Reprise depuis l’accueil Android TV/Google TV ; examiner les API de profils et l’éventuel lien aux comptes Google, avec consentement et repli par sélection manuelle. Pas d’accord Google supposé ni de synchronisation implicite entre frères/comptes.
- [ ] **C4 — iOS optionnel.** Décider effort, distribution, coûts et moteur avant engagement. Garder les contrats clients portables.

## M — Familles supplémentaires

- [ ] **M1 — Livres audio.** Chapitres, vitesse, signets, minuterie, reprise fiable, séries/auteurs, hors-ligne ; socle de qualité inspiré d’Audiobookshelf/Audible.
- [ ] **M2 — Podcasts.** Abonnements/flux, nouveaux épisodes, téléchargement, reprise et rétention configurable.
- [ ] **M3 — Ebooks.** Formats retenus, lecteur adapté, progression, couvertures/auteurs, accessibilité et synchronisation.
- [ ] **M4 — YouTube.** Recherche/abonnements/lecture et intégration TV ; étudier base existante, maintenance et possibilité d’une expérience sans publicité. Compatibilité et conditions à vérifier avant choix ; pas de promesse universelle.

## Travaux transversaux permanents

- [ ] Internationalisation effective, traductions comme contributions, cohérence langue des métadonnées/bandes-annonces.
- [ ] Accessibilité clavier/tactile/TV, contraste des thèmes, tailles d’écran et absence de dépendance au survol.
- [ ] Mesures de performance, caches bornés, indexation incrémentale et absence de processus inutiles au repos.
- [ ] Sécurité : isolation des extensions, réseau/SSRF, secrets, sessions, permissions sur toute route, fichiers malformés et maintenance des dépendances.
- [ ] Documentation des contrats, compatibilité et dépréciations ; conserver un exemple contributeur exécutable.

## Entretien du dépôt

- [x] Dossiers serveur/ressources/documentation et guides historiques séparés (0.0.85).
- [x] Dossier de reprise public, décisions consolidées et roadmap globale (0.0.85).
- [x] Passe de confidentialité des fichiers suivis, archives de paquets et objets historiques accessibles (voir [périmètre](security/PRIVACY.md)).
- [ ] Choix final de licence et politique de confiance communautaire : décisions encore ouvertes, ne pas les deviner.
