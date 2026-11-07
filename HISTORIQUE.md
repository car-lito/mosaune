## 0.0.65 — collecte du suivi pilotée par le composant installé

Le composant acquisition 0.7.0 fournit un plan de lectures borné : épisodes, saisons et pagination de la file. Les objets privés sont projetés en observations typées par le contrat commun côté hôte. Limites dépassées et pages intermédiaires incomplètes arrêtent le suivi ; aucun repli natif si le composant est ancien. Migration45 conservant retraits, désactivations et remplacements ; familles14.0/14.1. Le stockage des réglages, les prévisualisations et le rapprochement local restent à extraire. Le budget global des tests du serveur passe de20 à30minutes : la suite Windows précédente avait atteint sa limite globale pendant un test lancé depuis4secondes, sans échec d’assertion signalé.

## 0.0.64 — programme d’envoi fourni par l’extension

Le composant acquisition 0.6.0 construit les actions, les champs et les références nécessaires aux créations, mises à jour, surveillances et recherches. Le serveur valide leur périmètre puis résout les destinations et identifiants privés. Aucun envoi natif de secours ; un ancien composant doit être mis à jour. Les ressources existantes et les saisons non demandées sont préservées ; les épisodes dupliqués ou invalides sont refusés. Migration44, familles13.0/13.1. Le stockage de configuration et la préparation du suivi restent à extraire.

## 0.0.63 — formats des connecteurs dans le composant partagé

Le composant acquisition 0.5.0 déclare les champs JSON des réponses et des écritures, les objets imbriqués et les noms des recherches. Mosaune traduit ce format vers un vocabulaire commun sans transmettre les données privées au module. Les champs inconnus sont conservés, les grands entiers ne sont plus arrondis et les collisions/clé JSON dupliquées sont refusées. Une ancienne extension sans format exige une mise à jour, sans repli silencieux. Migration43 et familles12.0/12.1. La coordination métier et le stockage des réglages des connecteurs restent à extraire.

## 0.0.62 — formulaires des connecteurs déclarés par l’extension

Le composant partagé 0.4.0 fournit titres, instructions, libellés, ordre et limites des champs. Le serveur contrôle les sources des listes et les secrets, refuse les formulaires périmés avant le réseau et garde la pause accessible sans composant actif. Migration42, familles11.0/11.1. Les formats des objets API et la configuration métier restent à extraire.

## 0.0.61 — opérations réseau des connecteurs déclarées par l’extension

Le composant partagé 0.3.0 fournit les routes Radarr/Sonarr, leurs paramètres numériques, délais et limites de réponse. Le serveur appelle des opérations nommées et conserve les clés, le serveur et le sous-chemin configurés. Destinations absolues, traversées de chemins, changements de méthode et redirections sont refusés. Profil chargé une fois par session puis oublié ; désactivation et remplacement interrompent les appels. Migration41 et familles10.0/10.1. Un ancien remplaçant sans profil réseau exige une mise à jour, sans routes natives de secours. Les objets métier des API, formulaires et configuration restent à extraire.

## 0.0.60 — suivi des demandes remplaçable

Le composant partagé Radarr/Sonarr 0.2.0 interprète désormais les épisodes, les files de téléchargement et les fichiers locaux vérifiés : progression, blocages, importation et disponibilité partielle ou complète. Les observations et résultats sont bornés. Un ancien composant sans suivi doit être mis à jour, sans repli natif. Migration 40 avec désactivations, retraits et choix tiers conservés ; familles 9.0/9.1. Les échanges HTTP, les formulaires et le rapprochement des chemins restent intégrés.

## 0.0.59 — demandes de téléchargement remplaçables

Les familles Films et Séries partagent le composant Demandes Radarr et Sonarr. Son programme prépare les ajouts, les saisons surveillées et les recherches. Il peut être installé, remplacé, désactivé ou retiré depuis les composants de la famille. Un remplacement invalide une confirmation antérieure. La désactivation annule les échanges actifs, y compris le suivi.

Le serveur conserve les clés, les chemins existants, le contrôle des saisons autorisées et l’enregistrement préalable des demandes. Le transport HTTP et l’interprétation du suivi restent à extraire. Migration 39, familles fournies Films/Séries 1.1.0 et exemples 8.0/8.1 pour les contributeurs.

## 0.0.58 — sources musicales déclarées par l’extension

Le composant musical 0.4.0 fournit son profil de transport à travers le contrat describe-transport. MusicBrainz, Wikidata, Commons et les pochettes utilisent ses déclarations de sources, délais, limites et cache. Le serveur contrôle les destinations autorisées et conserve le stockage partagé.

La migration 38 met à jour le composant fourni 0.3.0 en conservant les choix d’activation. Les composants retirés, remplacés ou modifiés sont respectés. Les anciens modules utilisent leur politique historique. Les familles 7.0/7.1 distribuent le nouveau composant avec Mosaune 0.0.58 minimum.

## 0.0.57 — configuration et transport du paquet TMDB

Le paquet signé TMDB 0.7.0 déclare ses trois sources, leurs limites, la langue et la région du catalogue. Les anciens paquets restent lisibles ; les réglages personnels sont conservés lors d’un retour à une version antérieure. La migration ne remplace que l’archive d’origine 0.6.0 sans historique de remplacement.

## 0.0.56 — transport commun des fournisseurs

Validation locale : 214 tests serveur et 45 tests web réussis ; contrôles Go/Svelte, exécutables Windows/Linux amd64/ARM64 et paquet CasaOS construits. Les contrats du kit restent inchangés.

- TMDB, MusicBrainz et leurs sources d’images passent par un service HTTP commun : destinations autorisées, redirections contrôlées, limites de réponse, validation et cache partagé. Les contrats Wasm et les versions des familles restent inchangés.
- Attente de cadence interruptible, même quand une autre requête occupe la file. Les erreurs d’annulation conservent leur cause ; les résultats annulés ou invalides ne sont pas enregistrés dans le cache.
- Clé TMDB ajoutée uniquement à la requête sortante : les paramètres de l’appelant restent intacts, les clés du cache sont publiques. Messages d’erreur sans URL contenant le secret.
- Cadences conservées (TMDB 300 ms, MusicBrainz 1 100 ms), mêmes limites de réponse et enveloppe globale du cache. Les images déjà conservées ne sont pas supprimées.
- Le transport est un service du noyau, pas une nouvelle extension visible. Les adaptateurs, autorisations déclaratives de nouvelles sources et configurations restent à généraliser avant des fournisseurs entièrement indépendants. Aucun déploiement NAS.

## 0.0.55 — images musicales pilotées par l’extension

Validation : 208 tests serveur, 45 tests web, tests du kit, contrôles Svelte/Go et trois exécutables réussis. Essai HTTP sur installation neuve avec composant 0.3.0, désactivation/réactivation et redémarrage vérifiés. Aucun déploiement NAS.

