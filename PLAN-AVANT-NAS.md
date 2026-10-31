## Étape 0.0.47 — distribution des familles par archive

- [x] Import local de programmes et écran dans une même archive ; installation et mise à jour coordonnées, prévisualisation, conservation des choix et des permissions.
- [x] Exemples complets et outil de préparation pour les contributeurs.
- [ ] Intégrer les familles aux dépôts distants ; ajouter restauration globale et compatibilité entre familles, puis poursuivre les services métier.

## Priorité ajustée — extensions regroupées, 30 octobre 2026

Installer quelques grandes extensions (Films, Séries, Musique), avec leurs composants internes et réglages avancés, plutôt qu’une extension visible par petite fonction. Les composants restent remplaçables ou cumulables selon leur contrat. Fournisseurs, connecteurs et thèmes ayant un usage propre restent séparés ; les dépendances techniques partagées sont gérées automatiquement.

**Étape 0.0.46 livrée :** manifestes de composition, cycle commun des composants extraits et des écrans, gestionnaire regroupé. **Suite :** distribution et mise à jour de familles complètes, puis extraction des services encore intégrés. Les contrats et extractions existants sont conservés. [Décision et checklist détaillées](ARCHITECTURE-EXTENSIONS-GROUPEES.md).

## Étape 0.0.45

- [x] Distribuer TMDB comme un paquet signé installé une fois, sans exécution directe d’un module embarqué de remplacement.
- [x] Retrait persistant, activation refusée sans réinstallation et migration respectant les choix antérieurs.
- [ ] Extraire les services hôtes TMDB (transport, configuration, identifiants Sonarr), puis MusicBrainz et connecteurs.

## Étape 0.0.44

- [x] Retirer le moteur TMDB historique et conserver une seule implémentation des recherches/fiches/catalogues dans le module.
- [x] Migrer le choix historique sans réactiver un fournisseur désactivé ni un accès refusé ; conserver les paquets installés.
- [x] Pause/retrait persistants, réponses tardives refusées et mode de secours sans substitution.
- [ ] Achever la distribution indépendante et les services hôtes TMDB, puis les autres fournisseurs et connecteurs.

## Étape 0.0.43 — catalogue vidéo TMDB

- [x] Exécuter découverte, suggestions, sélections, profils et choix d’affiches dans le paquet signé TMDB ; contrôler services et résultats, refuser le retour silencieux pour un ancien paquet choisi.
- [ ] Retirer le mode intégré historique et généraliser fournisseurs, transport d’images et identités externes avant de déclarer TMDB entièrement indépendant.

## Étape 0.0.42 — écrans médias remplaçables

- [x] Distribuer les six écrans de collection dans des paquets déclaratifs ; remplacer leur composition sans modifier le noyau.
- [x] Refuser les conflits de rubrique, conserver les droits et le retrait au redémarrage, fournir un remplacement installable.
- [ ] Raccorder ces interfaces aux familles complètes et poursuivre l’extraction des services, profils et formulaires.

## Étape 0.0.41 — pages et menus déclaratifs

- [x] C-UI-01 — Un paquet ajoute des pages, des menus et des rubriques de paramètres sans modifier le routeur. Import/dépôts, droits, désactivation, mise à jour et retrait ; exemple Salon personnel.
- [ ] C-UI-02 — Migrer les écrans métiers restants, puis formulaires persistants, traductions et compositions avancées.

[Essayer les pages](ESSAYER-LES-PAGES-041.md). Cette étape ne termine pas les familles médias complètes.

## Étape 0.0.40 — ensembles de composants

Préparer, désactiver et retirer en lot les composants déjà extraits de Films, Séries et Musique : aperçu, transaction unique, révision/empreinte, conservation des remplaçants et des composants partagés. Recettes déclaratives embarquées, sans nouveau processus au repos. Cela ne constitue pas encore trois familles médias complètes : écrans et services intégrés restent à extraire. [Guide](ESSAYER-LES-ENSEMBLES-040.md) · [Checklist pondérée : environ 60 %](AVANCEMENT-EXTENSIONS.md).

- [x] C-MEDIA-02d — Gestion groupée transactionnelle des composants existants, prévisualisation et protection des composants communs. C-MEDIA-02 reste ouvert pour les familles complètes.

## Critère de sortie confirmé — 21 octobre 2026

Avant le NAS, le noyau conserve les services communs (comptes, permissions, chargement, transactions et composition UI) ; toutes les fonctions médias doivent venir d’extensions réellement installables et retirables. Une séparation de fichiers source ou un interrupteur ne suffit pas. Les lecteurs personnalisés viennent après ce découpage.

