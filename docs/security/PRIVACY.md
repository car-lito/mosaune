# Confidentialité et audit du dépôt

## Règles permanentes

Ne pas publier `data/`, `media/`, bases SQLite, sessions, clés API, clés privées de signature ou TLS, configurations réelles et notes de travail privées. Les tests utilisent des secrets factices et des données temporaires. Un mot de passe de fixture n’est jamais un mot de passe d’installation.

Les clés **publiques** de vérification des paquets et les signatures sont destinées à être distribuées. Les supprimer casserait la confiance des paquets ; ne pas les confondre avec une clé privée. Les noms publics du dépôt et le pseudonyme GitHub de son propriétaire restent nécessaires aux liens et publications officielles.

Les exemples de déploiement sont fictifs et doivent être adaptés. `127.0.0.1`, les adresses de documentation et les domaines `example`/`test` ne sont pas des adresses personnelles. Les URL avec utilisateur/mot de passe dans les tests servent à vérifier leur refus : ne pas supprimer ces tests sous prétexte qu’ils contiennent les mots « secret » ou « password ».

## Passe du 26 novembre 2026

Périmètre initial : 651 fichiers suivis (666 contenus examinés avec les entrées d’archives), 2 267 contenus examinés dans les objets Git accessibles localement, dont contenus textuels d’archives (10 archives courantes, 43 occurrences historiques). Les catégories recherchées comprenaient chemins utilisateurs, marqueurs personnels connus, adresses privées, e-mails, clés privées, formats de jetons de services et URL contenant des identifiants. Les cas signalés ont été relus : exemples fictifs, tests de rejet et crédits de test. Les commits inspectés utilisent un pseudonyme public et une adresse GitHub `noreply`.

**Aucune clé privée, aucun jeton de service reconnaissable ni donnée personnelle confirmée n’ont été trouvés dans ce périmètre.** Il n’a donc pas été nécessaire de réécrire l’historique ou de retirer un secret connu. Les adresses réseau d’exemple actuelles ont été rendues explicitement fictives dans la documentation. Les notes privées locales restent exclues du dépôt.

Limites : recherche par motifs et revue ciblée, pas une certification d’absence de secrets. Des valeurs opaques sans format connu peuvent échapper à cette passe. Les données personnelles locales non suivies n’ont pas été exportées ni auditées ; les secrets configurés dans GitHub, forks, objets inaccessibles et services tiers ne font pas partie du périmètre. Les binaires/archives sont inspectés de manière bornée, pas par décompilation exhaustive.

Le contrôle `node scripts/check-repository.mjs` empêche notamment la réintroduction de dossiers de données, clés privées et formats de jetons courants parmi les fichiers suivis. Il ne remplace pas la revue humaine des configurations et rapports.

## Si une fuite est découverte plus tard

Ne pas recopier la valeur dans un ticket ou un chat. Identifier le fichier et la nature du secret, prévenir le propriétaire, révoquer/renouveler le secret concerné puis retirer sa publication. Préparer une réécriture d’historique seulement en coordination explicite avec les contributeurs ; un simple commit de suppression n’efface pas l’historique, les clones ou les caches.
