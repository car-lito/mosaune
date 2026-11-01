# Projection de catalogue v1

La capacité media.catalogue.v1 transforme les informations visibles d’un fichier en une proposition de groupe et de tri. Elle ne reçoit aucun accès à SQLite, au disque ou au réseau. Les règles officielles sont compilées dans trois programmes WASI distincts ; le serveur ne lie pas leur code Go et l’interface ne reconstitue pas les règles d’album ou de série.

## Protocole

- describe avec un objet vide renvoie {"version":1}.
- project reçoit {"items":[Facts,...]} et renvoie {"items":[Item,...]}, dans le même ordre et avec exactement la même longueur.
- Facts : identifiant opaque, nom, nom de fichier sans dossier, titre, fournisseur/identifiant, série, artiste, artiste d’album, album, année, saison, épisode, disque et piste. Le serveur filtre les permissions avant cet appel. Il transmet des copies nettoyées, limitées à 512 octets UTF-8 par texte ; la base conserve les valeurs complètes.
- Item : key, title, subtitle, order (deux entiers), sortText. key vide représente un fichier individuel. L’interface délimite chaque clé par type de média et identifiant du module : un module ne peut fusionner ses résultats avec ceux d’un autre type ou éditeur.
- Contrat limité à 128 éléments maximum ; le serveur 0.0.50 traite des lots de 64, réduits automatiquement si le JSON dépasse 48 Kio. Réponse vérifiée, textes de 2 Kio maximum et sans caractères de contrôle, titre non vide, entiers de tri entre 0 et 1 000 000. Aucun HTML n’est exécuté.

La clé doit être stable et ne pas dépendre de la position ou du contenu des autres éléments du lot. Un lot ne représente pas toute la collection. Les métadonnées sélectionnées ne sont pas toutes les propriétés possibles d’un média : étendre le contrat par une capacité versionnée lorsqu’un autre usage le nécessite.

## Paquet et remplacement

Manifeste exécutable schéma 2, une contribution de rôle media.catalogue.v1 et cible films, series ou music. Une seule capacité, aucun service ni dépendance pour cette version. Type de fiche de dépôt : catalogue. Mosaune 0.0.35 minimum. Identités mosaune.* réservées aux paquets fournis.

L’import local et les dépôts GitHub passent par le même aperçu et le même installateur que les reconnaisseurs. Installation désactivée ; activation et sélection distinctes. Une mise à jour conserve le rôle et le choix. Une version précédente reste restaurable. Exemple indépendant : examples/catalogue ; la construction produit build/example-catalogue.mosaune.zip.

## Cycle de vie et ressources

Le serveur calcule la projection à la lecture de la bibliothèque autorisée, sans réécrire les métadonnées. Pour une projection nouvelle, un moteur est chargé par type présent dans la requête, utilisé pour ses lots puis fermé. Un cache mémoire conserve au maximum une projection par type (2 Mio de JSON par projection, plus les allocations Go). Sa clé comprend le programme, son identité et exactement les informations déjà filtrées pour le compte. Changer une métadonnée, la sélection ou le programme invalide la projection ; l’état actif est contrôlé même pour un résultat en cache. Aucun cache sur disque. Les requêtes de projection sont sérialisées par catalogue pour limiter les moteurs simultanés. Limites communes : mémoire linéaire WASM 64 Mio (ce n’est pas la mémoire totale du processus), délai par invocation de 4 secondes et budget de projection de 45 secondes par type depuis 0.0.50. La réponse JSON est bornée par le transport avant décodage, sans sérialisation supplémentaire de validation dans le programme invité.

Une désactivation, un retrait ou un changement de sélection annule les calculs en cours. Les résultats tardifs sont refusés avant publication. En cas de panne, de désactivation ou de mode secours, tous les fichiers concernés restent visibles individuellement et un message explique l’absence d’organisation. Aucun remplacement implicite par le programme officiel.

Une réactivation ne demande pas de nouveau scan. Les clients déjà ouverts récupèrent le nouveau résultat au prochain chargement de bibliothèque ; l’interface qui change le module se rafraîchit immédiatement. Les données historiques, pochettes, progressions et fichiers ne sont pas supprimés.

## Périmètre restant

Ce contrat extrait la composition des groupes et leur ordre, pas toute la famille média. Les associations aux œuvres distantes, les saisons détaillées, les pages, les playlists, les connecteurs et la lecture ont encore des règles à extraire. Il n’autorise pas encore un nouveau type de média inconnu du noyau.

## Index de recherche optionnel — Mosaune 0.0.50 minimum

Item peut déclarer search: {version: 1, entries: [...]}. Chaque entrée contient target (group ou item), title, subtitle, terms (au plus huit textes), priority (0 à 100), presentation (contrat presentation/v1 obligatoire) et, facultativement, metadataPresentation. Ce dernier autorise explicitement une présentation plus précise provenant des métadonnées associées. Une seule entrée par cible, au plus quatre entrées dans l’enveloppe ; le contrat courant offre deux cibles. Réponse complète limitée à 256 Kio, textes à 2 Kio sans contrôles. Les inconnues du protocole sont rejetées.

Le résultat correspond toujours au fichier d’entrée, et group à son groupe projeté : aucune identité externe, URL, commande ou permission ne peut être injectée. La navigation et les images sont résolues par l’hôte. Les thèmes continuent de contrôler les styles ; les formats d’image restent bornés par presentation/v1.

Le moteur générique normalise accents, casse et séparateurs, exige tous les mots, puis classe titre exact, préfixe, titre contenant et mots-clés ; priority départage ces classes. Les groupes sont dédoublonnés, avec un maximum de 24 résultats locaux (12 dans l’interface actuelle). Aucun appel de programme à chaque frappe.

Index absent : compatibilité avec les anciens composants, recherche par nom de fichier uniquement, sans reconstituer des règles d’albums ou de séries. Index présent avec liste vide : aucun candidat, choix respecté. Un module absent/désactivé ne fournit plus son index, même s’il a été mis en cache. Les autres clients déjà ouverts actualisent leur bibliothèque pour recevoir l’état nouveau.

Les modules officiels 0.2 fournissent films, séries/épisodes et albums/morceaux. L’exemple indépendant catalogue 2.0 fournit mots-clés, type et vignette différents. Les familles d’exemple 3.x nécessitent 0.0.50 ; les archives publiées antérieures restent immuables. Ce contrat ne choisit pas les fournisseurs distants et n’implémente pas la recommandation.