- [x] C-MEDIA-01 — 0.0.29 : reconnaissance locale Films/Séries/Musique exécutée dans trois modules WASI indépendants ; scan réel raccordé, lots, validation et conservation du catalogue.
- [x] C-MEDIA-02a — 0.0.30 : cycle persistant des trois reconnaisseurs officiels (installer, activer, désactiver, retirer), annulation effective, migration, cartes UI et catalogue conservé. [Guide](ESSAYER-LES-EXTENSIONS-030.md). Import externe et familles médias complètes non couverts.
- [x] C-MEDIA-02b — 0.0.31 : import local de reconnaisseurs externes, manifeste commun, sélection explicite, mise à jour/restauration, exemple contributeur et tests de scan réel. [Guide](ESSAYER-LES-EXTENSIONS-031.md). Les dépôts pour ce format et les rôles média complets restent ouverts.
- [x] C-MEDIA-02c — 0.0.32 : reconnaisseurs externes distribués par dépôts GitHub, aperçu et confirmation, versions installées/mises à jour, exemple public et restauration hors ligne après retrait du dépôt. [Guide](ESSAYER-LES-EXTENSIONS-032.md).
- [ ] C-MEDIA-02 — Installer, choisir, désactiver et retirer les paquets média et leurs capacités, avec dépendances et conservation des données communes. Les programmes de reconnaissance livrés ne remplissent pas encore ce critère.
- [x] C-MEDIA-03a — 0.0.33 : tags audio et pochettes embarquées dans un module WASI avec accès borné au seul fichier ouvert, cycle de vie persistant, conservation des données et tests MP3/FLAC. [Guide](ESSAYER-LES-EXTENSIONS-033.md). Import de lecteurs tiers à ajouter.
- [x] C-MEDIA-03b — 0.0.34 : lecteurs de tags tiers importables et distribuables par dépôt, sélection explicite, mise à jour/restauration et stockage partagé avec les reconnaisseurs. [Guide](ESSAYER-LES-EXTENSIONS-034.md).
- [x] C-MEDIA-03c — 0.0.35 : regroupements et ordre des catalogues fournis par trois modules WASI installables/remplaçables ; interface générique, exemple externe, import local et dépôts communs, conservation des fichiers individuels lorsque le module manque. [Guide](ESSAYER-LES-EXTENSIONS-035.md). Associations distantes, pages, saisons détaillées et autres comportements à poursuivre.
- [x] C-MEDIA-03d — 0.0.36 : saisons et liste d’épisodes projetées par une extension remplaçable, contrat de liste déclarative, exemple externe, permissions et conservation des fichiers ; haut des fiches et autres médias à poursuivre. [Guide](ESSAYER-LES-EXTENSIONS-036.md).
- [x] C-MEDIA-03e — 0.0.37 : rubriques descriptives des fiches Films/Séries (résumé, informations, distribution, bandes-annonces) composées par une extension remplaçable. Contrat public, rendu thémable et exemple indépendant. En-têtes/actions/suggestions restent à extraire. [Guide](ESSAYER-LES-EXTENSIONS-037.md).
- [x] C-MEDIA-03f — 0.0.38 : textes des en-têtes Films/Séries/Musique par extension remplaçable, informations ordonnées et thémables, exemple externe et contrôles album/morceau. Affiches, artistes et commandes restent intégrés. [Guide](ESSAYER-LES-EXTENSIONS-038.md).
- [x] C-MEDIA-03g — 0.0.39 : lot commun de comportements des fiches (commandes, artistes, couvertures, fichiers/pistes, rapprochement et suggestions) dans une extension remplaçable ; exemple externe, import/dépôts et conservation. Les adaptateurs exécutant les actions et les fournisseurs restent à extraire. [Guide](ESSAYER-LES-EXTENSIONS-039.md).
- [ ] C-MEDIA-03 — Extraire les règles des catalogues spécialisés et leurs comportements ; supprimer les replis métier restants dans le noyau.
- [ ] C-MEDIA-04 — Contributions UI des modules (navigation, pages, réglages, recherche), thèmes et traductions, sans route centrale par type de média.
- [ ] C-MEDIA-05 — Extraire les fournisseurs/connecteurs restants et vérifier le démarrage sans extension média, puis l’ajout et le retrait de chacune.

Guide : [Extraction 0.0.29](ESSAYER-LES-EXTENSIONS-029.md). Les fonctionnalités ci-dessus ne sont pas toutes livrées.

État au 20 octobre 2026 : catalogue officiel public, intégration et dédoublonnage 0.0.22 livrés. Sources du serveur publiées sur https://github.com/car-lito/mosaune ; vérifications Windows/Linux et construction ARM64 automatisées. 0.0.23 : HTTPS direct et arrêt SIGTERM validés, images Docker amd64/arm64 publiées et téléchargements anonymes vérifiés. Catalogue CasaOS 0.4.15 préparé ; import réel encore à tester. 0.0.24 : désactivation réelle de TMDB livrée et images publiées après tests. 0.0.25 : présentation extensible des résultats de recherche livrée et images publiées après tests. 0.0.26 : dépendances persistantes et premier raccordement du paquet TMDB livrés ; images publiées après tests. 0.0.27 : réglages génériques par propriétaire et migration TMDB livrés ; images publiées après tests. 0.0.28 : autorisations persistantes et premier contrôle TMDB livrés ; images publiées après tests. Aucun déploiement NAS.

