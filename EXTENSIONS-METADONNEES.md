## Présentation dans la recherche — 0.0.25

Le champ facultatif `presentation` traverse les fiches normalisées et les contrats du moteur. Le type affiché, les mots-clés et les proportions ne sont pas codés en dur dans le renderer. Voir le [contrat et l’exemple Jeu/Console](extension-kit/presentation/v1/README.md).

## Cycle de vie hôte — 0.0.24

L’activation du fournisseur est indépendante du choix natif/Wasm. Le pilote TMDB expose `PUT /api/v1/metadata/providers/tmdb/state` (administrateur, corps `{ "enabled": false }`). La liste des fournisseurs expose `enabled`, distinct de `configured`.

Le contrat interne facultatif `lifecycleProvider` fournit un état et des contextes annulables appartenant à l’hôte. Une opération complète garde son contexte jusqu’à la persistance ; les écritures finales utilisent `providerCommit` avant toute transaction SQL. La désactivation annule les contextes existants ; une réactivation ne les ressuscite pas. Les invités Wasm ne peuvent pas modifier cet état.

La désactivation ne supprime ni les données communes ni les secrets. Les images déjà enregistrées restent consultables, même si la référence de cache a expiré. Aucun téléchargement d’image manquante n’est effectué. Les limites de stockage restent appliquées. Les autres fournisseurs ne sont pas arrêtés. Ce contrat ne remplace pas encore le cycle générique de désinstallation des futurs paquets.

# Fournisseurs de métadonnées — contrat initial v1

> **État actuel 0.0.19 :** le programme TMDB peut être installé et actualisé depuis une archive signée indépendante du serveur. [Guide](ESSAYER-LES-FOURNISSEURS-019.md), [contrat](extension-kit/PAQUETS-FOURNISSEURS.md). Le catalogue et les clés restent gérés par l’hôte ; les opérations d’images et de suggestions spécialisées demeurent natives. Les notes précédentes ci-dessous sont historiques.


> État actuel 0.0.17 : recherche et fiches riches TMDB peuvent passer par le module WASI. Nouvelle capacité facultative metadata.video-details.v1, données validées puis remises au catalogue commun. Images et suggestions spécialisées restent intégrées. Activation : Réglages → Extensions → TMDB. Les notes 0.0.16 ci-dessous sont historiques.


> Mise à jour du 15 octobre 2026 : Mosaune 0.0.16 relie maintenant la recherche TMDB au [kit d’extensions](extension-kit/README.md), avec un service réseau contrôlé par l’hôte. Activation facultative dans les paramètres ; mêmes associations et même cache. [Essayer](ESSAYER-TMDB-EXTENSION.md) · [Contrat des services hôte](extension-kit/SERVICES-HOTE.md). Les fiches riches et images utilisent encore les contrats intégrés ci-dessous. L’installateur de fournisseurs exécutables reste à faire.

Décision du 10 octobre 2026, à la demande du porteur du projet : les fournisseurs sont séparés des modules de lecture. Un fournisseur pourra couvrir plusieurs médias, et plusieurs fournisseurs pourront couvrir un même média.

## Disponible en 0.0.4

Le contrat `MetadataProvider`, défini dans metadata_provider.go, comporte quatre opérations :

| Opération | Responsabilité |
|---|---|
| Info | Identifiant, nom, version du contrat, médias pris en charge, besoin de clé, domaines réseau déclarés et attribution |
| Search | Recherche d'une œuvre depuis un titre, un type et éventuellement une année |
| Details | Fiche normalisée depuis un identifiant opaque et, si pertinent, une saison et un épisode |
| Artwork | Octets d'une image référencée par le fournisseur, avec accès au secret si nécessaire |

Les identifiants externes sont des chaînes associées à leur fournisseur. TMDB convertit les siens en nombres dans son adaptateur ; le noyau n'impose pas ce format aux autres fournisseurs.

Le noyau conserve les clés, le catalogue, les images normalisées, la provenance des champs et les choix de correspondance. Les tags locaux sont extraits séparément. Une sélection manuelle n'est pas remplacée par l'identification automatique. Une modification du fichier invalide sa fiche précédente.

Les écrans de recherche obtiennent la liste des fournisseurs via l'API. La configuration v1 accepte un secret textuel optionnel ; un fournisseur sans clé peut fonctionner directement. Le guide d'inscription et le logo TMDB restent spécifiques à cet adaptateur.

TMDB est le seul fournisseur livré. Un second fournisseur factice dans les tests vérifie le remplacement sans changement des routes ou du catalogue, avec des identifiants d'œuvres non numériques.

## Capacité facultative de découverte — 0.0.6

Un fournisseur peut aussi implémenter `DiscoveryProvider.Discover(ctx, secret, query)`. `DiscoveryQuery` porte le type de média, la page et soit un titre à rechercher, soit un identifiant d'œuvre servant de point de départ. `DiscoveryPage` retourne les fiches légères et le nombre de pages. Le noyau annonce automatiquement `discovery: true` dans la description du fournisseur. Le contrat de métadonnées v1 reste inchangé et un ancien fournisseur fonctionne sans cette capacité.

TMDB est le premier adaptateur : recherche films/séries et recommandations par œuvre, langue française, 20 résultats par page, 50 pages maximum. Les identifiants des autres fournisseurs restent opaques. La déduplication combine fournisseur, type et identifiant ; elle ne fusionne pas deux fournisseurs par titre.

