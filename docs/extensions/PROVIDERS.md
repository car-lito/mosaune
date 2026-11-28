## Références d’images — 0.0.84

Un adaptateur approuvé peut implémenter `ImageReferenceProvider.ImageReferenceKinds(ref string) []string` pour exposer ses images par `GET /api/v1/metadata/images/{provider}?ref=…`. Cette méthode doit être pure, rapide et sans entrée/sortie : elle valide une référence opaque et renvoie ses types de médias autorisés. Une liste vide refuse la référence. Elle ne télécharge rien ; `Artwork` reste responsable de l’obtention des octets via les transports autorisés et bornés du fournisseur.

Le serveur limite la référence à 600 octets et la liste à 16 types, sans doublon ni type absent de `Info().Kinds`. Il faut au moins un même type autorisé à la fois par la composition et par le compte. Ces contrôles précèdent toute consultation du cache : un cache existant ne contourne pas une référence devenue invalide ni les permissions. L’inventaire expose `imageReferences`, calculé depuis l’interface réellement implémentée, sans se fier à une valeur déclarée dans `Info()`.

Le cache partagé conserve son fonctionnement : index par fournisseur et référence, images normalisées en JPEG, taille bornée, réponses privées et maintien des images déjà stockées lorsque le fournisseur est désactivé. Une image manquante ne déclenche alors aucun appel au fournisseur. Le cycle de vie invalide les réponses en cours et empêche leur mise en cache après désactivation, même si l’adaptateur ignore l’annulation.

TMDB accepte ses chemins d’affiches et références `yt:` validées pour Films/Séries. MusicBrainz réutilise la validation `VisualQuery` du contrat musical pour les éditions, groupes d’éditions et fichiers Commons. Un autre adaptateur peut définir une grammaire différente sans changer le routeur. La déclaration de ce contrat ne constitue pas une autorisation réseau et n’ajoute pas de chargeur de code natif téléchargé.

## Correction d’association musicale — 0.0.83

Le contrat facultatif `AlbumMatchProvider.MatchAlbums(context.Context, credential, AlbumMatchQuery)` reçoit `title`, `artist`, `id` et `page`. Le titre et l’artiste sont limités à 250 octets, l’identifiant à 200, les pages à 1–50. Sans identifiant, le titre doit contenir au moins deux octets. L’adaptateur valide ses propres identifiants et liens ; il renvoie `errAlbumMatchInput` pour une saisie invalide, sans accès réseau arbitraire.

Le résultat `AlbumMatchResult` contient une `DiscoveryPage` (20 résultats maximum, pages entre 1 et 50) et `Detailed`. Ce dernier permet de remplacer un résumé déjà en cache par les informations complètes d’une édition, notamment ses pistes. Le serveur impose l’identité du fournisseur sélectionné, conserve les œuvres dans le stockage commun et vérifie le cycle de vie avant chaque écriture et avant de livrer les résultats.

`POST /api/v1/music/matches/{provider}` exige l’accès musical, une famille musicale autorisée et un fournisseur chargé prenant en charge ce type. Il ne choisit jamais une autre source à sa place. L’inventaire des fournisseurs expose `albumMatches`, calculé depuis l’interface effectivement implémentée. La fenêtre de correction sélectionne les sources activées, configurées et disponibles ; s’il y en a plusieurs, l’utilisateur choisit explicitement.

L’ancienne route `POST /api/v1/music/matches` reste un alias MusicBrainz déclaré dans son adaptateur. La validation de ses liens `/release/` et sa recherche d’éditions ne résident plus dans le routeur commun. L’association finale conserve la route et les contrôles existants sur les bibliothèques ; elle n’altère pas les fichiers locaux. Ce contrat concerne les adaptateurs hôtes approuvés, pas le chargement de code natif arbitraire.

## Profils et identification d’artistes — 0.0.82