# Mosaune — comptes, personnalisation et extensions avant le NAS

## Priorités confirmées le 19 octobre 2026

Le premier déploiement NAS attend un système d’extensions fonctionnel pour les familles prévues et des lecteurs personnalisés pour films, séries et musique. Le pilote TMDB seul ne remplit pas ce critère. Ordre de travail : terminer et généraliser les extensions, puis livrer les lecteurs, puis tester une installation neuve sur CasaOS. La publication du code et des images de test peut continuer entre-temps.

- [ ] Généraliser le cycle complet des paquets : dépendances, réglages et stockage isolés, permissions, installation, mise à jour, désactivation et désinstallation ; arrêt réel des tâches et conservation des données communes.
- [ ] Extraire les modules Films, Séries, Musique, les fournisseurs et les connecteurs en paquets réellement indépendants ; vérifier qu’un module peut fonctionner sans les autres.
- [ ] Rendre les contributions d’interface, menus, réglages et langues utilisables par des extensions externes, avec les contrats de thèmes existants ; documenter et démontrer un paquet créé hors du noyau.
- [ ] Valider les dépôts officiels/communautaires, les dépendances manquantes et les pannes de paquets sans rendre Mosaune inutilisable. Les travaux détaillés C et D ci-dessous restent la checklist de référence.
- [ ] Lecteur vidéo Mosaune pour films et séries : commandes propres, navigation dans la lecture, reprise, plein écran, sélection des pistes audio/sous-titres et gestion explicite des formats non pris en charge. Séparer interface, session de lecture et moteur de décodage ; réutiliser des moteurs éprouvés sans réécrire les codecs. Prévoir l’adaptation des flux côté serveur lorsqu’elle est nécessaire, avec limites adaptées au Pi.
- [ ] Lecteur musique Mosaune : commandes propres, file, précédent/suivant, lecture aléatoire/répétition, déplacement dans le morceau, volume et continuité pendant la navigation ; intégration aux thèmes et modules.
- [ ] Vérifier ces parcours avant le premier déploiement NAS. Les applications Android/TV et la parité complète Plex/Plexamp restent des jalons distincts.

Installation NAS choisie : repartir de zéro, créer le premier administrateur et installer/configurer les extensions comme un nouvel utilisateur. Aucun transfert du catalogue Windows n’est requis. CasaOS expose les médias par des volumes : Mosaune utilise les chemins à l’intérieur du conteneur. Il faudra seulement vérifier ces montages et leurs permissions, notamment les correspondances avec les chemins des conteneurs Radarr/Sonarr.

Précisions lecteurs : [cahier des charges du 19 octobre](LECTEURS-ET-EXTENSIONS.md). Lecteur vidéo livré automatiquement avec Films ou Séries, une seule dépendance partagée ; lecteur audio livré avec Musique ; tous deux remplaçables et thémables. Références Plex pour bandeau vidéo et plein écran, Spotify pour bandeau musical et vue immersive ouverte par clic sur la pochette avec indication au survol/focus.

### Découpage des extensions en cours — 19 octobre

- [x] C-DEPS-01 — Résolution de capacités et associations explicites intégrées au moteur ; dépendance partagée, protection des consommateurs actifs et arrêt en cascade explicite. [Contrat et limites actuelles](extension-kit/dependencies/README.md).
- [ ] C-DEPS-02 — Registre persistant commun et choix de fournisseurs exposés dans l’interface ; intégration aux installateurs et dépendances de versions.
- [x] C-DEPS-02a — 0.0.26 : registre SQLite et révisions, choix conservés, plans d’installation/mise à jour/retrait, restauration d’un moteur et états de panne ; premier raccordement transactionnel du paquet TMDB et section Dépendances et démarrage. [Contrat du registre](extension-kit/registry/README.md).
- [x] C-CONFIG-01 — 0.0.27 : stockage générique des réglages par propriétaire, isolation API des secrets, révisions et raccordement transactionnel TMDB ; migration au schéma 16 avec sauvegarde. [Contrat](extension-kit/configuration/STORAGE.md).
- [ ] C-CONFIG-02 — Migrations privées explicites et aperçu des changements ; chiffrement des secrets au repos avec gestion de clé, sauvegarde et récupération ; généralisation des services et permissions aux autres familles.
- [x] C-ACCESS-01 — 0.0.28 : stockage des demandes et accords par propriétaire/opération, révisions, conservation de l’intersection lors des mises à jour, refus des nouveaux accès implicites. [Contrat](extension-kit/authorization/README.md).
- [x] C-ACCESS-02 — 0.0.28 : premier adaptateur TMDB, révocation commune aux deux moteurs, annulation et refus des résultats tardifs, conservation des données, formulaire administrateur et redémarrage testés.
- [ ] C-SERVICES-01 — Autorisations durables par extension et opération de service ; une déclaration du paquet ne vaut pas autorisation. Les accès supplémentaires d’une mise à jour doivent apparaître dans son aperçu et ne pas être accordés silencieusement.
- [ ] C-SERVICES-02 — Révocation effective : bloquer les nouveaux appels, annuler ceux en cours et empêcher les réponses tardives de modifier les données ; conserver réglages et métadonnées communes. Tests d’isolation entre deux extensions, y compris après redémarrage.
- [ ] C-SERVICES-03 — Présenter les accès dans la fiche de l’extension et raccorder les adaptateurs hôtes des autres familles ; vérifier les droits de l’utilisateur et les limites de chaque opération, sans donner les clés ou un accès libre au disque au code invité.
- [ ] C-DEPS-02b — Généraliser les familles, leurs services et permissions, la sélection interactive entre fournisseurs et les politiques de compatibilité des paquets ; le pilote TMDB refuse encore les dépendances externes.
- [ ] C-DEPS-03 — Dépendances officielles installées une fois, aperçu des changements, mises à jour et restauration cohérentes.