- Le composant Musique décide des sources et replis pour les pochettes et portraits ; il interprète les relations MusicBrainz, les images Wikidata/Commons et les crédits. Cette logique n’est plus compilée dans le noyau.
- Contrat visual/describe-visuals v1 avec état opaque, plans contrôlés, six étapes maximum et délais bornés. Les images binaires restent hors du programme Wasm. Aucun accès réseau direct ou nouvelle carte d’extension.
- Désactivation : requêtes en cours annulées, images déjà conservées disponibles même après expiration, aucun résultat négatif enregistré du fait de la pause. Le cache reste commun.
- Composant fourni 0.3.0, migration 36 conservant états, retraits et remplaçants. Nouveaux exemples de familles 6.0/6.1 ; anciens paquets publiés inchangés.
- Le transport HTTP, la cadence, la configuration et les adaptateurs restent à généraliser. Aucun déploiement NAS.

## 0.0.54 — requêtes musicales pilotées par l’extension

Validation locale : 205 tests serveur, 45 tests web, tests du kit et contrôles Svelte/Go réussis. Après le dernier ajustement de migration, contrôles musicaux ciblés et compilation des trois exécutables relancés avec succès.

- Le composant musical prépare les recherches et leurs paramètres : albums, correction titre/artiste, artistes, éditions, discographies et prochaines sorties. Les expressions MusicBrainz quittent les parcours correspondants du noyau.
- L’hôte valide le plan avant tout appel : chemins relatifs bornés, identités, paramètres et pagination ; aucune URL, méthode, clé ou en-tête arbitraire. Le transport, les images et le cache restent à extraire.
- Migration 35 du composant fourni 0.1.0 vers 0.2.0, avec état et sélection conservés. L’identité réservée, l’absence d’import et l’intégrité des octets enregistrés sont vérifiées ; les compilations officielles avec différentes versions de Go sont prises en charge. Les remplaçants externes restent choisis et doivent annoncer le protocole de requêtes v1 ; aucune implémentation native de secours.
- Archives d’exemple 5.0/5.1 et guide du contrat mis à jour ; les anciens fichiers publiés sont immuables.
- Compilation des programmes intégrés commune à Docker et aux builds locaux, à partir des fichiers déclarés dans Go. Corrige l’échec Docker 0.0.53 qui omettait music-results ; les validations Windows/Linux de cette version avaient réussi.

## 0.0.53 — interprétation MusicBrainz dans un composant remplaçable

Validation : 203 tests serveur réussis (dont 17 relancés après adaptation des scénarios de migration), 45 tests web, tests du kit, contrôles Svelte/Go et trois exécutables construits. Essai HTTP sur installation neuve : désactivation/réactivation du composant et redémarrage vérifiés. Aucun déploiement NAS.

- Albums, éditions, pistes, crédits et artistes sont interprétés par un programme Wasm installé dans la famille Musique. L'implémentation Go correspondante n'est plus liée au noyau.
- Contrat metadata.music-results.v1, validation à l'entrée et à la sortie, absence de service réseau ou d'accès aux fichiers pour le programme. Le serveur conserve le transport MusicBrainz, le cache et les images : le fournisseur n'est pas encore entièrement extrait.
- Désactivation/retrait sans repli natif, annulation des opérations actives, choix d'un paquet externe et conservation du cache commun. Migration 34 : les choix musique désactivés ou absents restent respectés ; aucune réinstallation au redémarrage après retrait.
- Regroupement dans Musique ; exemples de familles 4.0/4.1 avec versions minimales adaptées. Les archives publiées précédemment ne sont pas remplacées.
- Réponses JSON limitées à 512 Kio pour ce composant, enveloppe 1 Mio et mémoire linéaire plafonnée à 64 Mio, sans changement des plafonds globaux. Une réponse trop volumineuse échoue explicitement et conserve les métadonnées existantes.
- Test des conteneurs corrigé : la version attendue du paquet TMDB initial est 0.6.0. C'était la cause de l'échec du contrôle conteneur 0.0.52, après réussite des tests Windows/Linux.

## 0.0.52 — découverte déclarée par le paquet TMDB installé

Validation : 201 tests serveur réussis (dont 10 contrôles réseau relancés après autorisation de l’environnement), 45 tests web, Svelte sans erreur ni avertissement, tests du kit et analyses Go réussis. Trois exécutables construits ; essai HTTP sur installation neuve avec le paquet signé 0.6.0 et redémarrage réussi. Aucun parcours navigateur ni déploiement NAS.

- TMDB 0.6.0 emporte ses recherches, suggestions et sélections dans son manifeste signé. Le serveur lit le paquet installé ; mise à jour, retour arrière, retrait et redémarrage suivent sa déclaration.
- Une déclaration vide reste vide. Les archives historiques restent compatibles et inchangées ; aucun paquet retiré n'est recréé. Une installation neuve reçoit TMDB 0.6.0, les installations existantes gardent leur version jusqu'à une mise à jour explicite.
- Cache borné à la petite déclaration, renouvelé selon l'empreinte du paquet ; aucune conservation supplémentaire du programme Wasm en mémoire. Vérification sous le verrou du moteur avant un appel de découverte.
- Aucune nouvelle carte d'extension, migration de base ou permission réseau. Les adaptateurs génériques, MusicBrainz et les connecteurs restent à extraire.
- [Vérifier cette étape](ESSAYER-LE-PAQUET-TMDB-052.md).

## 0.0.51 — déclaration commune de recherche et de découverte

Validation : 200 tests serveur, 45 tests web, contrôles Svelte/Go et essai HTTP sur installation temporaire réussis. Trois exécutables reconstruits ; aucun déploiement NAS.

- Contrat discovery/v1 : types recherchables, libellés, suggestions similaires et sélections nommées, avec sélecteur de type ou de contributeur.
- Recherche sur plusieurs fournisseurs compatibles, jusqu’à six requêtes ; erreurs d’une source indépendantes des autres. Les messages et champs de recherche ne citent plus TMDB en dur.
- Les lignes de découverte ne connaissent plus MusicBrainz : le fournisseur indique si des contributeurs identifiés sont requis. Trois contributeurs visibles maximum, aucun nom deviné.
- Les écrans peuvent référencer une source et une sélection nommées sans liste spécifique aux fournisseurs dans le noyau. Le serveur refuse les fonctions non déclarées ; les déclarations n’accordent aucun droit réseau supplémentaire.
- Compatibilité de recherche pour les anciens adaptateurs sans déclaration, absence explicite respectée. Pas de migration de base ni nouvelle carte d’extension.
- Limite : déclarations livrées par les adaptateurs actuels, pas encore chargeables depuis tout paquet fournisseur externe. MusicBrainz et des services de découverte restent intégrés. [Guide](ESSAYER-LA-DECOUVERTE-051.md).

## 0.0.50 — index de recherche fourni par les catalogues

Validation : 40 tests web, kit, contrôles Svelte/Go, quatre tests serveur ciblés et essai HTTP isolé réussis. Trois exécutables reconstruits. Suite serveur complète locale interrompue pour lenteur ; contrôles GitHub à consulter pour sa validation intégrale.