L’inventaire `GET /api/v1/metadata/providers` expose `profiles`, `resolveArtists` et `available`. Le serveur calcule ces capacités depuis les interfaces `ProfileProvider` et `ArtistResolver` réellement implémentées ; une valeur annoncée par `Info()` ne suffit pas. L’identification requiert également les profils et le type `music`. L’interface retient les fournisseurs activés, configurés et disponibles. Plusieurs sources impliquent un choix explicite, sans préférence fondée sur leur nom.

`POST /api/v1/people/{provider}/resolve` reçoit `{ "name": "…" }` et renvoie les candidats `Contributor`. Cette opération musicale exige les permissions correspondantes, même si le fournisseur propose aussi des films. Son appel utilise `beginProvider`, puis `providerCommit` avant de livrer le résultat. Désactiver puis réactiver le fournisseur ne rend pas une ancienne réponse valide. Le client conserve le fournisseur des candidats et utilise ensuite `GET /api/v1/people/{provider}/{id}?page=…`.

Les liens de profils déjà identifiés gardent leur fournisseur. Sans source d’identification disponible, les albums locaux de l’artiste restent affichés. Cette étape ne généralise pas encore la correction d’association musicale, la route des images, ni le chargement d’adaptateurs natifs depuis un dépôt.

## Distribution depuis un dépôt — 0.0.81

Pour recevoir des paquets depuis un dépôt, un adaptateur implémente `providerRepositoryManager` en plus du contrat de gestion. Sa méthode `repositoryPackageIDs()` déclare les identifiants exacts des paquets acceptés, au maximum 128, sans doublon. Cette déclaration appartient au code hôte approuvé ; le catalogue distant ne peut pas l’ajouter ni la remplacer. Un gestionnaire local qui ne déclare pas cette capacité reste utilisable par import local.

Le schéma du dépôt ne change pas. Pour une entrée `kind: provider`, Mosaune cherche un seul adaptateur chargé qui déclare son `id`. Aucun résultat ou plusieurs résultats bloquent le téléchargement. Il n’y a ni repli TMDB, ni premier résultat choisi arbitrairement. Une fois le fichier téléchargé, son empreinte est vérifiée, puis `parseProviderPackage` assure la vérification propre à l’adaptateur. Le manifeste vérifié doit correspondre à l’identifiant, la version, le nom, l’éditeur, la description et la version minimale annoncés dans le dépôt.

`GET /api/v1/extension-repositories` fournit `providerBindings` (`packageId`, `providerId`, `name`) et les versions installées des gestionnaires chargés. La réponse au téléchargement porte `X-Mosaune-Package-Provider`, produit par le serveur après résolution et validation. L’interface vérifie ce destinataire puis utilise les routes nommées de l’adaptateur pour l’aperçu et l’installation. Ces opérations revalident le fichier et l’état attendu ; télécharger un paquet ne l’installe pas.

Le test de référence utilise un second adaptateur et un format de test signé avec une autre clé. Il couvre le parcours de dépôt jusqu’à l’installation, l’inventaire des versions, les permissions, le refus avant accès réseau, les métadonnées trompeuses et une signature invalide malgré une empreinte correcte. TMDB conserve son propre format et sa clé ; les anciens catalogues restent compatibles.

## Cartes et réglages des fournisseurs — 0.0.80

`GET /api/v1/metadata/provider-management` renvoie aux administrateurs les fournisseurs effectivement chargés, leurs états et leurs commandes disponibles : `activation`, `packages`, `runtimeControl` et `configurationOwner`. Ces capacités proviennent des interfaces implémentées par l’adaptateur approuvé, jamais d’une simple déclaration dans un paquet. Sans fournisseur, la liste est vide et les autres réglages restent accessibles.

L’interface utilise ces descriptions pour construire les cartes et un même composant de détail. Elle charge l’état d’un paquet uniquement quand sa fiche est ouverte ; une erreur de ce paquet n’empêche pas l’affichage des autres extensions. Le formulaire de configuration existant est relié par son propriétaire, sans identifiant TMDB dans la page Extensions. Le sélecteur d’import local présente les adaptateurs qui savent vérifier et installer des archives ; changer de fournisseur annule l’aperçu précédent.

