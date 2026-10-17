État au 17 octobre 2026 : catalogue officiel public, intégration et dédoublonnage 0.0.22 livrés. Sources du serveur publiées sur https://github.com/car-lito/mosaune ; vérifications Windows/Linux et construction ARM64 automatisées. 0.0.23 : HTTPS direct et arrêt SIGTERM validés, images Docker amd64/arm64 publiées et téléchargements anonymes vérifiés. Catalogue CasaOS 0.4.15 préparé ; import réel encore à tester. Aucun déploiement NAS.

# Mosaune — comptes, personnalisation et extensions avant le NAS

## Obligatoire avant le NAS : désactivation réelle des fournisseurs

- [ ] C-LIFECYCLE-01 — Distinguer l’activation du fournisseur du choix de son moteur (intégré ou extension). « Désactivé » doit arrêter le fournisseur, sans retour silencieux au moteur intégré.
- [ ] Bloquer tous ses nouveaux accès réseau : recherche, découverte, fiches, images, enrichissement automatique et tâches de fond ; annuler les traitements en cours lorsque possible et ignorer leurs réponses tardives.
- [ ] Conserver les métadonnées communes déjà acquises, les affiches conservées localement, les corrections et les préférences. Ne pas affecter les autres fournisseurs actifs, par exemple MusicBrainz lorsque TMDB est arrêté.
- [ ] Expliquer l’état dans l’interface et tester l’absence de nouveaux appels, y compris après redémarrage et lors d’une recherche déjà en cours.

Constat 0.0.22 : l’interrupteur TMDB désactive actuellement son moteur d’extension et réactive le fournisseur intégré. Ce chantier n’est donc pas terminé, même si le cycle de vie du paquet exécutable fonctionne.

## Avancement 0.0.20 — catalogue et dépôts publics

- [x] C-STORE-01 — Index versionné séparé des formats de paquets, cartes filtrables, choix de version et installation après aperçu.
- [x] C-STORE-02 — Dépôts GitHub publics ajoutables ; actualisation explicite, cache du dernier succès, retrait indépendant de la désinstallation.
- [x] C-STORE-03 — Téléchargement borné, intégrité et concordance du manifeste, politique de signature des fournisseurs conservée, droits admin.
- [x] C-STORE-04 — Catalogue intégré de démonstration et export d’un dépôt complet prêt à publier, guides utilisateur/auteur.
- [x] C-STORE-05 — Publication du vrai dépôt officiel et ajout de son adresse au catalogue par défaut (0.0.21).
- [ ] C-TRUST-01 — Clés/éditeurs communautaires exécutables et budgets globaux avant ouverture au code non fiable.

Les thèmes tiers sont acceptés ; les fournisseurs exécutables restent les paquets TMDB signés reconnus. Aucun dépôt privé, autre hébergeur ou mise à jour automatique. Voir ESSAYER-LE-CATALOGUE-020.md.

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
- [x] Fabrication d’images Linux ARM64 et amd64 après les tests ; publication GHCR de `0.0.23-preview` et d’une étiquette liée au commit, récupération anonyme vérifiée.
- [ ] Définition CasaOS/Compose et catalogue Mosaune accessible par lien adapté à la version de CasaOS installée. Version utilisateur confirmée : CasaOS 0.4.15. Modèle et archive préparés ; import réel encore à valider.
- [x] Volume persistant `/data` pour catalogue, configuration, images choisies et paquets ; médias en lecture seule. Compte et catalogue conservés après redémarrage en test conteneur ; droits des dossiers réels du NAS encore à préparer.
- [ ] Canaux stable et test avec numéros de version identifiables ; choisir une mise à jour dans CasaOS, conserver les données, sauvegarder avant les migrations.
- [ ] Vérifier sur Pi installation, redémarrage, mise à jour et restauration ; mesurer la consommation et valider deux comptes et les permissions.
- [ ] Migrer les chemins du catalogue Windows vers les dossiers montés du NAS sans perdre associations et progressions.

Parcours cible : publication d’une version sur GitHub → tests et construction automatique → image sur GHCR → récupération de cette version par CasaOS. La publication seule ne met pas le NAS à jour : il doit récupérer l’image et recréer le conteneur en gardant ses volumes. L’automatisation de cette dernière action reste un choix distinct.

## Limite du premier jalon NAS

Objectif proposé : comptes réellement séparés, paramètres organisés, affiches personnalisées et première version fonctionnelle du système de paquets (y compris ajout d’un dépôt communautaire et retrait propre), puis déploiement CasaOS. Le catalogue communautaire fourni, la liberté absolue de refaire tous les écrans et la parité des thèmes sur Android/TV ne sont pas des prérequis du premier essai NAS.

Les étapes A à D restent testables sur Windows. Les essais ARM64 et la préparation du conteneur peuvent avancer avant le déploiement définitif pour détecter tôt les contraintes matérielles.

## Références vérifiées

- [OWASP — stockage des mots de passe](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [GitHub — publication d’images Docker](https://docs.github.com/en/actions/tutorials/publish-packages/publish-docker-images)
- [CasaOS — guide des catalogues tiers et formats](https://github.com/IceWhaleTech/CasaOS-AppStore/blob/main/docs/guides/third-party-store-guide.md)