- Les composants de catalogue fournissent leurs candidats de recherche : groupe/fichier, titres, mots-clés, priorités et présentation. Films, Séries et Musique restent trois familles visibles.
- Recherche locale générique, accents neutralisés, plusieurs mots, titres exacts prioritaires, dédoublonnage des œuvres distantes contre groupes et fichiers locaux.
- Séries et épisodes, albums et morceaux peuvent être proposés séparément. Index fourni seulement à partir des métadonnées autorisées, sans nouvelles permissions ni moteur à chaque frappe.
- Migration SQLite 33 des seuls catalogues officiels installés 0.1 vers 0.2, conservant états et remplaçants. Les anciens composants gardent une recherche générique par nom de fichier ; un index explicitement vide est respecté.
- Exemple contributeur enrichi ; familles d’exemple 3.0/3.1 exigeant 0.0.50, sans écraser les archives publiques 2.x.
- La recherche distante, la découverte et les autres services métier restent à extraire. [Guide](ESSAYER-LA-RECHERCHE-050.md).

## 0.0.49 — retour arrière coordonné des familles

Validation locale : 195 tests serveur réussis, 3 essais optionnels ignorés, 33 tests web, Svelte sans erreur ni avertissement, kit et analyses Go réussis. Exécutables Windows/Linux amd64/Linux ARM64 construits. Parcours HTTP réel sur une installation isolée : archives officielles 2.0 puis 2.1, redémarrage, retour à 2.0 avec activation conservée, second redémarrage, réinstallation de 2.1. Aucun contrôle visuel navigateur ni déploiement NAS effectué.

- Point de restauration persistant créé atomiquement lors d’une mise à jour de famille ; retour depuis sa carte avec aperçu et confirmation.
- Restauration des programmes et de l’écran modifiés, tout en conservant les états, réglages, autorisations, composants partagés et remplaçants choisis.
- Refus des aperçus périmés et des composants modifiés séparément ; un échec laisse tout inchangé. Un seul point est conservé par famille, consommé après retour et supprimé lors du retrait.
- Schéma SQLite 32. Les mises à jour précédant cette version ne reçoivent pas d’historique inventé. Les composants réinstallés sans version antérieure ne peuvent pas être restaurés.
- Aucun nouveau service métier extrait ni déploiement NAS. [Guide d’essai](ESSAYER-LE-RETOUR-ARRIERE-049.md).

## 0.0.48 — familles dans le catalogue GitHub

Validation finale : 188 tests serveur validés, 3 essais optionnels ignorés, 33 tests web, Svelte sans erreur ni avertissement, kit et analyses Go, trois exécutables Windows/Linux amd64/Linux ARM64. Un premier essai local avait dépassé ses délais pendant des constructions concurrentes ; les deux tests concernés puis la suite complète finale ont réussi. Parcours HTTP sur serveur temporaire avec le vrai GitHub : ancienne archive 1.1 vers 2.0, mise à jour 2.1, activation et état conservés après redémarrage. Aucun test visuel navigateur ni déploiement NAS dans cette étape.

- Téléchargement vérifié des archives de familles depuis les dépôts ; contrôle des empreintes, manifestes, limites et compatibilité avant prévisualisation.
- Versions installées visibles, aperçu partagé entre catalogue et import local, retour vers la fiche de la famille après installation.
- Catalogue officiel v2 distinct pour préserver les anciens clients ; export de six exemples publics via `-export-family-repository`. Aucun accès aux données du serveur pendant cet export.
- Exemples 2.0/2.1 pour ne pas réutiliser les versions locales 1.x avec des octets différents ; estampilles Git retirées des constructions WASM distribuées.
- Pas de nouvelle migration de base, de nouveau service métier extrait ni de déploiement NAS. [Guide](ESSAYER-LE-CATALOGUE-048.md).

## 0.0.47 — archives de familles installables

Validation locale : 185 tests serveur validés, 33 tests web, kit et analyses Go, Svelte sans erreur ni avertissement, trois exécutables Windows/Linux amd64/Linux ARM64. Trois essais serveur optionnels ignorés. Parcours HTTP sur serveur Windows temporaire : installation, activation, mise à jour, redémarrage, retrait et réinstallation ; outil de création d’archive exécuté et empreintes vérifiées. Vérification visuelle non terminée : connexion au compte fictif refusée par le contrôle automatique du navigateur. Aucun déploiement NAS.

- Format d’archive borné regroupant les composants exécutables existants et un écran ; import local avec aperçu statique et contrôle de l’ensemble avant validation.
- Installation et mise à jour transactionnelles, états et droits conservés, remplaçants non écrasés, réutilisation des composants partagés, refus des aperçus périmés et des versions incohérentes.
- Exemples Films/Séries/Musique 1.0 et 1.1 téléchargeables, outil de préparation d’archives pour les contributeurs, regroupement des programmes importés dans la fiche de leur famille.
- Migration schéma 31. Catalogue distant de familles, restauration globale, nouveaux médias et services encore intégrés restent à terminer. [Guide utilisateur](ESSAYER-LES-FAMILLES-047.md).

## 0.0.46 — familles médias et composants avancés

Validation locale : 179 tests serveur validés (suite complète puis les six tests de regroupement après correction d’une fixture d’autorisation), 33 tests web, kit et analyses Go, Svelte sans erreur ni avertissement, trois exécutables Windows/Linux amd64/Linux ARM64. Trois essais serveur optionnels ignorés sur cette machine. Parcours HTTP avec redémarrage et parcours navigateur sur une installation temporaire. Aucun déploiement NAS.

- Manifestes de regroupement fournis pour Films, Séries et Musique : composants exclusifs/partagés, écran et version de composition.
- Préparation, désactivation et retrait des programmes déjà extraits et de leur écran dans une même transaction. Une erreur annule toute l’opération ; un aperçu périmé est refusé.
- Migration schéma 30 sans activation ni réinstallation implicite. Choix des remplaçants et droits existants conservés ; conflits vérifiés sur toutes les rubriques d’un écran alternatif.
- Cartes principales regroupées ; composants techniques fournis dans les réglages avancés et filtre du catalogue.
- Archives de familles, mises à jour coordonnées, cumul de contributions et services encore intégrés restent à terminer. [Essai et limites](ESSAYER-LES-FAMILLES-046.md).

## 0.0.45 — distribution du module TMDB

Validation locale : 173 tests serveur, 33 tests interface, kit et analyses Go, Svelte sans erreur ni avertissement, trois exécutables Windows/Linux amd64/Linux ARM64. Parcours HTTP sur un vrai serveur Windows temporaire : retrait, redémarrage, activation refusée sans paquet, réinstallation puis activation. Vérifications conteneurs déléguées à GitHub Actions ; aucun déploiement NAS.

- Provisionnement unique du paquet signé de démarrage dans le stockage et le registre communs ; exécution uniquement du paquet installé, suppression du module WASI de remplacement.
- Retrait persistant et activation impossible sans paquet ; réinstallation explicite par import ou catalogue, puis activation distincte.
- Migration schéma 29 conservant versions existantes, désactivation, refus réseau et réglages. Ancien module absent et en pause laissé absent.
- Interface distinguant paquet absent, installé et en pause ; guide ESSAYER-TMDB-045.md.
- Correction d’un test d’association musicale révélé sur Linux : vérifier la date conservée de chaque fichier, sans supposer un horodatage identique entre deux fichiers.
- Adaptateur réseau/configuration et identifiants Sonarr encore dans le noyau ; aucune revendication d’indépendance complète du fournisseur.

