# Mosaune

[![Vérifications](https://github.com/car-lito/mosaune/actions/workflows/verify.yml/badge.svg)](https://github.com/car-lito/mosaune/actions/workflows/verify.yml)

Un espace pour vos films, séries et musiques, hébergé chez vous. Mosaune réunit une bibliothèque locale, des fiches enrichies et la découverte de contenus dans une interface commune. Le projet privilégie un serveur léger en Go et une architecture extensible.

**Prototype 0.0.30 — développement actif.** Le mode local reste le démarrage par défaut. Cette version ajoute la [gestion des modules de reconnaissance](ESSAYER-LES-EXTENSIONS-030.md) : installation, activation, arrêt et retrait persistants, avec conservation du catalogue. Les familles médias complètes restent à extraire. Les modules média et les lecteurs restent à extraire/construire avant le premier essai NAS. Elle conserve HTTPS direct, les images Docker amd64/arm64 et un catalogue CasaOS 0.4.15 ; ce n’est pas encore la bêta ni le premier déploiement validé sur NAS. [Préparation réseau et conteneurs](deploy/README.md).

## Déjà disponible

- Films, séries et musique : dossiers persistants, identification, affiches, recherche, grandes fiches et profils.
- Reprise de lecture, file musicale, playlists, favoris et liste personnelle.
- Comptes locaux, invitations, permissions par bibliothèque et type de média.
- Demandes Radarr/Sonarr et suivi des arrivées après configuration.
- Thèmes déclaratifs, catalogue de dépôts GitHub et pilote TMDB installable en Wasm signé.

Les modules média sont encore intégrés au serveur. Les fournisseurs communautaires exécutables ne sont pas encore ouverts ; seuls les paquets pilotes reconnus sont exécutés. L'interrupteur du pilote TMDB revient actuellement au moteur intégré : la désactivation complète du fournisseur figure dans les travaux obligatoires avant NAS. Les métadonnées et images sont fournies par les services configurés ; leur disponibilité dépend de ces services.

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