Pour une commande d’exécution distincte de l’activation globale, implémenter `providerRuntimeManager` : `providerRuntime()` retourne les états publics et `setProviderRuntime(ctx, enabled)` applique la transition contrôlée. La route administrateur `PUT /api/v1/metadata/providers/{id}/runtime` exige un booléen `enabled` explicite. L’adaptateur reste responsable de ses transactions, de l’annulation des appels et du refus d’une activation impossible. TMDB réutilise sa logique existante ; son ancienne URL est conservée comme alias.

L’état d’exécution utilise les clés communes `installed`, `enabled`, `safeMode` et `serviceAccess` lorsqu’elles sont pertinentes. Les fournisseurs sans commande propre sont affichés sans interrupteur fictif. MusicBrainz ne reçoit pas de nouveaux boutons d’installation : ses composants restent gérés dans la famille Musique.

Cette étape généralise les cartes, fiches, formulaires et imports locaux. Le catalogue distant conserve son parcours de distribution pilote ; elle ne change pas le format signé TMDB et ne charge pas automatiquement un adaptateur natif téléchargé.

## Gestion des paquets fournisseurs — 0.0.79

Un adaptateur approuvé peut implémenter `metadataPackageProvider` pour proposer le cycle de vie de ses paquets. Le routeur commun ne connaît ni sa base de données, ni sa clé de signature, ni son identifiant de fournisseur. Il applique l’authentification, la restriction administrateur, la limite de 9 Mio, la confirmation de l’empreinte du fichier, la sérialisation des mutations et les réponses communes.

L’adaptateur assure la vérification cryptographique, la compatibilité du manifeste avec son moteur et Mosaune, les contrôles de versions, la persistance atomique et la gestion du programme chargé. Les contrôles de l’état attendu doivent être refaits à la mutation : un aperçu n’accorde aucune autorisation et ne réserve aucune version. Une restauration doit vérifier les empreintes de la version actuelle **et** de la version précédente. Le retrait ne doit pas effacer les médias ni les données communes consommées par d’autres composants.

API administrateur, `id` étant l’identifiant de l’adaptateur chargé :

| Méthode | Chemin | Fonction |
| --- | --- | --- |
| GET | `/api/v1/provider-packages` | Adaptateurs qui implémentent réellement le contrat |
| GET | `/api/v1/provider-packages/{id}` | Paquet installé, version précédente et état |
| POST | `/api/v1/provider-packages/{id}/preview` | Vérification d’une archive envoyée dans le corps |
| POST | `/api/v1/provider-packages/{id}/install` | Installation avec `X-Mosaune-Package-Checksum` et `X-Mosaune-Previous-Checksum` |
| POST | `/api/v1/provider-packages/{id}/rollback` | Restauration avec JSON `checksum` et `previousChecksum` |
| DELETE | `/api/v1/provider-packages/{id}` | Retrait avec JSON `checksum` |

Les anciennes URL `/preview` et `/install` sans identifiant restent des alias TMDB vers les mêmes contrôles. L’URL historique `/example` reste réservée aux exemples TMDB. Les exemples signés et la lecture de la version précédente appartiennent désormais à cet adaptateur, pas au routeur commun. Un fournisseur absent ou sans contrat de paquets renvoie 404 ; un compte ordinaire reste refusé avant tout appel à l’adaptateur.

Les tests couvrent un deuxième adaptateur simulé, l’isolation des fournisseurs, les permissions, les aperçus sans mutation, les échecs et états périmés, ainsi que de vrais paquets signés TMDB sur les nouvelles URL. Le format signé `packageformat` reste le format pilote TMDB ; cette étape ne prétend pas fournir un chargeur natif universel ni terminer les écrans spécifiques des fournisseurs.

## Composition au lancement — 0.0.78

`-providers tmdb,musicbrainz` sélectionne les adaptateurs hôte à construire, dans cet ordre. `-providers none` n’en construit aucun. La variable `MOSAUNE_PROVIDERS` fournit la valeur par défaut ; une option explicite est prioritaire. Identifiants inconnus, doublons et éléments vides sont refusés avant ouverture du catalogue. Sans configuration, les deux adaptateurs habituels restent chargés.