## 0.0.44 — suppression du moteur TMDB historique

Recherche, fiches et catalogue utilisent uniquement le module. Suppression des parseurs natifs dupliqués, migration unique du choix historique, pause persistante et arrêt sans substitution après retrait ou en mode de secours. Les appels tardifs sont annulés avant enregistrement ; l’identification interrompue reste en attente. Le module fourni et les services hôtes restent à extraire complètement. [Parcours d’essai](ESSAYER-TMDB-044.md).

## 0.0.43 — catalogue vidéo dans le module TMDB

TMDB 0.5.0 traite la découverte, les suggestions, les sélections d’accueil, les profils et les affiches alternatives dans son programme WASI. Contrat borné, requête de service contrôlée, réponses validées et refus de substitution silencieuse pour les anciens paquets. Paquet signé fourni au catalogue, guide utilisateur et tests de révocation. Le mode intégré historique demeure explicitement sélectionnable ; l’indépendance complète reste ouverte.

## 0.0.42 — écrans médias distribués en extensions

Quatre paquets fournissent Accueil/listes, Films, Séries et Musique. Les rangées sont déclarées dans leurs manifestes ; l’ancien Dashboard monolithique est retiré. Les pages peuvent disparaître et être remplacées par un paquet externe, avec contrôle des conflits et droits par compte. Cinéma compact est un exemple disponible dans le dépôt inclus. Provisionnement unique au schéma 27, conservation des données et accès aux réglages même sans écran actif. Services métiers et lecteurs restent à extraire.

## 0.0.41 — 29 octobre 2026

Pages et menus d’extensions via ui.pages.v1 : composition déclarative, collections et raccourcis, navigation et paramètres communs, accès par compte, retrait effectif des routes, dépôt/import et exemple Salon personnel. Les thèmes existants habillent les composants. Correction du scénario Docker 0.0.40 qui réactivait Musique avant de tester à tort la persistance des tags désactivés. [Guide et limites](ESSAYER-LES-PAGES-041.md).

# 0.0.40 — Ensembles de composants médias

- Gestion groupée de Films, Séries et Musique avec aperçu détaillé et application atomique.
- Composants de fiches partagés et remplaçants conservés, refus des aperçus périmés et annulation après validation seulement.
- Checklist pondérée du découpage, estimé à environ 60 %. Les familles complètes restent à terminer.
- Correction de la validation Windows 0.0.39 : moins de sérialisation dans les grosses fiches, budget borné adapté et durée globale de test portée à vingt minutes.

## 0.0.39 — comportements des fiches par extension

Commandes, artistes, format de couverture, pistes/fichiers, rapprochement des pistes manquantes et présentation des suggestions extraits ensemble. Rendu déclaratif, permissions conservées, exemple tiers et transport borné pour grandes éditions. [Guide](ESSAYER-LES-EXTENSIONS-039.md).

## 0.0.38 — En-têtes par extension

- Titres et informations du haut des fiches Films/Séries/Musique composés par un programme WASI remplaçable, avec le thème existant.
- Contrat public media.detail-header.v1 et exemple indépendant « Dans le cadre » ; import, dépôts, sélection, mise à jour et restauration communs.
- Distinction album/morceau contrôlée côté serveur ; permissions et associations vérifiées avant de transmettre les faits. Les notes portent le fournisseur réel.
- Désactivation sans perte de données ni remplacement automatique : titre simple et commandes conservés. Migration25 préservant les choix précédents.
- Affiches, liens d’artistes, commandes, suggestions et listes musicales restent intégrés ; aucun lecteur nouveau ni déploiement NAS.

## 0.0.37 — Rubriques descriptives par extension

- Informations des fiches : programme WASI indépendant pour les résumés, informations complémentaires, portraits et bandes-annonces Films/Séries.
- Contrat déclaratif media.detail-sections.v1 : blocs ordonnés, trois régions, quatre composants partagés et thémables, références contrôlées par le serveur.
- Installation, sélection explicite, désactivation, retrait, mise à jour et restauration via le moteur commun ; données conservées.
- Exemple externe « Fiche épurée », import et dépôts GitHub, migration24.
- En-têtes, commandes, suggestions, autres comportements et lecteurs restent à extraire ; aucun déploiement NAS.

## 0.0.36 — Saisons et listes d’épisodes par extension

- Nouveau module Fiches des séries installable, désactivable et remplaçable via le cycle commun ; migration23 sans changement des autres choix.
- Contrat déclaratif media.detail-list.v1, ordre/groupes/libellés fournis par le programme isolé, rendu partagé et thémable. Pas de HTML ou JavaScript externe.
- Liste simple lorsque le module manque, échoue ou dépasse ses limites. Fichiers, métadonnées et progression conservés.
- Exemple indépendant, import local et dépôts GitHub, tests de permissions, migration, remplacement, restauration et références invalides.
- Le reste des fiches et les autres comportements restent à extraire ; aucun déploiement NAS.

## 0.0.35 — Catalogues organisés par extensions

Trois programmes distincts fournissent les regroupements et le tri des films, séries et albums. Remplacement par paquet externe local ou dépôt, sélection explicite et cycle de vie partagé. Les métadonnées sont filtrées par permissions avant exécution ; aucune réécriture du catalogue. Module absent ou en panne : fichiers individuels et explication. Migration 22 sauvegardée, exemple contributeur, tests de contrats et de bibliothèque réelle. Les grandes familles médias ne sont pas encore entièrement extraites.

## 0.0.34 — 25 octobre 2026

- Lecteurs de métadonnées musicales tiers : import local, dépôts GitHub et choix explicite, sans substitution silencieuse.
- Cycle commun avec les reconnaisseurs : activation, retrait, mises à jour et restauration ; état et données conservés. Migration sauvegardée vers le schéma 21.
- Accès au seul fichier analysé affiché avant confirmation, contrat borné et refus des changements de rôle ou des accès supplémentaires.
- Exemple contributeur ID3v1 indépendant, deux versions et catalogue reproductible. Les modules média complets et leurs interfaces restent à extraire.

## 0.0.33 — 25 octobre 2026

- Lecture des informations musicales et pochettes déplacée dans un module WASI indépendant ; suppression du parseur natif du noyau.
- Nouvelle carte Informations des fichiers musicaux : activation, désactivation, désinstallation et réinstallation hors ligne. État persistant, catalogue et pochettes conservés.
- Service limité aux blocs du fichier en cours ; aucune ouverture de chemin ou connexion réseau par l’extension. Fichiers inchangés ignorés, analyse avant transaction et annulation des sessions après désactivation.
- Étape distincte de progression pour les informations musicales. Préparation temporaire des pochettes sur disque avec plafond.
- Migration du catalogue vers le schéma 20, contrats et guide contributeur. Import de lecteurs de tags tiers encore à ajouter.

## 0.0.32 — 24 octobre 2026

