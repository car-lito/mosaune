# Extensions regroupées, composants extensibles

Décision produit du 30 octobre 2026, après la 0.0.45. Cette cible remplace la présentation de chaque petit composant technique comme une extension indépendante. Elle ne change pas l’exigence d’un noyau minimal avant les lecteurs et le NAS. Ce document décrit la cible. La 0.0.46 livre une première composition fournie et un cycle commun des composants extraits et de leur écran ; les archives de familles et leur cycle complet de mises à jour restent à construire.

## Ce que l’utilisateur installe

Une extension principale correspond à une fonctionnalité compréhensible : **Films**, **Séries**, **Musique**, puis Livres audio, Podcasts, etc. Une installation classique doit demander quelques choix, sans sélection manuelle de chaque reconnaisseur, écran ou élément de fiche.

Les fonctions ayant un usage et un cycle propres restent des extensions distinctes : TMDB, MusicBrainz, Radarr, Sonarr, thèmes et futurs lecteurs alternatifs. Un fournisseur partagé par Films et Séries n’est installé qu’une fois. Il peut être proposé pendant l’installation, avec ses éventuels réglages et accès réseau clairement indiqués.

## Ce que contient une extension

Un paquet principal peut livrer plusieurs composants : reconnaissance, catalogue, écrans, fiches et commandes, et plus tard le lecteur adapté. Il possède une version et une identité d’éditeur, une fiche dans le gestionnaire, ses réglages regroupés et un cycle d’installation/mise à jour/retrait cohérent.

Les composants gardent des contrats explicites et des identifiants stables. Ils ne deviennent pas des blocs de code indissociables. Un contributeur peut modifier une partie ou fournir un remplaçant sans réécrire l’extension principale. L’organisation recommandée des sources suit les fonctionnalités, avec un manifeste de paquet et des sous-dossiers de composants, ainsi qu’un exemple minimal de remplacement.

Un composant partagé peut rester une dépendance technique installée automatiquement et visible dans les détails avancés, plutôt qu’une carte supplémentaire dans la liste principale. Il n’est retiré que lorsqu’aucune extension ni remplacement choisi n’en a besoin.

## Remplacement ou cumul

Chaque point d’extension précise son mode de composition, ses entrées/sorties et ses droits :

- **Remplacement exclusif** : un seul composant choisi pour une fonction donnée, par exemple le lecteur vidéo ou un mode de reconnaissance. Le choix et la règle de retour à une implémentation disponible sont explicites ; aucun changement silencieux après un retrait ou une mise à jour.
- **Contributions cumulables** : plusieurs composants peuvent ajouter des informations, des actions ou des rubriques lorsque le contrat l’autorise. Le noyau contrôle les identités, l’ordre, les doublons, les conflits et les budgets d’exécution. Il n’exécute pas automatiquement plusieurs implémentations d’une fonction exclusive.

Un même paquet peut exposer plusieurs contributions. L’interface principale montre le paquet une seule fois. Un complément dépendant de Films apparaît dans ses réglages avancés et reste traçable avec son éditeur, ses accès et sa version. Le catalogue permet de découvrir les compléments depuis la fiche de leur extension principale ; ils restent également recherchables.

## Installation et cycle de vie

Le manifeste regroupe les composants fournis et déclare les dépendances externes, leur compatibilité, les contributions et les réglages. Le format doit être versionné et compatible avec la migration des installations existantes.

Avant une opération, Mosaune calcule un plan complet : paquets concernés, dépendances partagées, permissions demandées, remplacements conservés et éventuels conflits. Une opération interrompue ne doit pas laisser une famille à moitié installée. Les modifications persistantes sont validées ensemble, puis les appels et contributions concernés sont arrêtés ou rechargés.

Une mise à jour de Films conserve un remplaçant choisi par l’utilisateur. Une suppression ne retire pas une dépendance utilisée par Séries. Les données communes, corrections et médias sont conservés ; les données privées d’un paquet ne sont supprimées qu’en respectant les consommateurs et la politique de retrait annoncée.

## Présentation

La liste principale montre les grandes extensions, fournisseurs, connecteurs, thèmes et compléments installés volontairement. Les composants techniques fournis par défaut sont regroupés dans la fiche de leur propriétaire, sous des réglages lisibles, avec une section avancée permettant de voir et remplacer les composants.

Le bouton principal d’une extension média doit agir sur toute sa fonction et ses contributions. Masquer des cartes sans réunir le cycle d’installation et les dépendances ne remplit pas cette décision.

## Passage depuis l’architecture actuelle

Les contrats WASI, le registre, les droits, les paquets signés, les thèmes et les composants déjà extraits restent réutilisables. Les recettes d’ensembles de la 0.0.40 constituent un point de départ, mais elles ne gèrent pas encore des familles complètes. Les nouvelles capacités doivent désormais être pensées comme des composants de leur extension propriétaire, sans multiplier les extensions principales par défaut.

État en 0.0.46 :

- [x] Trois manifestes de composition versionnés dans extensions/media-*/extension.json, identifiant les composants exclusifs et partagés ainsi que l’écran par défaut.
- [x] Installer/activer, désactiver et retirer ensemble les composants déjà extraits et l’écran ; aperçu, transaction, vérification des conflits et des droits modifiés entre aperçu et validation.
- [x] Adopter les installations existantes sans réactivation ; mémoriser les écrans de remplacement, y compris après leur désactivation ou leur retrait.
- [x] Regrouper les cartes fournies et placer leurs composants dans les réglages avancés ; filtre explicite dans le catalogue.

Travail restant :

- [ ] Étendre les manifestes de composition à des archives distribuables : éditeur, compatibilité, dépendances versionnées et contributions cumulables.
- [ ] Généraliser l’installation aux archives de familles et leur mise à jour coordonnée ; les opérations 0.0.46 réutilisent les installateurs de composants fournis.
- [ ] Fournir un exemple contributeur : une extension principale, un composant remplaçant et une contribution cumulable.
- [ ] Vérifier installation partielle, dépendance partagée, mise à jour avec remplacement, retrait et retour depuis une installation existante.
- [ ] Terminer l’extraction des services encore intégrés et vérifier le noyau seul, puis les familles complètes.

L’estimation globale reste 68 % : cette étape rend la gestion des composants existants plus cohérente, mais n’extrait pas de nouveau service métier. Les familles complètes et le noyau seul restent les critères de fin.