Les 0.0.26 à 0.0.28 raccordent le premier installateur, ses réglages et les autorisations TMDB à ce socle ; cela ne termine pas le système d’extensions et n’extrait pas encore les modules média. Guide : [Essayer cette étape](ESSAYER-LES-EXTENSIONS-028.md).

## Désactivation réelle des fournisseurs — pilote TMDB 0.0.24

- [x] C-LIFECYCLE-01 — Distinguer l’activation du fournisseur du choix de son moteur (intégré ou extension). « Désactivé » doit arrêter le fournisseur, sans retour silencieux au moteur intégré.
- [x] Bloquer tous ses nouveaux accès réseau : recherche, découverte, fiches, images, enrichissement automatique et tâches de fond ; annuler les traitements en cours lorsque possible et ignorer leurs réponses tardives.
- [x] Conserver les métadonnées communes déjà acquises, les affiches conservées localement, les corrections et les préférences. Ne pas affecter les autres fournisseurs actifs, par exemple MusicBrainz lorsque TMDB est arrêté.
- [x] Expliquer l’état dans l’interface et tester l’absence de nouveaux appels, y compris après redémarrage et lors d’une recherche déjà en cours.

Livré pour TMDB en 0.0.24 : état indépendant et persistant, annulation des opérations natives/Wasm, protection contre les réponses tardives et conservation du catalogue. Contrat hôte facultatif réutilisable par les futurs fournisseurs ; MusicBrainz reste actif et n’a pas encore son propre interrupteur. Guide : [ESSAYER-TMDB-024.md](ESSAYER-TMDB-024.md).

À la migration, TMDB reste actif comme il l’était effectivement auparavant : l’ancien interrupteur sélectionnait seulement le moteur. Désactivez-le une première fois avec le nouvel interrupteur pour couper ses accès. La désinstallation du paquet exécutable reste distincte de la suppression du fournisseur intégré, de sa clé et de ses données privées.

## Présentation des résultats — 0.0.25

- [x] D-SEARCH-01 — Aperçus adaptés au contenu et type affiché sous le titre ; renderer commun thémable.
- [x] D-SEARCH-02 — Objet de présentation versionné, labels/mots-clés libres, proportions et cadrage validés ; transmission depuis les contrats de métadonnées jusqu’au catalogue et à la recherche.
- [ ] D-SEARCH-03 — Nouvelles sources de recherche pour personnes, saisons et modules média supplémentaires ; ce contrat n’installe pas ces sources à lui seul.

Guide : [Présentation v1](extension-kit/presentation/v1/README.md).

## Avancement 0.0.20 — catalogue et dépôts publics

- [x] C-STORE-01 — Index versionné séparé des formats de paquets, cartes filtrables, choix de version et installation après aperçu.
- [x] C-STORE-02 — Dépôts GitHub publics ajoutables ; actualisation explicite, cache du dernier succès, retrait indépendant de la désinstallation.
- [x] C-STORE-03 — Téléchargement borné, intégrité et concordance du manifeste, politique de signature des fournisseurs conservée, droits admin.
- [x] C-STORE-04 — Catalogue intégré de démonstration et export d’un dépôt complet prêt à publier, guides utilisateur/auteur.
- [x] C-STORE-05 — Publication du vrai dépôt officiel et ajout de son adresse au catalogue par défaut (0.0.21).
- [ ] C-TRUST-01 — Clés/éditeurs communautaires exécutables et budgets globaux avant ouverture au code non fiable.

Les thèmes tiers sont acceptés ; les fournisseurs exécutables restent les paquets TMDB signés reconnus. Depuis 0.0.32, les dépôts acceptent aussi les reconnaisseurs externes sans services ni dépendances, avec confirmation explicite et éditeur non authentifié. Aucun dépôt privé, autre hébergeur ou mise à jour automatique. Voir ESSAYER-LE-CATALOGUE-020.md.

## Avancement 0.0.19 — fournisseur installable signé