Pour une installation de développement minimale, dans un **nouveau dossier dédié** :

```sh
./mosaune -data-dir ./minimal-data -providers none -initial-media none -initial-home=false
```

Remplacer `./mosaune` par le nom de l’exécutable téléchargé. Le code de configuration affiché au démarrage permet de créer l’administrateur. Les comptes, paramètres et gestionnaire d’extensions restent accessibles ; Musique peut ensuite être préparée avec ses composants locaux, sans fournisseur en ligne.

Le filtre `-providers` s’applique à **chaque lancement**, contrairement aux options `-initial-media` et `-initial-home`, qui ne réinstallent rien dans une base existante. Omettre un adaptateur conserve ses paquets, clés et choix enregistrés. Pour utiliser ensuite ce fournisseur, le réinclure au prochain démarrage et installer/activer son paquet si nécessaire : le retour d’un adaptateur ne réinstalle pas silencieusement un paquet absent.

`node scripts/test-minimal.mjs [chemin-exécutable]`, après construction, vérifie ce parcours sur un catalogue temporaire : création administrateur, installation de Musique seule, paramètres et redémarrage. Le script n’utilise aucun média personnel et ne publie ni code de configuration ni session.

Les fabriques sélectionnables restent des adaptateurs hôte approuvés dans `suppliedMetadataProviders`. Cette option ne charge pas de code Go arbitraire depuis un dépôt et ne retire pas les programmes embarqués du binaire.

## Dépendance à un composant sélectionné — 0.0.77

Un adaptateur hôte peut implémenter moduleBackedProvider et retourner providerModuleBinding (catalogue, emplacement). Le lien vise l’emplacement et suit le choix enregistré, pas l’identité d’un éditeur. MusicBrainz utilise music-results ; un autre adaptateur peut utiliser un autre emplacement sans modifier le mécanisme commun.

providerModuleAvailable consulte la sélection, la présence du paquet et son activation dans le registre, ainsi que le mode de secours. Ce contrôle léger ne charge pas le programme en mémoire ; son intégrité est vérifiée à l’ouverture effective de l’opération. Une erreur de lecture rend le service indisponible. La liste des fournisseurs et les appels de découverte reflètent ce statut.

beginProvider acquiert une opération du cycle de vie des exécutables et conserve le descripteur et le programme sélectionnés. Les appels imbriqués pour le même lien réutilisent ce contexte ; ils ne libèrent pas l’opération extérieure. Mélanger deux liens différents dans une même opération est refusé. Le décodage musical utilise cet instantané plutôt que de rouvrir une sélection susceptible de changer.

providerCommit protège l’enregistrement final avec le verrou de ce cycle de vie, puis l’éventuel état propre au fournisseur. Le verrou doit être acquis avant toute transaction SQL et relâché après sa validation. Désactivation, retrait, remplacement ou arrêt invalident les appels existants ; réactiver le paquet ne réactive pas leurs anciens contextes. Les méthodes sans liaison et les fournisseurs à cycle de vie propre conservent leur comportement.

Les adaptateurs et leurs connexions HTTP restent disponibles dans la composition hôte pour prendre en charge les installations ultérieures. Les programmes lourds ne sont ouverts qu’à la demande. Le contrat n’autorise aucun programme invité à accéder directement à SQL ni à déclarer des destinations réseau arbitraires. Les fournisseurs autonomes et la composition minimale complète restent à finaliser.

## Installation initiale des adaptateurs — 0.0.75

La fabrique peut déclarer InstallDefault(context.Context, *sql.Tx) error. Ce contrat appartient au code hôte approuvé ; aucun programme invité ne reçoit la connexion SQL. La composition officielle attribue cette fonction à TMDB ; MusicBrainz n’a pas de paquet initial à installer par ce mécanisme.

