# Archives de familles — format 1

Disponible à partir de Mosaune 0.0.47. Ce format regroupe les contrats existants, il ne rend pas encore les services métier indépendants et ne permet pas d’inventer un nouveau type de média.

## Structure

Une archive ZIP contient family.json, une archive exécutable par composant et une archive d’écran. Aucun chemin de sous-dossier, fichier supplémentaire, doublon, lien ou extraction sur le disque n’est accepté. Le manifeste n’exécute pas de script.

Champs de family.json : schema=1, id, name, publisher, version, minCore, kind, components, screen. kind vaut films, series ou music. L’identité d’éditeur est déclarative : les identifiants mosaune.* et org.mosaune.media.* sont réservés pour la famille, les restrictions existantes des composants sont conservées.

Chaque entrée components contient slot, file et sha256. Fournir exactement les slots du manifeste de composition extensions/media-<kind>/extension.json. Chaque archive exécutable garde son propre extension.json (schéma 2), son module.wasm, son identifiant et sa version. Le rôle doit correspondre au slot. Les capacités et services restent limités par les validateurs actuels.

screen possède slot=screen, file et sha256. Son archive contient un manifeste d’interface API 3 avec exactement une page fournissant la rubrique kind. Les identifiants org.mosaune.screens.* sont réservés. Aucun changement de rubrique pendant une mise à jour.

Les trois exemples téléchargeables dans Mosaune donnent des manifestes et programmes complets en versions 1.0.0 et 1.1.0. Décompressez l’exemple dans votre dossier de travail pour inspecter ses composants. Les modules partagés emploient des identifiants communs entre exemples afin d’éviter de multiplier les installations lorsque ces modules sont nécessaires.

## Construire une archive

Préparez les archives exécutables avec l’outil existant extension-kit/cmd/package-module. Placez-les avec l’archive d’écran à côté de family.json. Depuis le dépôt :

    go run scripts/package-family.go chemin/vers/family.json famille.zip

L’outil recalcule les empreintes de chaque membre et produit un ZIP déterministe. Il n’exécute pas les programmes et ne remplace pas la validation de compatibilité effectuée par le serveur. Gardez une nouvelle version pour chaque contenu différent.

## Installation et mise à jour

POST /api/v1/media-bundle-archives/preview reçoit application/zip. La réponse contient le manifeste, la version précédente, les opérations prévues et checksum. Envoyez le même fichier à /import avec X-Mosaune-Plan-Checksum. Les routes nécessitent un administrateur, la session et le jeton anti-CSRF ordinaires. Le plan inclut l’empreinte de l’archive, l’état du registre, les choix, les écrans, leurs droits et l’archive précédemment installée.

La prévisualisation n’exécute aucun programme. L’import valide les programmes modifiés dans WASI avant toute écriture persistante. Programmes, versions antérieures individuelles, sélections, écran et enregistrement de la famille partagent une transaction. Les appels concernés sont annulés après validation uniquement.

Les nouveaux programmes et écrans restent en pause. Une mise à jour conserve leurs états ; les accès utilisateurs d’un écran existant sont conservés, aucun nouveau droit n’est accordé automatiquement. Les choix externes ou le retour volontaire à une implémentation fournie restent respectés. Un composant conservé n’est ni installé en doublon ni mis à jour par cette archive. Les nouvelles versions d’une archive doivent conserver le même ID, les mêmes IDs de composants et le même écran ; aucune suppression implicite de composant.

Une archive identique peut réparer ses composants absents lorsqu’ils sont toujours sélectionnés. Une archive de version antérieure est refusée. Une autre identité de famille nécessite un retrait préalable. La restauration globale de famille reste à construire.

## Limites

32 Mio de ZIP externe ; au plus neuf membres ; manifeste 64 Kio ; chaque ZIP exécutable 9 Mio et code 8 Mio ; plafonds cumulés de décompression et de code à 64 Mio chacun. Les ZIP et manifestes internes passent aussi leurs validations existantes. Les paquets sont lus en mémoire sous ces limites, jamais extraits par le serveur.

Schéma de base 31 : media_bundle_archives conserve la composition importée et ses identifiants. Les programmes vivent dans le stockage commun existant ; l’archive externe complète n’est pas conservée. Pas de nouveau processus résident.

À venir : catalogue distant de familles, signatures et confiance d’éditeurs, compatibilités inter-familles, changement de composition, restauration coordonnée, contributions cumulables et nouveaux types de médias.