Les œuvres et leurs choix utilisateur sont séparés des fichiers dans la table `works` (migration SQLite v4). Le noyau conserve favoris et À voir, calcule la disponibilité depuis les fichiers identifiés et disponibles, normalise les images et protège l'accès aux modules désactivés. Les affiches passent par un proxy local à partir de références fournisseur enregistrées ; aucun URL arbitraire n'est accepté du navigateur. Une panne de fournisseur n'efface pas les listes. Pas de nouvelle dépendance ni de moteur de recommandation local.

Tests : fournisseur de découverte factice avec identifiants opaques, coexistence avec le fournisseur v1 sans découverte, cache, pagination TMDB simulée, erreurs, disponibilité, conservation des listes et masquage du module vidéo. Le fournisseur est toujours du code intégré de confiance ; cette interface ne crée pas encore un système de plugins isolés et installables.

## Ajouter un fournisseur aujourd'hui

Depuis 0.0.7, deux autres capacités facultatives existent : `ShelfProvider.Shelf` pour les sélections éditoriales/sorties, et `ProfileProvider.Profile` pour les personnes et leurs œuvres, avec pagination. Le noyau filtre les types selon les modules activés. Les identifiants restent propres au fournisseur.

MusicBrainz est un deuxième fournisseur réel, sans clé, limité ici aux éditions musicales. Les requêtes JSON sont bornées à 2 Mio, espacées d'au moins 1,1 seconde et mises en cache 24 heures ; images via Cover Art Archive, redirections HTTPS limitées aux hôtes Archive.org attendus. Catalogue partagé avec TMDB, images normalisées et protégées par les mêmes limites. Les métadonnées MusicBrainz enrichissent une œuvre associée explicitement, sans remplacer les tags du fichier. Tracklist limitée à 200 pistes ; les albums et rééditions sont distingués par identifiant d'édition. Le débit limite et les erreurs externes ne bloquent pas les fichiers locaux.

Les sélections TMDB sont mises en cache six heures ; les autres requêtes TMDB conservent le cache sept jours. Les classements retournés ne sont pas présentés comme des recommandations calculées à partir de tous les goûts de l'utilisateur.

1. Implémenter le contrat dans un fichier Go séparé. Borner les lectures, requêtes, délais et caches ; ne jamais renvoyer de clé dans les erreurs.
2. L'ajouter à `catalog.providers` dans openCatalog. Le premier fournisseur configuré compatible est choisi pour l'identification automatique ; les recherches manuelles permettent de choisir le fournisseur.
3. Ajouter les tests de correspondance, d'erreurs et d'attribution, puis recompiler Mosaune.

Cette procédure exige encore une compilation. Les fournisseurs intégrés sont du code de confiance exécuté dans le serveur. Les domaines déclarés décrivent leurs besoins ; ils ne constituent pas un confinement. L'adaptateur TMDB fixe actuellement ses hôtes et refuse les redirections et chemins d'images arbitraires.

## Prévu pour les extensions installables

- [ ] Manifeste : identifiant, version, compatibilité du contrat, médias et permissions.
- [ ] Installation/activation/désactivation sans recompilation ; désinstallation sans perte des fiches conservées.
- [ ] Processus isolé ou autre solution mesurée sur ARM64, quotas et accès réseau limité ; aucun accès direct aux médias ou au catalogue.
- [ ] Ordre des fournisseurs et repli explicite, protection des corrections manuelles.
- [ ] Configurations plus riches, protection et rotation des secrets.
- [ ] Évolution versionnée du modèle pour personnes, albums, livres et éditions.
- [ ] Tests de compatibilité publiés, contrôle des paquets, retour arrière et magasin communautaire.

Cette livraison prépare l'extension du code. Elle ne livre pas encore le magasin de plugins, leur installation à la volée ni leur isolation.

## Compléments 0.0.8

Le modèle accepte des images de personnes, saisons et miniatures de bandes-annonces. ArtistResolver est une capacité facultative de recherche de personnes par nom. La recherche d’édition précise utilise actuellement MusicBrainz ; un futur second fournisseur musical nécessitera de généraliser ce point d’entrée. Les plugins installables restent différés.

Les références d’images natives sont validées avant réseau : chemins TMDB, clés YouTube de onze caractères, UUID MusicBrainz et noms Commons JPEG/PNG. Les réponses Commons déterminent les URL HTTPS autorisées sur upload.wikimedia.org ou thumb.wikimedia.org ; les crédits restent du texte, pas du HTML exécuté. Les anciennes redirections HTTP Archive.org sont mises à niveau vers HTTPS, jamais exécutées en HTTP. Les JSON Commons/Wikidata sont bornés et cachés sept jours ; images normalisées dans le cache partagé, échecs dix minutes.


## Cadrage du 13 octobre 2026

La demande utilisateur détaille désormais modules média, fournisseurs, connecteurs et extensions d’interface/langue, droits des comptes, catalogue officiel et dépôts GitHub tiers. Voir [PLAN-AVANT-NAS.md](PLAN-AVANT-NAS.md). À la désinstallation, les données exclusives du fournisseur sont retirées ; les fiches incorporées au catalogue du module média et les ressources encore référencées sont conservées. Ce périmètre est planifié, pas implémenté par le contrat Go actuel.


## Capacité d’affiches alternatives — 0.0.12

PosterProvider.Posters fournit une liste de références, langues et dimensions pour une œuvre. TMDB est le premier adaptateur. Le noyau vérifie que le choix appartient à cette liste, télécharge et normalise les octets puis conserve la correction dans un stockage durable indépendant du cache du fournisseur. Le contrat est facultatif : un ancien fournisseur fonctionne sans proposer d’alternatives. Cette capacité Go ne rend pas le paquet installable à chaud. Références : [Images TMDB](https://developer.themoviedb.org/reference/movie-images), [langues des images](https://developer.themoviedb.org/docs/image-languages).
