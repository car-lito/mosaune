# Décisions produit en vigueur

Consolidées le 26 novembre 2026. Ce document transmet les intentions convenues, y compris non développées. Lire [STATUS](../STATUS.md) pour le niveau de réalisation.

## Vision, nom et priorités

Mosaune est un serveur multimédia personnel auto-hébergé, réunissant collection locale et découverte dans une interface commune. Films, Séries et Musique sont prioritaires. Livres audio, podcasts, ebooks et vidéos YouTube viendront comme familles optionnelles. Le nom **Mosaune / Projet Mosaune** a été retenu pour sa sonorité ; aucun acronyme n’est imposé. Le dépôt ne garantit aucune disponibilité de domaine ou de marque.

Le projet vise la contribution communautaire et un magasin gratuit d’extensions. Préférer des outils gratuits et open source, en respectant leurs licences. La licence de Mosaune n’est pas encore choisie.

Ordre convenu : **terminer les extensions → lecteurs personnalisés vidéo et musique → installation neuve sur NAS**. Le propriétaire accepte de repartir d’une base vide pour cet essai. Une migration depuis le poste de développement n’est donc pas un préalable. Les chemins des médias se configurent via les volumes du conteneur ; aucun chemin personnel n’est codé dans l’application.

La sobriété doit permettre une utilisation sur Raspberry Pi 5 avec 8 Go de RAM. Go reste le langage serveur ; Rust peut servir à une partie précise si un besoin mesuré le justifie. Éviter les services lourds permanents et les réécritures spéculatives. Garder Windows et Linux amd64/ARM64 construisibles. Les budgets RAM/CPU et limites de lecture doivent être mesurés, pas promis à partir d’une compilation réussie.

## Modularité et distribution

- Le squelette fournit comptes, permissions, stockage partagé, transactions, exécution contrôlée, gestion des extensions et composition UI.
- Une installation classique expose quelques grandes familles : Films, Séries, Musique. Leurs sous-composants restent remplaçables ou cumulables, avec réglages avancés. Pas des centaines de cartes techniques.
- Fournisseurs, connecteurs et thèmes à usage autonome peuvent avoir leur carte. Dépendances partagées : une seule installation, retrait contrôlé.
- Installer, mettre à jour, sélectionner, désactiver, retirer et restaurer doivent fonctionner réellement. Une fonction absente ou défaillante ne doit pas revenir silencieusement par un moteur caché.
- Désactiver conserve les données déjà acquises. Désinstaller supprime les données privées et secrets du paquet quand plus rien ne les utilise. Métadonnées, affiches, progressions et autres données communes restent dans le stockage partagé : pas une bibliothèque parallèle par fournisseur.
- Préserver les remplaçants sélectionnés et les ressources utilisées par une autre famille. Afficher les dépendances et conflits avant une opération.
- Dépôt officiel facilement accessible et possibilité d’ajouter des dépôts GitHub communautaires. Un paquet non officiel ne reçoit aucune permission implicite.
- Contrats pour ajouter médias, fournisseurs, résultats de recherche, pages, menus, réglages et langues. Libellés et formats d’aperçu extensibles (album, jeu, console, auteur…), sans une nouvelle branche centrale pour chaque domaine.

## Interface et parcours

Interface moderne, lisible et chaleureuse, inspirée des usages Plex/Spotify sans copie de leurs assets, marques ou écrans. L’intuitivité prime sur les effets. Ne pas exposer les détails d’implémentation dans les parcours destinés au public.

Accueil en rangées par média : reprendre avec progression, ajouts récents, sorties et suggestions pertinentes. Rubriques médias plus spécialisées. Marquer lu/non lu pour gérer la reprise. Films au cinéma et albums à venir doivent pouvoir être gardés dans la liste puis demandés au service compatible.

Recherche visible dans l’en-tête, résultats déroulants adaptés à la rubrique, locaux et distants distingués. Albums carrés, films verticaux, type sous le titre. Pas de page complète de recherche ni d’onglet Découvrir imposés. Suggestions notamment sur les grandes fiches des contenus.

