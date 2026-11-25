# Instructions de contribution et de reprise

Ce fichier s’applique à tout le dépôt. Lire dans cet ordre :
1. [README](README.md) : présentation et lancement.
2. [État réel](docs/STATUS.md) : fonctions existantes et limites.
3. [Décisions produit](docs/product/DECISIONS.md) : engagements à préserver.
4. [Architecture et arborescence](docs/architecture/OVERVIEW.md).
5. [Roadmap](docs/ROADMAP.md) : prochain lot et critères de validation.
6. Le contrat du domaine concerné dans [l’index technique](docs/README.md).

Une demande explicite du propriétaire peut changer ces décisions : mettre alors les documents à jour. Les guides de `docs/archive/` ne sont pas des exigences actuelles. L’historique explique les étapes passées ; ses anciennes cases non cochées ne sont pas une seconde roadmap.

## Contraintes à préserver

- Finir les extensions regroupées, puis les lecteurs personnalisés, puis une installation NAS neuve. Publier du code ou une image de test n’autorise pas un déploiement sur un serveur personnel.
- Quelques familles visibles (Films, Séries, Musique), avec composants internes remplaçables ou cumulables. Ne pas créer une carte d’extension par petite fonction.
- Le noyau possède comptes, permissions, stockage commun, transactions, cycle de vie et composition. Les règles médias appartiennent aux extensions. Déplacer du code dans un dossier ne termine pas une extraction.
- Préserver les choix explicites et l’invalidation des appels après désactivation. Ne pas réactiver un fournisseur retiré ni utiliser un moteur caché de remplacement.
- Respecter les thèmes déclaratifs et leurs tokens, y compris pour les futurs lecteurs. Pas de palette figée propre à un nouvel écran. Les contributions doivent suivre des contrats versionnés.
- Serveur Go, web Svelte/TypeScript, SQLite, modules WASI bornés. Garder ARM64, compilation sans CGO et repos léger. Rust seulement pour un besoin mesuré, pas une réécriture spéculative. Aucun service Python permanent.
- Médias en lecture seule. Jamais de catalogue personnel, comptes, clés, configuration privée ou notes de travail dans Git. Fixtures synthétiques et répertoires temporaires pour les tests.
- Les adaptateurs natifs actuels sont du code approuvé embarqué. Une signature de paquet n’autorise pas l’exécution native arbitraire. Ne pas affaiblir les contrôles de transport, taille, signature, permissions ou révision.
- Un lecteur web personnalisé n’ajoute pas les codecs absents du navigateur. Ne pas promettre du transcodage lourd sur Pi sans mesure ni afficher de succès fictif.
- La licence propre au projet reste à choisir par le propriétaire. Ne pas inventer une licence générale ou une autorisation de redistribution.

## Méthode de travail

Privilégier commandes et tests adaptés ; éviter les vérifications répétitives par navigateur. Le propriétaire peut participer aux essais visuels. Expliquer les résultats en français accessible et brièvement. Les critères de sortie font foi, pas un pourcentage calculé artificiellement.

Vérifier l’état Git avant d’éditer et préserver les changements existants. Ne pas utiliser le catalogue ou les médias personnels comme fixtures. Ne pas lancer d’agents parallèles sauf demande explicite.

Suivre l’arborescence documentée. Mettre les tests auprès du code concerné. Un déplacement impose de vérifier imports, `go:embed`, scripts, Docker, CI, exemples et liens. Ne pas supprimer une migration ou un ancien paquet signé sans démontrer qu’aucun parcours de compatibilité n’en dépend.

Commandes de référence depuis la racine : `npm ci`, `node scripts/project.mjs check`, `node scripts/project.mjs build`, `node scripts/test-minimal.mjs`, `node scripts/check-repository.mjs`. Pour une réorganisation, utiliser `node scripts/project.mjs ci`. Voir le [guide de construction](docs/development/BUILD.md).

Actualiser STATUS, ROADMAP et CHANGELOG lorsqu’un changement les affecte. Le README reste une introduction, pas une accumulation des notes de versions. Ne jamais afficher un secret dans un audit. Une fuite historique confirmée exige d’informer le propriétaire et de préparer sa rotation ; une réécriture d’historique doit être explicitement coordonnée.