- [x] C-PACKAGE-01 — Archive indépendante, signature/empreinte/compatibilité et dépendances de l’adaptateur vérifiées avant exécution.
- [x] C-PACKAGE-02 — Installation et mise à jour de TMDB sans remplacement du serveur ; contrôle préalable, transaction atomique et une version de retour.
- [x] C-PACKAGE-03 — Restauration, désactivation, désinstallation, reprise après redémarrage et mode secours ; conservation des données communes.
- [x] C-PACKAGE-04 — Deux paquets signés prêts à tester, script de publication et parcours UI validé.
- [ ] C-PACKAGE-05 — Autres fournisseurs et adaptateurs, stockage isolé générique, migrations par paquet et dépendances générales.
- [ ] C-TRUST-01 — Gestion des clés/éditeurs, dépôts communautaires et budgets de compilation/processus avant ouverture au code non fiable.

Le cycle est réel pour le pilote TMDB signé. Ce n’est pas encore un magasin universel de fournisseurs ni l’extraction des gros modules média. Guide : ESSAYER-LES-FOURNISSEURS-019.md.


## Avancement 0.0.18 — 15 octobre 2026

- [x] C-CONFIG-01 — Schéma de formulaire versionné, rendu thémable, validation serveur et sémantique des secrets ; premier binding TMDB réutilisant sa connexion.
- [x] C-SERVICES-01 — Services/opérations déclarés par le module, intersection avec droits hôte de chaque invocation, description visible des accès.
- [x] UI-018 — Interrupteurs éteints contrastés et barres natives masquées, défilement conservé.
- [ ] C-CONFIG-02 — Stockage isolé générique et cycle de vie/migrations des réglages de paquets installables.
- [ ] C-SERVICES-02 — Politiques réseau/services déclarées dans les manifestes installables, validation de dépendances et mise à jour de fournisseurs autonomes.

Ces sous-étapes ne ferment pas encore C-RUNTIME-05b2. Voir ESSAYER-LA-018.md et extension-kit/configuration/README.md.


## Avancement 0.0.17 — 15 octobre 2026

- [x] Paramètres : défilement indépendant de la navigation et du contenu, adaptation mobile.
- [x] Extensions : cartes, recherche, activation des modules réellement activables, grandes fiches et onglets Sur le serveur / À télécharger.
- [x] C-RUNTIME-05b1 — Capacité facultative de fiches vidéo riches, pilote TMDB, validation et catalogue commun.
- [ ] C-RUNTIME-05b2 — Politiques et réglages déclarés par paquet, stockage dédié et installation/mise à jour de fournisseurs indépendants.

Les cases générales ci-dessous restent ouvertes lorsqu’elles couvrent davantage de fonctions. Les six fonctions intégrées sont présentées en cartes, mais ne sont pas encore des paquets désinstallables. Guide : ESSAYER-LES-EXTENSIONS-017.md.


Cadrage du 13 octobre 2026, actualisé le 14 octobre. Fonctionnalités demandées consignées ; les cases ouvertes ne désignent pas des fonctions livrées. Livraisons : 0.0.12 (affiches films/séries et paramètres), 0.0.13 (comptes et permissions), 0.0.14 (premiers paquets de thèmes et droits d’usage). Extensions exécutables, boutique et accès réseau restent à réaliser.

## Ordre proposé et critères de livraison

### A. Affiches et paramètres structurés

Livraison 0.0.12 : alternatives TMDB classées avec priorité au français, import pour films/séries même non identifiés, choix durable et réinitialisation. Navigation Bibliothèques/Métadonnées/Téléchargements/Serveur livrée. Le tri selon une langue de compte, les rubriques personnelles et le chargement de rubriques d’extensions restent dépendants des étapes suivantes. Guide : [ESSAYER-LES-AFFICHES.md](ESSAYER-LES-AFFICHES.md).

- [ ] Choisir une affiche parmi les alternatives du fournisseur, en privilégiant la langue du compte et en conservant un repli.
- [x] Importer une image locale, prévisualiser, enregistrer ou annuler ; revenir à l’affiche automatique.
- [x] Stocker la sélection comme correction explicite de l’œuvre ; priorité sur les actualisations automatiques, cohérence entre accueil et fiche.
- [x] Valider type réel, taille et dimensions des images, normaliser leur contenu. Conserver les images choisies hors du cache susceptible d’éviction.
- [ ] Préparer le même contrat pour séries, saisons et albums, sans promettre toutes ces interfaces dans la première livraison.
- [ ] Paramètres avec navigation : Mon compte, Apparence, Langue ; administration : Utilisateurs, Bibliothèques, Métadonnées, Téléchargements, Extensions, Serveur et sauvegardes.
- [ ] Préparer les emplacements où une extension autorisée peut ajouter une section de paramètres.

### B. Comptes et permissions

Livraison 0.0.13 : comptes locaux, invitations à usage unique valables sept jours, champ de mot de passe facultatif côté admin, récupération et révocation, droits voir/lire/modifier/demander par bibliothèque et par type de média, données privées, thèmes clair/sombre/système. La 0.0.14 ajoute les droits d’usage des paquets de thèmes. Le français reste la seule langue. Le transport HTTPS et les capacités des futurs paquets exécutables sont encore à traiter. Guide : ESSAYER-LES-COMPTES.md.