- Paquets de reconnaissance raccordés aux dépôts GitHub : cartes, filtre, versions installées, aperçu et installation/mise à jour via le même contrôle transactionnel que l’import local. Ouverture directe de leur fiche après installation.
- Téléchargement borné, concordance du manifeste, empreinte et politique hôte avant aperçu ; identité déclarative clairement signalée, signature du pilote TMDB inchangée. Aperçu invalidé après échec d’installation, nouvelle vérification requise.
- Catalogue contributeur reproductible avec deux versions, disponible dans les fichiers de construction et sur la branche publique `recognition-preview` du dépôt d’extensions. La branche principale reste compatible avec les anciens clients.
- Tests du cycle dépôt → installation → mise à jour → restauration hors ligne et des erreurs ; 122 tests serveur et 22 tests interface validés localement, compilations Windows/Linux/ARM64 réussies.
- Le système média complet, les contributions UI et les lecteurs restent à extraire. Aucun déploiement NAS. [Guide](ESSAYER-LES-EXTENSIONS-032.md).

## 0.0.31 — 23 octobre 2026

- Enveloppe de paquet exécutable locale (schéma 2), capacités et contributions versionnées, version minimale du noyau et empreinte. Lecture statique bornée, sans extraction ; source et éditeur non authentifiés clairement signalés. Le pilote TMDB signé reste distinct et inchangé.
- Import administrateur avec prévisualisation et confirmation liée aux octets et à la révision du registre. Validation du programme dans le moteur isolé avant installation ; mise à jour et historique précédent atomiques avec le registre, restauration disponible.
- Plusieurs reconnaisseurs installables pour un média, choix explicite persistant indépendant de l’activation. Aucun remplacement automatique après retrait ou désactivation ; changement de programme annule les résultats non enregistrés. Schéma 19, sauvegarde et conservation des états antérieurs.
- Cartes des modules externes, activation, sélection, restauration et retrait ; exemple contributeur indépendant et commande de fabrication de paquet. Tests du véritable scan avec un module importé.
- Reste à faire : téléchargement de ces paquets depuis les dépôts, contributions UI, familles médias complètes et dispatch des dépendances des futurs rôles. Les reconnaisseurs acceptés actuellement n’ont ni service ni dépendance. Aucun déploiement NAS.

## 0.0.30 — 22 octobre 2026

- Modules de reconnaissance enregistrés dans le registre commun, programmes installés dans un stockage privé et installation/réinstallation hors ligne depuis les versions officielles fournies. Migration au schéma 18 avec sauvegarde ; installation initiale unique.
- Activation, désactivation et retrait persistants, annulation des scans non enregistrés et refus des résultats annulés après réactivation. Écritures du catalogue et changements d’état ordonnés ; retrait atomique du registre et du programme, sans effacement des médias et métadonnées communes.
- Trois cartes administrateur, fiches détaillées et confirmation de retrait ; réinstallation depuis la rubrique disponible, activation explicite. Mode de secours appliqué aux reconnaisseurs.
- Ce cycle porte sur la reconnaissance uniquement : paquets communautaires, choix entre implémentations, tags, écrans et autres fonctions des familles médias restent à extraire. Aucun déploiement NAS.

## 0.0.29 — 21 octobre 2026

- Reconnaissance des fichiers extraite dans trois programmes WASI : Films, Séries et Musique. Formats et règles des noms retirés du serveur ; contrat versionné commun et réponses validées.
- Un moteur interprété temporaire par scan, appels par lots de 32, sans accès aux fichiers, au réseau ni aux secrets. Le mode interprété évite la compilation native initiale trop lente sous émulation ARM64, sans augmenter les limites du serveur. Validation avant la transaction ; catalogue précédent conservé en cas d’erreur.
- Progression distincte pour la reconnaissance des noms et l’enregistrement. Tags audio et corrections existantes conservés. Construction indépendante des trois programmes sous Windows/Linux et dans Docker.
- Ces composants sont encore livrés avec le serveur, pas installables individuellement. Les écrans, tags, catalogues spécialisés et connecteurs restent à extraire. Aucun déploiement NAS.

## 0.0.28 — 20 octobre 2026

- Autorisations persistantes par extension et opération, distinctes des demandes du manifeste. Aucun nouvel accord automatique lors des mises à jour ; une opération retirée puis réintroduite nécessite un nouvel accord.
- Premier adaptateur complet : TMDB. Accès en ligne commun aux moteurs natif et Wasm, modifiable dans sa fiche par un administrateur, avec protection contre les modifications concurrentes.
- Révocation : interruption des opérations en cours et refus des résultats tardifs, même après réautorisation ; conservation des clés, fiches et images enregistrées. Les identifications interrompues restent en attente.
- Catalogue schéma 17 avec sauvegarde. TMDB conserve une fois son accès historique et reste autorisé par défaut sur une installation neuve ; les nouveaux propriétaires génériques démarrent sans accord.
- Autres adaptateurs, aperçu des nouveaux accès dans les installateurs génériques et extraction des modules média restent à raccorder. Aucun déploiement NAS.

## 0.0.27 — 19 octobre 2026

- Réglages des extensions : stockage générique par propriétaire dans SQLite, schémas enregistrés, valeurs ordinaires et secrets séparés, révisions contre les écrasements concurrents. Les secrets ne sont jamais renvoyés par l’API ; pas de chiffrement au repos ajouté.
- Routes de configuration communes réservées à l’administrateur et liées aux modules enregistrés ; compatibilité des anciennes routes conservée. Rechargement proposé après une erreur.
- Mise à jour/restauration : refus de supprimer ou reclassifier une valeur existante sans migration explicite. Le formulaire, l’archive et le registre évoluent dans une même transaction.
- Catalogue schéma 16 avec sauvegarde : déplacement de la clé TMDB dans le stockage commun. Fournisseur intégré et paquet partagent cette clé, conservée lors du retrait du seul paquet.
- Autres familles, services/permissions génériques, migrations privées, chiffrement des secrets et extraction des modules média restent en cours.

## 0.0.26 — 19 octobre 2026

- Moteur : dépendances par capacités, choix explicites entre fournisseurs, protection des consommateurs actifs et désactivation en cascade.
- Registre commun SQLite : plans de changements, révisions contre les écritures périmées, choix persistants et restauration avec états réels ; aucune sélection silencieuse d’une alternative.
- Paquet TMDB installé : archive et registre modifiés dans la même transaction lors des installations, mises à jour, restaurations et retraits ; choix de moteur durable et contrôle avant démarrage. Section Dépendances et démarrage dans sa fiche.
- Migration du catalogue au schéma 15 avec sauvegarde et reprise des paquets existants vérifiés, sans exécuter de code à la migration.
- Le pilote TMDB refuse encore les dépendances externes. Extraction des modules média, autres installateurs, réglages privés et interface générale des choix restent en cours. Lecteurs personnalisés prévus après les extensions, installation NAS neuve ensuite.

## 0.0.25 — 18 octobre 2026

- Recherche : pochettes carrées pour la musique, affiches verticales pour films et séries, type visible sous le titre.
- Distinction Album/Single/EP selon les métadonnées MusicBrainz ; repli « Sortie musicale » pour un type non renseigné. Les morceaux restent distingués des albums.
- Contrat déclaratif partagé pour les extensions : libellé libre, mots-clés, ratio, forme et cadrage de l’aperçu. Validation à la frontière du module et avant stockage, renderer commun respectant le thème, exemple Jeu/Console documenté.
- Pas de nouveau module Jeux ni de recherche globale de personnes/saisons dans cette version ; ces futurs résultats pourront réutiliser le contrat.