Sur une base neuve, provider_bootstrap enregistre les identifiants avant les migrations. Les anciennes étapes créent leurs tables mais reportent le provisionnement TMDB. Après les migrations, tous les installateurs sélectionnés et le marqueur completed sont validés dans une seule transaction (schéma 51). Un installateur doit utiliser la transaction reçue et éviter les effets externes irréversibles. Échec ou interruption conservent la sélection pour la reprise ; une fabrique manquante, un doublon ou un enregistrement illisible échoue sans installation partielle. Une nouvelle valeur par défaut n’est pas ajoutée à une reprise.

Une composition initiale vide reste vide après réouverture avec les fabriques officielles. Les bases existantes gardent leur parcours historique et leurs choix, sans rejouer les installateurs. Les paquets retirés ne réapparaissent pas. L’installation manuelle d’un fournisseur absent ne reçoit pas automatiquement les accès accordés à la composition officielle initiale.

Cette étape ne fournit pas encore de sélecteur utilisateur au premier démarrage. Les familles médias et leurs migrations ne suivent pas encore toutes ce contrat ; un catalogue sans fournisseur n’est donc pas un noyau entièrement vide.

## Composition des adaptateurs hôte — 0.0.74

Le catalogue reçoit une liste de fabriques identifiées (metadataProviderFactory). La liste officielle se trouve dans provider_composition.go ; les réglages privés de démarrage sont dans les adaptateurs. Le contrat metadataProviderRuntime complète MetadataProvider avec StartMetadataProvider(context.Context) et CloseMetadataProvider() error. Ces fonctions appartiennent au code hôte approuvé, pas à un programme invité pouvant accéder au catalogue ou à sa base.

Avant toute création, la composition vérifie les identifiants, les doublons et les constructeurs. Un constructeur retournant une instance partielle avec une erreur doit la rendre refermable. Une erreur de démarrage, une annulation ou une identité différente entraîne la fermeture de toutes les instances créées, en ordre inverse. Les erreurs initiales et de fermeture sont conservées. La liste n’est exposée aux requêtes qu’une fois tous les adaptateurs démarrés ; les adaptateurs ne doivent pas consulter une composition partiellement publiée. Un constructeur reste responsable des ressources qu’il ne retourne pas.

La fermeture du catalogue annule d’abord les appels des composants partagés, puis ferme les adaptateurs en ordre inverse avant la base. TMDB annule également ses contextes fournisseurs et interdit définitivement de nouvelles opérations sur cette instance ; aucune préférence persistante n’est changée. Les appels MusicBrainz relèvent du moteur partagé de reconnaissance, annulé avant fermeture ; son adaptateur libère les connexions HTTP inactives.

openCatalogWithProviders permet de tester une composition vide ou de remplacement. L’ouverture habituelle conserve TMDB et MusicBrainz. Depuis 0.0.75, l’installation initiale suit le contrat décrit plus haut. Ce point d’entrée interne ne signifie ni noyau seul achevé, ni chargement de fournisseurs arbitraires depuis GitHub, ni nouvelle option utilisateur.

## Présentation dans la recherche — 0.0.25

Le champ facultatif `presentation` traverse les fiches normalisées et les contrats du moteur. Le type affiché, les mots-clés et les proportions ne sont pas codés en dur dans le renderer. Voir le [contrat et l’exemple Jeu/Console](../../extension-kit/presentation/v1/README.md).

## Cycle de vie hôte — 0.0.24

L’activation du fournisseur est indépendante du choix natif/Wasm. Le pilote TMDB expose `PUT /api/v1/metadata/providers/tmdb/state` (administrateur, corps `{ "enabled": false }`). La liste des fournisseurs expose `enabled`, distinct de `configured`.

Le contrat interne facultatif `lifecycleProvider` fournit un état et des contextes annulables appartenant à l’hôte. Une opération complète garde son contexte jusqu’à la persistance ; les écritures finales utilisent `providerCommit` avant toute transaction SQL. La désactivation annule les contextes existants ; une réactivation ne les ressuscite pas. Les invités Wasm ne peuvent pas modifier cet état.

La désactivation ne supprime ni les données communes ni les secrets. Les images déjà enregistrées restent consultables, même si la référence de cache a expiré. Aucun téléchargement d’image manquante n’est effectué. Les limites de stockage restent appliquées. Les autres fournisseurs ne sont pas arrêtés. Ce contrat ne remplace pas encore le cycle générique de désinstallation des futurs paquets.

