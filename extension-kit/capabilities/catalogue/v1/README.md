# Projection de catalogue v1

La capacité media.catalogue.v1 transforme les informations visibles d’un fichier en une proposition de groupe et de tri. Elle ne reçoit aucun accès à SQLite, au disque ou au réseau. Les règles officielles sont compilées dans trois programmes WASI distincts ; le serveur ne lie pas leur code Go et l’interface ne reconstitue pas les règles d’album ou de série.

## Protocole

- describe avec un objet vide renvoie {"version":1}.
- project reçoit {"items":[Facts,...]} et renvoie {"items":[Item,...]}, dans le même ordre et avec exactement la même longueur.
- Facts : identifiant opaque, nom, nom de fichier sans dossier, titre, fournisseur/identifiant, série, artiste, artiste d’album, album, année, saison, épisode, disque et piste. Le serveur filtre les permissions avant cet appel. Il transmet des copies nettoyées, limitées à 512 octets UTF-8 par texte ; la base conserve les valeurs complètes.
- Item : key, title, subtitle, order (deux entiers), sortText. key vide représente un fichier individuel. L’interface délimite chaque clé par type de média et identifiant du module : un module ne peut fusionner ses résultats avec ceux d’un autre type ou éditeur.
- Lots de 128 éléments maximum, réduits automatiquement si le JSON dépasse 48 Kio. Réponse vérifiée, textes de 2 Kio maximum et sans caractères de contrôle, titre non vide, entiers de tri entre 0 et 1 000 000. Aucun HTML n’est exécuté.

La clé doit être stable et ne pas dépendre de la position ou du contenu des autres éléments du lot. Un lot ne représente pas toute la collection. Les métadonnées sélectionnées ne sont pas toutes les propriétés possibles d’un média : étendre le contrat par une capacité versionnée lorsqu’un autre usage le nécessite.

## Paquet et remplacement

Manifeste exécutable schéma 2, une contribution de rôle media.catalogue.v1 et cible films, series ou music. Une seule capacité, aucun service ni dépendance pour cette version. Type de fiche de dépôt : catalogue. Mosaune 0.0.35 minimum. Identités mosaune.* réservées aux paquets fournis.

L’import local et les dépôts GitHub passent par le même aperçu et le même installateur que les reconnaisseurs. Installation désactivée ; activation et sélection distinctes. Une mise à jour conserve le rôle et le choix. Une version précédente reste restaurable. Exemple indépendant : examples/catalogue ; la construction produit build/example-catalogue.mosaune.zip.

## Cycle de vie et ressources

Le serveur calcule la projection à la lecture de la bibliothèque autorisée, sans réécrire les métadonnées. Pour une projection nouvelle, un moteur est chargé par type présent dans la requête, utilisé pour ses lots puis fermé. Un cache mémoire conserve au maximum une projection par type (2 Mio de JSON par projection, plus les allocations Go). Sa clé comprend le programme, son identité et exactement les informations déjà filtrées pour le compte. Changer une métadonnée, la sélection ou le programme invalide la projection ; l’état actif est contrôlé même pour un résultat en cache. Aucun cache sur disque. Les requêtes de projection sont sérialisées par catalogue pour limiter les moteurs simultanés. Limites communes : mémoire linéaire WASM 64 Mio (ce n’est pas la mémoire totale du processus), délai par invocation de 2 secondes et budget de projection de 15 secondes par type.

Une désactivation, un retrait ou un changement de sélection annule les calculs en cours. Les résultats tardifs sont refusés avant publication. En cas de panne, de désactivation ou de mode secours, tous les fichiers concernés restent visibles individuellement et un message explique l’absence d’organisation. Aucun remplacement implicite par le programme officiel.

Une réactivation ne demande pas de nouveau scan. Les clients déjà ouverts récupèrent le nouveau résultat au prochain chargement de bibliothèque ; l’interface qui change le module se rafraîchit immédiatement. Les données historiques, pochettes, progressions et fichiers ne sont pas supprimés.

## Périmètre restant

Ce contrat extrait la composition des groupes et leur ordre, pas toute la famille média. Les associations aux œuvres distantes, les saisons détaillées, les pages, les playlists, les connecteurs et la lecture ont encore des règles à extraire. Il n’autorise pas encore un nouveau type de média inconnu du noyau.