- [x] Assistant de création du premier administrateur ; verrou de configuration initiale pour qu’un visiteur du réseau ne puisse pas s’approprier le serveur.
- [ ] Mots de passe hachés avec sel individuel et Argon2id, réglages mesurés sur Pi. Ils ne sont ni stockés en clair ni chiffrés de façon réversible. Pas de promesse de sécurité absolue.
- [ ] Sessions révocables, déconnexion, modification et récupération administrative du mot de passe, limitation des essais, protection des mutations et transport HTTPS pour les accès réseau.
- [ ] Rôles simples par défaut, avec permissions détaillées par utilisateur et par bibliothèque : consulter, lire, demander, modifier métadonnées/affiches, gérer playlists partagées, administrer dossiers/comptes/extensions. Refus par défaut pour une nouvelle permission sensible.
- [x] Vérification de chaque accès côté serveur, y compris recherche, images, lecture directe, tâches et notifications. Masquer un bouton ne suffit pas.
- [x] Séparer progressions, favoris, listes, playlists privées, préférences et historique des comptes. Attribuer les données personnelles existantes au premier administrateur lors de la migration.
- [x] Préserver le catalogue et les corrections d’affiches partagés ; les utilisateurs ayant le droit de modifier une affiche changent la fiche commune.
- [ ] Préférences langue, thème et apparence par compte ; droit d’utiliser une extension distinct du droit de l’installer/configurer.
- [x] Tests avec deux comptes : impossibilité d’accéder à une bibliothèque interdite par URL directe ; impossibilité de modifier les données privées d’un autre compte.

### C. Extensions réellement installables

**Pilote 0.0.16, 15 octobre :** service hôte nommé avec droits par invocation et premier module TMDB branché aux recherches du catalogue. Interrupteur administrateur facultatif, migration format 11, clés et cache côté hôte, corrections conservées. Détails/images toujours natifs ; boutiques et paquets fournisseurs exécutables toujours à faire. Guide : ESSAYER-TMDB-EXTENSION.md.

- [x] C-RUNTIME-05a — Premier service hôte contrôlé et recherche TMDB réelle, contrats partagés, cycle activation/désactivation et droits administrateur.
- [ ] C-RUNTIME-05b — Fiches riches, paramètres et politiques réseau déclarés par paquet, stockage dédié, installation/mise à jour de fournisseurs autonomes.

**Prototype du moteur, 14 octobre :** module séparé extension-kit, protocole JSON versionné, capacités indépendantes, SDK facultatif et exemple WASI prêt à lancer. Tests Windows d’exécution, erreurs, quotas, annulation et isolement des fichiers/environnement ; compilation Linux amd64/arm64. Pas de nouvelle version serveur ni de fournisseur externe installable dans l’interface à ce stade. Mesures et limites : [extension-kit/README.md](extension-kit/README.md).

- [x] C-RUNTIME-01 — Prototype réel séparant transport, modèles de capacités, SDK et exécution ; exemple compilé hors du moteur.
- [x] C-RUNTIME-02 — Une invocation à la fois, messages bornés, mémoire linéaire limitée, délais, annulation et cycle chargement/activation/retrait vérifiés sous Windows.
- [x] C-RUNTIME-03 — Mesures reproductibles sur Windows, exécutables sans CGO pour Windows et Linux amd64/arm64 ; démonstration accessible sans programmation.
- [ ] C-RUNTIME-04 — Exécution et consommation sur Pi, budgets globaux du processus et compilation de paquets non fiables.
- [ ] C-RUNTIME-05 — Services hôte pour réseau/secrets/stockage, pont vers le catalogue puis installation et mise à jour de fournisseurs réels.

Principe confirmé par l’utilisateur : des interfaces définies et modulaires, sans transformer le contrat initial en liste exhaustive de tout ce qu’une extension pourra faire. Nouvelles capacités versionnées et facultatives ; modèles de médias indépendants du transport. Les futures contributions UI réutilisent les composants et valeurs des thèmes, sans imposer à tous les créateurs une disposition figée.

Le contrat Go actuel est une préparation, pas une boutique fonctionnelle. Cette étape doit permettre d’ajouter un paquet sans modifier ni recompiler le noyau.

**Première livraison 0.0.14 :** paquets ZIP de thèmes réellement installables sans recompilation, manifeste/compatibilité/capacité v1, aperçu, activation, désactivation, mise à jour ascendante, retrait, permissions d’usage et préférence par compte. Exemple Ambre et guide auteur. Pas de code exécutable, de dépendances entre paquets ni de catalogue GitHub dans cette tranche ; les critères généraux ci-dessous restent ouverts lorsqu’ils couvrent ces familles supplémentaires.

- [x] C-THEME-01 — Installation atomique d’un paquet déclaratif externe, limites et validation, empreinte, refus des capacités inconnues.
- [x] C-THEME-02 — Cycle activation/désactivation/mise à jour/retrait, droits administrateur et d’usage, références des données, conservation au redémarrage.
- [x] C-THEME-03 — Exemple hors du noyau, contrat documenté, tests de paquets invalides et de permissions ; mode de secours sans thèmes externes.

