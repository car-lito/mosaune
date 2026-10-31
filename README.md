**0.0.46 :** Films, Séries et Musique regroupent leurs composants extraits et leur écran, avec opérations communes et réglages avancés. [Essayer les familles](ESSAYER-LES-FAMILLES-046.md).

**0.0.45 :** TMDB livré comme un paquet installé et réellement retirable ; aucune réactivation sans réinstallation. [Essayer le retrait et le retour](ESSAYER-TMDB-045.md).

**0.0.44 :** TMDB utilise uniquement son module de traitement, avec pause persistante et aucun moteur natif de remplacement. [Essayer et comprendre la migration](ESSAYER-TMDB-044.md).

**Étape 0.0.43 :** [catalogue, profils et affiches dans TMDB 0.5.0](ESSAYER-TMDB-043.md).

Écrans médias : [composition remplaçable — 0.0.42](ESSAYER-LES-ECRANS-042.md). [Checklist du découpage : environ 68 %](AVANCEMENT-EXTENSIONS.md).

# Mosaune

[![Vérifications](https://github.com/car-lito/mosaune/actions/workflows/verify.yml/badge.svg)](https://github.com/car-lito/mosaune/actions/workflows/verify.yml)

Un espace pour vos films, séries et musiques, hébergé chez vous. Mosaune réunit une bibliothèque locale, des fiches enrichies et la découverte de contenus dans une interface commune. Le projet privilégie un serveur léger en Go et une architecture extensible.

**Prototype 0.0.46 — développement actif.** Reconnaissance, tags audio, catalogues, saisons, descriptions, en-têtes et comportements des fiches sont exécutés dans des extensions installables et remplaçables. Les familles médias complètes, fournisseurs/connecteurs restants et lecteurs restent à terminer avant le premier essai NAS. HTTPS direct et images Docker amd64/arm64 sont préparés pour les essais ; ce n’est pas encore la bêta. [Plan avant le NAS](PLAN-AVANT-NAS.md).

## Déjà disponible

- Films, séries et musique : dossiers persistants, identification, affiches, recherche, grandes fiches et profils.
- Reprise de lecture, file musicale, playlists, favoris et liste personnelle.
- Comptes locaux, invitations, permissions par bibliothèque et type de média.
- Demandes Radarr/Sonarr et suivi des arrivées après configuration.
- Thèmes déclaratifs, catalogue de dépôts GitHub et pilote TMDB installable en Wasm signé.

Les fonctions des familles médias restent partiellement intégrées au serveur. La reconnaissance des noms fonctionne déjà en modules indépendants installables ; leurs programmes externes restent limités à ce contrat. Les fournisseurs exécutables restent les paquets TMDB signés reconnus. Désactiver TMDB arrête effectivement ses accès, sans effacer les fiches et affiches conservées. Les métadonnées et images dépendent des services configurés.

## Commencer

- [Construire et tester le projet](DEVELOPPEMENT.md)
- [Comprendre le dépôt et récupérer un exécutable de test](GITHUB-PREMIERS-PAS.md)
- [Configurer TMDB](CONFIGURER-TMDB.md) et [Radarr/Sonarr](CONFIGURER-LES-DEMANDES.md)
- [Plan avant le NAS](PLAN-AVANT-NAS.md)
- [Contrats et moteur d'extensions](extension-kit/README.md)
- [Créer un thème](ESSAYER-LES-THEMES.md)
- [Historique des étapes](HISTORIQUE.md)

Les vérifications automatiques reconstruisent l'interface et les extensions, testent Windows et Linux et produisent aussi un exécutable Linux ARM64. Une compilation ARM64 ne remplace pas les essais sur Raspberry Pi.

Les paquets officiels sont distribués dans [car-lito/mosaune-extensions](https://github.com/car-lito/mosaune-extensions). Ils ont leur propre cycle de mise à jour, distinct de celui du serveur.

## État et limites

Le lecteur utilise encore les capacités du navigateur : les codecs non pris en charge ne sont pas convertis automatiquement. Les applications Android/Android TV, le mode hors ligne, les autres types de médias et le magasin universel d'extensions restent prévus. Ne pas exposer ce prototype sur Internet. Le serveur écoute par défaut uniquement sur 127.0.0.1 ; l’accès réseau nécessite une adresse HTTPS et un certificat explicitement configurés.

Les médias et données personnelles restent hors du dépôt. Les tests ne nécessitent pas de clé API personnelle. Ne jamais publier le dossier data, une clé privée de signature ou un export de configuration contenant des secrets.

## Licence

La licence du code Mosaune reste à choisir. L'accès public aux sources ne vaut pas encore licence open source. Les dépendances gardent leurs propres licences ; voir leurs manifestes et les notices présentes dans licenses/ et [NOTICES-METADONNEES.md](NOTICES-METADONNEES.md).