# Fournisseurs de métadonnées — contrat initial v1

> **État actuel 0.0.19 :** le programme TMDB peut être installé et actualisé depuis une archive signée indépendante du serveur. [Guide](../archive/versions/ESSAYER-LES-FOURNISSEURS-019.md), [contrat](../../extension-kit/PAQUETS-FOURNISSEURS.md). Le catalogue et les clés restent gérés par l’hôte ; les opérations d’images et de suggestions spécialisées demeurent natives. Les notes précédentes ci-dessous sont historiques.


> État actuel 0.0.17 : recherche et fiches riches TMDB peuvent passer par le module WASI. Nouvelle capacité facultative metadata.video-details.v1, données validées puis remises au catalogue commun. Images et suggestions spécialisées restent intégrées. Activation : Réglages → Extensions → TMDB. Les notes 0.0.16 ci-dessous sont historiques.


> Mise à jour du 15 octobre 2026 : Mosaune 0.0.16 relie maintenant la recherche TMDB au [kit d’extensions](../../extension-kit/README.md), avec un service réseau contrôlé par l’hôte. Activation facultative dans les paramètres ; mêmes associations et même cache. [Essayer](../archive/versions/ESSAYER-TMDB-EXTENSION.md) · [Contrat des services hôte](../../extension-kit/SERVICES-HOTE.md). Les fiches riches et images utilisent encore les contrats intégrés ci-dessous. L’installateur de fournisseurs exécutables reste à faire.

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

Les écrans de recherche obtiennent la liste des fournisseurs via l'API. Un fournisseur sans clé peut fonctionner directement.

**Réglages déclarés (0.0.86).** Un adaptateur qui implémente `MetadataConfigurationProvider` annonce `configurationOwner` dans `GET /api/v1/metadata/providers`. L'écran « Affiches et fournisseurs » affiche alors le formulaire générique de `GET/PUT /api/v1/extensions/{owner}/configuration` : titre, description, étapes, liens et champs viennent du schéma signé du paquet (`settings.json`), pas d'un écran propre au fournisseur. Sans schéma disponible, l'écran indique que le formulaire manque au lieu d'afficher un champ de clé par défaut.

L'ancienne route `PUT/DELETE /api/v1/metadata/providers/{provider}` (`{ "secret": "…" }`) reste pour compatibilité. Elle applique les règles du **seul** champ secret déclaré par le fournisseur (longueurs, format `token`) ; plusieurs secrets ou un schéma indisponible sont refusés, sans emprunter les limites d'un autre fournisseur. Un adaptateur sans schéma garde des bornes génériques (non vide, 2048 caractères, sans espace). Les réglages des connecteurs Radarr/Sonarr restent réservés à leur capacité partagée (`acquisitionconfig.ReservedOwner`). Le logo d'attribution TMDB du pied de page reste spécifique.

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

La demande utilisateur détaille désormais modules média, fournisseurs, connecteurs et extensions d’interface/langue, droits des comptes, catalogue officiel et dépôts GitHub tiers. Voir [Roadmap actuelle](../ROADMAP.md). À la désinstallation, les données exclusives du fournisseur sont retirées ; les fiches incorporées au catalogue du module média et les ressources encore référencées sont conservées. Ce périmètre est planifié, pas implémenté par le contrat Go actuel.


## Capacité d’affiches alternatives — 0.0.12

PosterProvider.Posters fournit une liste de références, langues et dimensions pour une œuvre. TMDB est le premier adaptateur. Le noyau vérifie que le choix appartient à cette liste, télécharge et normalise les octets puis conserve la correction dans un stockage durable indépendant du cache du fournisseur. Le contrat est facultatif : un ancien fournisseur fonctionne sans proposer d’alternatives. Cette capacité Go ne rend pas le paquet installable à chaud. Références : [Images TMDB](https://developer.themoviedb.org/reference/movie-images), [langues des images](https://developer.themoviedb.org/docs/image-languages).