| Famille | Exemples | Rôle |
|---|---|---|
| Modules média | Films, Séries, Musique, futurs Livres | Règles et écrans propres au média, schéma logique du catalogue |
| Fournisseurs | TMDB, TVDB, MusicBrainz | Métadonnées normalisées, recherches et images |
| Connecteurs et fonctions | Radarr, Sonarr, autre traitement de demandes | Actions et événements pour les médias compatibles |
| Interface et localisation | Thèmes, dispositions, menus, langues | Personnalisation par points d’extension documentés |

- [ ] Manifeste versionné : identité, éditeur, version, compatibilité noyau/clients, dépendances, capacités, permissions, réglages, migrations et ressources.
- [ ] Dépendances exprimées par capacités lorsque pertinent : un fournisseur peut servir Films **ou** Séries, sans imposer les deux ; un traitement de demandes n’est pas forcément Radarr.
- [ ] Installation, activation, désactivation, mise à jour et désinstallation avec états explicites ; dépendances manquantes et incompatibilités expliquées.
- [ ] Séparer droits utilisateur et autorisations du paquet : bibliothèques autorisées, réseau, secrets dédiés, ressources et stockage. Aucun accès SQL général ni récupération de tous les secrets.
- [ ] Choisir le moteur d’exécution après un prototype d’isolation et de consommation sur ARM64 et Windows. Un processus séparé ne suffit pas à garantir le confinement.
- [ ] Limiter CPU, mémoire, délais et appels ; arrêter les tâches lors de la désactivation ; permettre un démarrage de secours sans extensions communautaires.
- [ ] Extraire progressivement les fonctions existantes en véritables paquets. Ne pas appeler « installable » une simple option du binaire.
- [ ] Installation/activation à chaud là où le contrat le permet ; pour une opération nécessitant recharge de l’interface, arrêt d’une lecture ou redémarrage, l’annoncer explicitement. Pas de promesse de toute migration sans interruption.
- [ ] Livrer un exemple documenté créé hors du code du noyau et un test de contrat, afin de prouver qu’un auteur tiers peut étendre Mosaune.
- [ ] Si aucun connecteur compatible n’est installé, expliquer au clic sur Demander ce qui manque. L’admin accède à l’installation ; un utilisateur ordinaire voit une explication adaptée. Distinguer absent, désactivé, non configuré, indisponible et permission refusée. Ne pas enregistrer une fausse demande réussie.

### D. Interface extensible et catalogue de paquets

**Refonte 0.0.15 :** thème commun au client web et aux paramètres, 996 réglages de composants, palette/polices/arrondis réutilisables et exceptions explicites par rubrique. L’utilisateur autorise ces exceptions : elles sont facultatives, jamais un retour implicite au vert. Même contrat pour le thème natif et les paquets ; audit automatique des styles. Les dispositions, menus et traductions restent à réaliser.

- [x] D-THEME-01 — Contrat déclaratif v2, composants consommateurs de variables et repli sur les valeurs natives.
- [x] D-THEME-02 — Palette claire/sombre, familles de police, tailles, arrondis, couleurs et états propres aux composants.
- [x] D-THEME-03 — Héritage global et exceptions par rubrique ; paramètres inclus ; compatibilité des paquets v1.
- [x] D-THEME-04 — Exemple complet et catalogue documenté téléchargeables, validation typée et tests empêchant le retour de styles visuels codés en dur.

Historique de la première tranche :

La 0.0.14 apporte un premier contrat de palettes claire/sombre, limité aux couleurs de consultation ; connexion et paramètres conservent le style natif. Les thèmes ne permettent pas encore de déplacer les composants ou d’ajouter des menus/langues. Les imports sont locaux et explicitement non authentifiés ; aucune adresse de dépôt officiel fictive n’a été ajoutée.

- [ ] Variables de thème et composants publics ; dispositions de pages, rangées, cartes, menus, actions et panneaux de paramètres extensibles.
- [ ] Traductions externalisées avec repli, pluriels et formats locaux ; langue de compte utilisée pour l’interface et les requêtes de métadonnées compatibles.
- [ ] Premiers thèmes et langues sous forme de données déclaratives. Les extensions interactives plus puissantes utilisent un contrat d’interface isolé, pas du JavaScript libre ayant tous les droits du site.
- [ ] Les personnalisations ne contournent ni les autorisations serveur, ni les écrans de connexion/permissions/récupération. Compatibilité annoncée séparément pour web, mobile et TV.
- [ ] Catalogue officiel configuré par défaut, identité officielle vérifiable ; ajout administrateur de dépôts GitHub communautaires respectant le format Mosaune.
- [ ] Paquets de versions publiées, manifestes, empreintes/signatures, permissions visibles et compatibilité vérifiée. Une URL GitHub quelconque ne devient pas automatiquement une extension.
- [ ] Ne jamais confondre signature de l’éditeur et garantie de sûreté de son code. Source non officielle clairement indiquée, mises à jour contrôlées.
- [ ] Installation interrompue sans paquet à moitié actif ; sauvegarde avant migration, retour à une version compatible et restauration si migration non réversible.