## 0.0.24 — 18 octobre 2026

- Interrupteur TMDB global, distinct du moteur natif ou extension ; état conservé après redémarrage et en mode secours.
- Arrêt des nouvelles recherches, suggestions, fiches, images et identifications automatiques ; annulation des opérations et rejet de leurs résultats tardifs.
- Catalogue, affiches enregistrées, corrections, clé et choix du moteur conservés ; MusicBrainz indépendant.
- Recherche et accueil actualisés après le changement, avec explication de l’état désactivé.
- Migration du catalogue vers le format 14 avec sauvegarde préalable. L’ancienne préférence de moteur ne désactive pas automatiquement le fournisseur.

# Notes historiques de développement

Ces notes décrivent les étapes précédentes. Pour compiler la version actuelle, suivre DEVELOPPEMENT.md ; pour le périmètre actuel, lire README.md. Les mentions de version, de statut de publication et de chemins ci-dessous sont historiques.

# Projet Mosaune — premier atelier

## Version 0.0.22 — cartes sans doublon

La vue « Toutes les sources » regroupe les paquets identiques des catalogues embarqué et distant. Priorité au dépôt officiel ; le filtre Source conserve chaque dépôt accessible. Identifiant, version, empreinte et métadonnées doivent correspondre pour regrouper les cartes. Aucun paquet ni donnée supprimé. Fermez le serveur puis relancez DEMARRER-WINDOWS.cmd.

## Version 0.0.21 — dépôt officiel en ligne