Fiches films/séries/albums : informations, acteurs/artistes cliquables, projets associés, images des saisons, pistes/épisodes et contenus similaires. Bandes-annonces dans la langue de l’application si disponibles, sinon autre langue. Morceaux absents distincts et demande de récupération. Playlists éditables, réordonnables, avec pochette personnalisée. Lecture d’un album au survol/focus ; sur mobile, accès par sa fiche acceptable.

Thèmes déclaratifs : palettes, polices, tailles, arrondis et états par tokens ; valeurs communes et variantes explicites par rubrique. La navigation ne doit pas perdre le thème en changeant de page. Les lecteurs seront thémables aussi. Menus et réglages extensibles, paramètres classés, colonne des paramètres défilant indépendamment. Masquer les barres de défilement sans empêcher le défilement ni le clavier. Les interrupteurs inactifs doivent rester contrastés.

Extensions en cartes avec interrupteur, fiche de réglages agrandie au clic et onglets installées/à télécharger. Sans gestionnaire de téléchargement compatible, expliquer ce qui manque au lieu d’afficher un faux succès.

## Comptes et données

Premier administrateur créé à l’installation. Mot de passe haché avec sel, jamais en clair ou sous un chiffrement réversible présenté comme un stockage de mot de passe. Comptes locaux et droits par bibliothèque/type/action ; paramètres serveur réservés à l’administrateur.

Champ mot de passe facultatif à la création par l’admin. Vide : l’utilisateur choisit son mot de passe via une invitation sécurisée. L’administrateur peut réinitialiser un mot de passe oublié, pas lire l’ancien. Ne pas permettre de prendre un compte en revendiquant seulement son nom au premier passage.

Langue, thème, favoris, listes et progression doivent appartenir au bon compte. Aucun secret ni export complet de configuration dans le dépôt ou les rapports de bug.

## Découverte et automatisation

Recommandations locales et distantes basées sur les goûts/historique, même pour les médias absents du serveur. Métadonnées et fichiers médias restent distincts. Les demandes passent par Radarr/Sonarr ou un connecteur compatible ; Mosaune n’est pas un nouveau client BitTorrent.

Prévoir le suivi de chanteurs, acteurs et auteurs : détecter leurs nouveaux projets, proposer ou demander selon les préférences, puis notifier la disponibilité réelle. Ne pas marquer un contenu disponible parce que son téléchargement a seulement démarré. Respecter quotas, sources, langues et licences.

## Lecteurs et clients

La [spécification des lecteurs](PLAYERS.md) détaille les interactions. Films/Séries partagent un lecteur vidéo fourni par défaut, Musique son lecteur audio. Ils sont remplaçables par des extensions compatibles et thémables. Une session commune conserve file, état et progression.

Vidéo : bandeau inférieur persistant avec aperçu, commandes, vue agrandie et plein écran. Musique : bandeau persistant et clic sur la pochette pour la vue immersive, avec indication au survol/focus. Navigation sans coupure, pistes audio, sous-titres et reprise. À terme : chapitres, intros/génériques, musique au niveau d’usage Plexamp/Spotify, livres audio au niveau Audiobookshelf/Audible. Ce sont des objectifs, pas des équivalences déjà atteintes.

Une base open source est autorisée : ne pas recréer les codecs. Distinguer conteneur, codec, lecture directe, remux et transcodage. Les moteurs peuvent différer entre web et applications natives. Aucun transcodage universel promis sur Pi.

Clients : web PC, Android mobile avec hors-ligne, Android TV avec télécommande et essais prioritaires sur Chromecast compatible APK. iOS reste optionnel selon effort/coûts. Explorer la reprise sur l’accueil Android TV/Google TV et le lien entre profils TV et comptes Mosaune : vérifier les API et consentements au moment du chantier, sans supposer l’accès aux comptes Google ni une intégration acquise.

YouTube : objectif d’un lecteur intégré confortable, si possible sans publicité, éventuellement basé sur un projet existant. Faisabilité, maintenance et conditions des services restent à étudier ; ne pas promettre une suppression garantie des publicités ni intégrer cette famille au noyau.

## Collaboration

Le propriétaire apprend le développement : français accessible, guides concrets, étapes testables. Il préfère les commandes à l’interface GitHub pour limiter les coûts et peut participer aux tests visuels. Ces décisions doivent rester dans le dépôt pour qu’un contributeur puisse reprendre sans accès au chat.