## Propriété des données et désinstallation

Le stockage est administré par le noyau, avec des espaces et modèles logiques appartenant aux modules média. Il peut rester dans une seule base SQLite ; aucune base concurrente par fournisseur n’est nécessaire. Les fournisseurs donnent des fiches normalisées via une API. La provenance reste enregistrée.

**Désactiver** arrête le code, les accès et les tâches de l’extension ; sa configuration et les données restent conservées pour réactivation.

**Désinstaller** retire le paquet, ses secrets, ses réglages privés et ses données exclusives. Les ressources utilisées par un autre module, une fiche du catalogue ou une correction utilisateur restent conservées. On suit les références réelles, pas seulement le fait que deux extensions utilisent le même fichier SQLite.

Exemple : les résultats TMDB déjà incorporés au catalogue Films sont utilisés par Films ; ils survivent donc au retrait de TMDB. Le cache privé et la clé TMDB disparaissent. Une commande de purge séparée peut retirer les contributions qui ne sont plus souhaitées, avec aperçu de l’impact. Les corrections manuelles et apports d’autres sources ne sont pas supprimés par erreur.

Le retrait d’un module média présente ses dépendants et l’impact sur son catalogue ; aucune suppression silencieuse de médias sur disque. Les fournisseurs partagés restent si d’autres modules en dépendent. Les services externes comme Radarr ne sont pas désinstallés et leurs téléchargements ne sont pas annulés lorsque leur connecteur Mosaune disparaît.

## E. Publication GitHub et installation CasaOS

- [x] E-GITHUB-01 — Dépôt public du serveur, premiers commits et guide GitHub ; médias, catalogue personnel, secrets et notes locales exclus.
- [x] E-GITHUB-02 — Tests automatiques validés sur Windows/Linux et exécutables Windows amd64, Linux amd64 et Linux arm64 produits depuis les seules sources du dépôt (première exécution GitHub réussie le 17 octobre 2026).
- [ ] Choisir la licence du projet avant de le présenter comme open source ; compléter les notices de distribution et le processus de release.
- [x] E-NETWORK-01 — HTTPS direct, origine autorisée, cookies sécurisés, certificat et arrêt SIGTERM vérifiés dans les conteneurs amd64 et arm64 (ARM sous émulation).
- [x] Fabrication d’images Linux ARM64 et amd64 après les tests ; publication GHCR de `0.0.24-preview` et d’une étiquette liée au commit, récupération anonyme vérifiée.
- [ ] Définition CasaOS/Compose et catalogue Mosaune accessible par lien adapté à la version de CasaOS installée. Version utilisateur confirmée : CasaOS 0.4.15. Modèle et archive préparés ; import réel encore à valider.
- [x] Volume persistant `/data` pour catalogue, configuration, images choisies et paquets ; médias en lecture seule. Compte et catalogue conservés après redémarrage en test conteneur ; droits des dossiers réels du NAS encore à préparer.
- [ ] Canaux stable et test avec numéros de version identifiables ; choisir une mise à jour dans CasaOS, conserver les données, sauvegarder avant les migrations.
- [ ] Vérifier sur Pi installation, redémarrage, mise à jour et restauration ; mesurer la consommation et valider deux comptes et les permissions.
- [ ] Option ultérieure, non bloquante pour le NAS : déplacer une bibliothèque ou migrer un catalogue existant en conservant associations et progressions. L’utilisateur choisit une installation NAS neuve.

Parcours cible : publication d’une version sur GitHub → tests et construction automatique → image sur GHCR → récupération de cette version par CasaOS. La publication seule ne met pas le NAS à jour : il doit récupérer l’image et recréer le conteneur en gardant ses volumes. L’automatisation de cette dernière action reste un choix distinct.

## Limite du premier jalon NAS

Objectif confirmé le 19 octobre 2026 : comptes séparés, paramètres organisés, affiches personnalisées, système d’extensions fonctionnel couvrant les familles prévues et lecteurs Mosaune pour films, séries et musique, puis installation neuve sur CasaOS. Les critères de la section « Priorités confirmées » et des étapes C/D s’appliquent ; un pilote de fournisseur ne suffit plus à autoriser le jalon NAS. Le nombre de contributions communautaires disponibles et les applications Android/TV ne conditionnent pas ce premier essai.

Les étapes A à D restent testables sur Windows. Les essais ARM64 et la préparation du conteneur peuvent avancer avant le déploiement définitif pour détecter tôt les contraintes matérielles.

## Références vérifiées

- [OWASP — stockage des mots de passe](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [GitHub — publication d’images Docker](https://docs.github.com/en/actions/tutorials/publish-packages/publish-docker-images)
- [CasaOS — guide des catalogues tiers et formats](https://github.com/IceWhaleTech/CasaOS-AppStore/blob/main/docs/guides/third-party-store-guide.md)