Le [dépôt officiel](https://github.com/car-lito/mosaune-extensions) est publié et proposé dans Réglages → Extensions → À télécharger. Cliquez sur **Charger le catalogue officiel**. Ambre et TMDB sont récupérés depuis GitHub puis vérifiés avant installation. Dernier catalogue conservé hors ligne ; actualisation volontaire, aucune installation automatique. Un dépôt officiel déjà ajouté en 0.0.20 est repris sans doublon. [Guide pas à pas](ESSAYER-LE-DEPOT-OFFICIEL-021.md).

## Version 0.0.20 — catalogue d’extensions

Cartes disponibles, filtres, choix de version et installation directe depuis le catalogue. Sources GitHub publiques ajoutables, actualisation manuelle, cache conservé hors ligne et retrait sans désinstaller les paquets. Exemples intégrés disponibles sans GitHub. Le pilote exécutable reste limité aux paquets TMDB signés.

Guide : [Essayer le catalogue](ESSAYER-LE-CATALOGUE-020.md). Modèle complet : [dépôt d’exemple](packages/mosaune-repository-example.zip). Contrat auteur : [dépôts GitHub](extension-kit/DEPOTS-GITHUB.md). Fermer Mosaune et sauvegarder data avant de lancer DEMARRER-WINDOWS.cmd ; migration du catalogue au format 13. Le dépôt officiel est désormais publié (voir la 0.0.21 ci-dessus).

Prototype local 0.0.22 — comptes et permissions, catalogue et paquets d’extensions installables ; demandes Radarr/Sonarr, suivi des arrivées ; photos, saisons, correction des albums et lecture au survol ; accueil en rangées, recherche contextuelle, grandes fiches et profils, albums MusicBrainz, playlists avec pochettes et reprise de tous les médias. Ce n'est pas encore la bêta ni un serveur à déployer sur le NAS.

## Nouveautés et mise à jour

**0.0.19 — premier fournisseur installable :** paquets TMDB signés, aperçu de compatibilité et accès, contrôle de démarrage, installation à chaud, mise à jour conservant une version de retour et restauration. Deux paquets prêts à télécharger dans Réglages → Extensions → À télécharger : 0.4.0 puis 0.4.1. Le serveur reste en 0.0.19 pendant leur mise à jour. Base au format 12 avec sauvegarde préalable ; autres fournisseurs et clés communautaires non pris en charge à ce stade. [Guide de test](ESSAYER-LES-FOURNISSEURS-019.md), [contrat et publication](extension-kit/PAQUETS-FOURNISSEURS.md).

**0.0.18 — réglages déclaratifs des extensions :** formulaire TMDB décrit dans un fichier, contrôles typés côté serveur, secrets conservés si laissés vides et retrait explicite. Services déclarés par le module et vérifiés à chaque invocation. Interrupteurs éteints plus contrastés ; barres de défilement masquées, défilement natif conservé. Base inchangée, clé existante réutilisée. [Guide de la 0.0.18](ESSAYER-LA-018.md), [contrat pour les créateurs](extension-kit/configuration/README.md).

**0.0.17 — extensions en cartes :** navigation des paramètres à défilement indépendant, onglets Sur le serveur / À télécharger, recherche et grandes fiches de réglages. TMDB peut aussi produire ses fiches riches via le moteur d’extensions. Activation dans **Réglages → Extensions → TMDB**. Les modules média restent intégrés et la boutique reste à construire. [Guide pas à pas](ESSAYER-LES-EXTENSIONS-017.md).

Les notes suivantes décrivent les versions précédentes.

**0.0.16 — premier branchement réel du moteur d’extensions :** la recherche TMDB peut maintenant s’exécuter dans un module séparé. Activation facultative dans **Réglages → Métadonnées → Affiches et fournisseurs** ; même clé, même catalogue, corrections conservées, retour au moteur habituel et mode de secours. Les fiches riches et images restent natives ; les paquets exécutables communautaires ne sont pas encore installables. [Guide de test](ESSAYER-TMDB-EXTENSION.md), [contrat des services hôte](extension-kit/SERVICES-HOTE.md).

Pour reconstruire les sources, après installation des dépendances web, utiliser `scripts/build-app.ps1` : le programme TMDB est compilé en Wasm avant son inclusion dans le serveur, puis les versions Windows/Linux sont construites. Les tests du module séparé demandent aussi `extension-kit/scripts/build-lab.ps1`.

**Chantier des extensions exécutables (14 octobre) :** un [laboratoire séparé](extension-kit/README.md) exécute une extension de démonstration et teste ses limites, son annulation et son cycle de vie. Le lanceur **TESTER-LE-MOTEUR-EXTENSIONS.cmd** est prêt à l’emploi. Le serveur reste en 0.0.15 ; ce prototype n’installe pas encore de fournisseur dans l’application et ne touche pas au catalogue.

**0.0.15 :** refonte des thèmes : fichier déclaratif complet, palette réutilisable, polices, tailles, arrondis et états des composants. Même moteur pour Mosaune et les thèmes installés ; paramètres inclus, exceptions par rubrique facultatives. Ambre 2.0.0 et anciens paquets compatibles. Guide : [ESSAYER-LES-THEMES.md](ESSAYER-LES-THEMES.md). Les points ci-dessous décrivent les étapes précédentes.

**0.0.14 :** Réglages → Extensions, import de paquets ZIP de thèmes avec vérification, activation/désactivation, mise à jour et désinstallation. Choix personnel et autorisations d’usage par compte, exemple Ambre clair/sombre, mode de secours. À cette étape les paramètres gardaient leur apparence native ; ils suivent le thème depuis 0.0.15. Guide : [ESSAYER-LES-EXTENSIONS.md](ESSAYER-LES-EXTENSIONS.md), contrat pour les auteurs : [CREER-UNE-EXTENSION.md](CREER-UNE-EXTENSION.md). Les modules/fournisseurs/connecteurs restent intégrés ; dépôts GitHub et extensions exécutables ne sont pas encore livrés.

**0.0.13 :** premier administrateur, comptes invités ou créés avec un mot de passe, permissions par bibliothèque, sessions révocables et données personnelles séparées. Mon compte, Utilisateurs et thèmes clair/sombre/système. Guide : [ESSAYER-LES-COMPTES.md](ESSAYER-LES-COMPTES.md). Le serveur reste local à cet ordinateur.

**0.0.12 :** choix d’affiches de films/séries (propositions TMDB ou import JPEG/PNG), aperçu, retour à l’affiche automatique et stockage permanent séparé du cache. Paramètres répartis entre Bibliothèques, Métadonnées, Téléchargements et Serveur. Guide : [ESSAYER-LES-AFFICHES.md](ESSAYER-LES-AFFICHES.md). Première étape du plan avant NAS ; les extensions installables restent à réaliser.

**Depuis 0.0.11 :** identification vidéo continue en arrière-plan, progression par bibliothèque, pause/reprise persistante et erreurs relançables. Le scan des fichiers a sa propre barre de progression. Les imports Radarr/Sonarr identifiés par leur chemin exact passent en priorité. Guide : [ESSAYER-LE-SCAN.md](ESSAYER-LE-SCAN.md). Les fiches en attente reprennent sans rescanner les dossiers.

Depuis 0.0.10 : une liste de dossiers vide dans Téléchargements explique maintenant quoi faire. Le bouton « Ajouter un dossier de films/séries » permet de saisir le chemin accessible par Mosaune, de l’analyser et de le sélectionner directement, sans perdre les réglages en cours.

Arrêter l'ancien serveur avec **Ctrl+C**, copier le dossier **data** pour garder un point de retour, puis relancer **DEMARRER-WINDOWS.cmd** et recharger la page. Le lanceur cible `build/mosaune-0.0.19-windows-amd64.exe`. Les dossiers, progressions, playlists, comptes et configuration fournisseur existants sont conservés ; la base est mise à niveau au démarrage.

La barre de recherche est en haut ; les résultats apparaissent dans un menu contextuel. Les affiches ouvrent une grande fiche, avec suggestions en bas de page. **Ma liste**, **Favoris**, **Playlists** et **Réglages** ont leurs entrées dans le menu. Guide actuel : [ESSAYER-LES-FICHES.md](ESSAYER-LES-FICHES.md). La navigation générale reste décrite dans [ESSAYER-LA-REFONTE.md](ESSAYER-LA-REFONTE.md). Les demandes vidéo peuvent maintenant être envoyées à Radarr/Sonarr après configuration et confirmation. Guide : [CONFIGURER-LES-DEMANDES.md](CONFIGURER-LES-DEMANDES.md). Les boutons musicaux restent préparatoires.

Après une modification des fichiers sur le disque, cliquer sur **Actualiser** pour le dossier concerné. Les fichiers déjà présents n’ont pas besoin d’un nouveau scan pour reprendre leur identification. Ouvrir ensuite une carte d'album ou de série pour afficher ses pistes/épisodes. **Fiche** ouvre les informations du fichier ; la recherche comprend aussi artistes et albums. Les versions MP3 et FLAC restent deux fichiers distincts dans l'album.

Pour les affiches, résumés et acteurs, suivre [CONFIGURER-TMDB.md](CONFIGURER-TMDB.md). Le fournisseur est facultatif. Les correspondances ambiguës se corrigent dans la fiche du média ; elles ne sont pas choisies automatiquement au hasard.

Le contrat Go prévu pour d'autres fournisseurs est expliqué dans [EXTENSIONS-METADONNEES.md](EXTENSIONS-METADONNEES.md). Les paquets installables de cette version concernent uniquement les thèmes déclaratifs : les fournisseurs demandent encore du code compilé dans le serveur.

La musique dispose maintenant d’une **file d’écoute** modifiable, du mélange de la suite, des commandes précédent/suivant et de trois modes de répétition. Les **playlists** sont enregistrées sur le serveur ; la file reste propre au navigateur et se retrouve sans démarrage automatique. Guide pas à pas : [ESSAYER-LA-MUSIQUE.md](ESSAYER-LA-MUSIQUE.md). Limites de cette première version : 200 pistes par liste, 100 playlists. Depuis 0.0.7, la progression musicale est conservée par fichier ; les albums identifiés peuvent être favoris. Le guide musical précédent est historique pour la disposition des écrans.

La base passe au format 11 avec la 0.0.16. Une copie SQLite cohérente `mosaune-before-extensions-…db` est créée dans data avant migration. Les anciens exécutables ne peuvent pas ouvrir ce format ; un retour à une ancienne version demande la copie du dossier data sauvegardée avant la mise à jour.

## Pour commencer sans savoir coder

1. Ouvrir le dossier du projet dans l'Explorateur Windows.
2. Double-cliquer sur **DEMARRER-WINDOWS.cmd**. Une fenêtre reste ouverte pendant le fonctionnement du serveur.
3. Ouvrir **http://127.0.0.1:47831** dans Edge ou Chrome si le navigateur ne s'ouvre pas automatiquement.
4. Au premier lancement, créer l’administrateur avec le code affiché dans la fenêtre du serveur. Puis dans **Réglages → Bibliothèques**, cliquer sur **Ajouter un dossier**. Commencer avec un petit dossier d'essai, contenant quelques morceaux ou vidéos.
5. Dans l'Explorateur Windows, ouvrir ce dossier puis faire **Ctrl+L**, **Ctrl+C** pour copier son adresse. Coller cette adresse dans **Chemin complet du dossier**.
6. Choisir **Musique**, **Films** ou **Séries**, donner éventuellement un nom, puis cliquer sur **Ajouter et analyser**. Les sous-dossiers sont aussi parcourus. Prévoir un dossier distinct par type de média.
7. Ouvrir la carte d'un album, film ou série, puis cliquer sur **Lire** ou sur le bouton ▶ d’une piste/un épisode. Le titre d’un morceau ouvre sa fiche. La musique continue lors de la navigation. Recharger la page : les fichiers du dossier doivent toujours être là.
8. Lire une vidéo compatible, avancer, mettre en pause puis fermer le lecteur. Après rechargement, la rangée **Reprendre le fil** permet de poursuivre. Son menu ••• propose de marquer le média vu/écouté ou non lu, avec position remise à zéro.
9. Après avoir ajouté ou retiré vous-même un fichier dans le dossier, cliquer sur **Actualiser** à côté de ce dossier. Mosaune actualise son catalogue sans supprimer de fichier sur le disque.

Pour arrêter le serveur lancé par ce script, revenir dans sa fenêtre et appuyer sur **Ctrl+C**.

Vérifier **v0.0.19** en bas de la barre latérale sur une grande fenêtre. Les anciens exécutables sont conservés. Le chantier du lecteur et des pistes est reporté ; son découpage est expliqué dans [LECTEUR-A-VENIR.md](LECTEUR-A-VENIR.md).

Le mode **Choisir des fichiers** reste disponible pour un essai ponctuel : ses entrées disparaissent au rechargement et leur progression n'est pas conservée. Les dossiers ajoutés dans **Vos dossiers**, eux, sont mémorisés par le serveur. Dans les deux cas, les originaux ne sont jamais modifiés.

## Ce qui fonctionne dans ce prototype

- Accueil et navigation Films / Séries / Musique, interface adaptée aux petites fenêtres.
- Serveur Go qui fournit l'interface et annonce les modules actifs.
- Catalogue SQLite conservé dans `data/mosaune.db` avec le lanceur Windows.
- Ajout de dossiers par type, analyse récursive des extensions reconnues, rafraîchissement manuel.
- Recherche dans le catalogue et les fichiers temporaires ; lecture depuis le serveur avec déplacement dans le média.
- Progression films/séries/musique enregistrée toutes les cinq secondes pendant la lecture, à la pause, au déplacement et à la fermeture du lecteur. Reprise sur lecture explicite ; commandes vu/écouté et non lu. Fin naturelle marquée terminée.
- Si le dossier est inaccessible ou l'analyse échoue, le catalogue précédent reste intact. Les fichiers absents après une analyse réussie sont masqués ; une modification de taille/date réinitialise la progression.
- Lecture persistante pendant la navigation interne, passage au fichier suivant du même album ou de la même série ; les fichiers temporaires gardent leur file par type.
- Tags musicaux et pochettes intégrées ; classement par album et artiste d'album, titres et numéros de pistes disponibles.
- Séries regroupées, saisons et épisodes issus des noms usuels S01E02 ou 1x02. Présentation en cartes et fiches détaillées.
- Fournisseur TMDB facultatif, mise en cache locale des textes/images, identification prudente et correction manuelle de correspondance.
- Demandes Radarr/Sonarr avec configuration guidée, confirmation, déduplication, suivi persistant, contrôle de disponibilité locale et notification dans l’interface.
- Démarrage possible avec vidéo seulement ou musique seulement.

Les deux modules sont encore des capacités intégrées au prototype, activables au démarrage. Ce n'est pas le futur mécanisme de plugins installables, ni une preuve de son isolation. L'apparence est une première proposition modifiable.

## Ce qui viendra ensuite

Installation NAS, recommandations personnalisées, connecteur musical, abonnements, webhooks d’import, comptes, apps Android et TV, hors-ligne mobile et extensions installables. Les commandes audio/vidéo de base restent celles du navigateur ; les commandes de file et de répétition sont fournies par Mosaune.

Cette première analyse est manuelle, parcourt tout le dossier et doit se terminer avant de fermer la page. Les arrivées liées aux demandes bénéficient désormais d’une analyse périodique des dossiers associés ; pas encore de surveillance générale du disque ni de tâche de scan reprenable, de correction des dossiers sources ni de pagination des grands catalogues. Les liens symboliques sont ignorés. Les noms viennent des fichiers, de leurs tags ou du fournisseur sélectionné. La progression est unique pour cette installation, sans comptes ni résolution des conflits entre onglets/appareils. Une fermeture brutale peut perdre les dernières secondes non envoyées. Les épisodes multiples gardent pour l'instant leur identification locale ; la correction d'une correspondance porte sur un fichier à la fois.

Ne pas placer la base SQLite sur un partage réseau. Pour sauvegarder le catalogue, arrêter Mosaune puis copier **tout le dossier data** ; les médias restent à sauvegarder séparément. En dehors du lanceur, l'option `-data-dir` choisit ce dossier. Sans option, il est créé à côté de l'exécutable. Les quatre médias fournis ont été utilisés en lecture seule avec un catalogue de test séparé. Aucun accès au NAS effectué.

Le serveur écoute uniquement sur **127.0.0.1**, une adresse accessible depuis cet ordinateur. Il n'a pas encore d'authentification et ne doit pas être publié sur Internet. Il peut lire un partage NAS si l’utilisateur l’ajoute explicitement dans ses dossiers. Aucun accès au NAS personnel n’a été effectué pendant le développement.

## La notion du jour

La **base de données** est le carnet de Mosaune : elle retient où se trouvent les médias et où vous en êtes dans une vidéo. SQLite range ce carnet dans un fichier, sans service supplémentaire à installer. Vos films restent dans leurs dossiers. Node et les outils de développement ne sont pas nécessaires pour lancer l'exécutable Go.

## Retours utiles pour la prochaine étape

- L'organisation Films / Séries / Musique est-elle claire ?
- Le style visuel te plaît-il, et le texte est-il suffisamment lisible ?
- La musique continue-t-elle quand tu changes d'espace ?
- Quel type de fichier fonctionne ou échoue, et dans quel navigateur ? Inutile d'envoyer le média lui-même.

## Pour reconstruire le projet plus tard

Prérequis : Node 24+, npm, et Go 1.26 ou supérieur. Versions exactes utilisées et résultats dans **VALIDATION.md**. Dépendances verrouillées dans `package-lock.json`, `go.mod` et `go.sum`. Le pilote [SQLite en Go](https://pkg.go.dev/modernc.org/sqlite) permet la construction avec `CGO_ENABLED=0`.

Depuis le dossier du projet, sous Windows ou Linux :

```sh
npm ci --ignore-scripts
npm run check
node --test web/src/progress.test.mjs web/src/collection.test.mjs web/src/music.test.mjs
npm run build
go test ./...
```

Sous Windows :

```powershell
New-Item -ItemType Directory -Force build
go build -trimpath -o build/mosaune-0.0.16-windows-amd64.exe .
./build/mosaune-0.0.16-windows-amd64.exe -data-dir ./data
```

Sous Linux x86-64 :

```sh
mkdir -p build
go build -trimpath -o build/mosaune-linux-amd64 .
./build/mosaune-linux-amd64 -data-dir ./data
```

Exemple pour n'activer que la musique : ajouter `-modules music` à la commande de démarrage. `-modules video` active films/séries. `-port 47832` change le port local. Ces options ne rendent pas le serveur accessible sur le réseau.

Les exécutables ARM64 et x86-64 produits sur Windows sont des vérifications de compilation. Ils ne prouvent pas encore un fonctionnement validé sur le Pi ou Omarchy. Aucun environnement de développement n'est installé globalement par cet atelier : les outils temporaires résident dans le dossier `work` voisin.

## Reprendre sur un autre poste

Conserver les sources, les guides et les fichiers de verrouillage. Les dossiers `node_modules`, `web/dist` et `build` se reconstruisent et sont exclus de Git. `data` contient l'état personnel, également exclu de Git ; ne pas le supprimer en pensant qu'il se reconstruit. Les chemins Windows du catalogue ne fonctionneront pas tels quels sous Linux : la migration des sources reste à préparer. Le dépôt distant n'est pas encore créé. La feuille de route maîtresse reste dans `outputs/ROADMAP-MEDIA.md`.

Avant une publication, choisir la licence du projet et ajouter les notices des dépendances. Aucun dépôt n'a été publié pendant cet atelier.
