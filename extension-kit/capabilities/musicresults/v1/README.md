# Interprétation des réponses musicales — v1

Le composant `metadata.music-results.v1` interprète les réponses du fournisseur musical sans accès au réseau, aux fichiers ou à la base. Le premier module fourni interprète le JSON MusicBrainz : listes d’albums, édition détaillée, artistes et identité de l’artiste d’un album de référence.

Depuis 0.0.54, le module prépare les requêtes MusicBrainz. Le serveur reste responsable de leur exécution, de leur cadence, du cache, des permissions et du stockage des images. Ce composant ne constitue donc pas encore un fournisseur indépendant complet.

Un paquet exécutable schema 2 annonce une contribution de rôle `metadata.music-results.v1`, de cible `music` et `minCore: 0.0.55`. Aucun service ni dépendance ne sont acceptés. Les identifiants `mosaune.*` sont réservés ; un auteur emploie son propre identifiant. Importation, sélection, désactivation, retrait, remplacement et restauration utilisent le gestionnaire commun. Le composant appartient à Musique, emplacement `music-results`.

Le programme répond à `describe` avec `{"Version":1}`, puis à `decode` avec les types Query/Result de contract.go. Operation vaut releases, release, artist, artists ou seed. Body contient une réponse JSON de 512 Kio maximum ; ID identifie la requête pour release/artist/seed, Page est comprise entre 1 et 50. Aucun secret n’est transmis. Les autres champs de la réponse MusicBrainz peuvent être ignorés par le module.

Résultats : au plus 20 albums, 200 pistes par édition, 30 contributeurs, pages bornées à 50, identités MusicBrainz vérifiées, texte et présentation bornés. L’identité de l’édition retournée doit correspondre à celle demandée. Les références de pochettes sont des identifiants ; aucune URL fournie par le module ne devient une requête arbitraire. Le serveur valide à nouveau tous les résultats.

Limites dédiées : enveloppe de 1 Mio, sortie de 512 Kio, mémoire linéaire de 64 Mio, 4 secondes par appel. Les plafonds globaux du runtime et du SDK restent inchangés. Les réponses dépassant la limite sont refusées explicitement ; les métadonnées existantes sont conservées. Un seul traitement musical par instance de fournisseur ; son moteur est fermé après l’opération, sans moteur conservé au repos. Le transport réseau garde ses délais propres.

Le code Go du décodeur est dans `modules/musicresults`, son point d’entrée Wasm et son descripteur dans `modules/music-results`. Seuls le binaire et les types du contrat sont intégrés au serveur, jamais l’implémentation Go du décodeur. Les archives d’exemple des familles 6.0/6.1 incluent ce composant pour Musique. Les versions publiées antérieures restent inchangées.

## Préparation des requêtes — depuis 0.0.54

Le programme répond aussi à describe-requests avec {"Version":1} et à plan avec les types Request/RequestPlan de requests.go. Le descripteur et l’emplacement restent identiques : on étend le même composant, sans nouvelle dépendance.

Request exprime une intention : discover, match, artists, release, seed, artist, artist-releases ou upcoming. Title et Artist sont du texte utilisateur (1 200 octets chacun), ID une identité validée, Page entre 1 et 50. Pour upcoming, l’hôte fournit Date en UTC au format YYYY-MM-DD : le module n’a besoin ni d’horloge ni de service réseau.

Exemple d’entrée : {"Operation":"match","Title":"Album","Artist":"Chanteur","Page":2}.
Le plan contient Path, Params, Decode et ID : par exemple {"Path":"/release/","Params":{"query":"release:\"Album\" AND artist:\"Chanteur\"","limit":"20","offset":"20"},"Decode":"releases","ID":""}.

L’hôte valide le chemin release/artist attendu et les identités. Seuls query, inc, limit (1–20) et offset (0–980) sont admis ; quatre paramètres maximum, valeurs bornées à 4 096 octets. Les en-têtes, origine réseau, méthode GET, format JSON, cadence, cache et délais restent sous contrôle de l’hôte. Le module ne reçoit aucun secret. Après le réseau, le même module interprète la réponse via decode. Une seule session et le même signal d’annulation couvrent plan, réseau et décodage.

Les paquets sans describe-requests v1 sont refusés à l’import ; ceux déjà installés restent conservés mais leurs recherches nécessitent une mise à jour. Le paquet fourni 0.1.0 est migré automatiquement vers 0.2.0, sans changement d’activation ni de sélection. La migration exige son identité réservée, l’absence de manifeste/archive d’import et des octets correspondant à l’empreinte enregistrée dans le registre. Elle n’impose pas un hash de compilation unique, qui varierait selon la version de Go. Il n’y a aucun repli vers une construction de requêtes dans le noyau.

Depuis 0.0.55, le même composant pilote aussi les images : [contrat de workflow des pochettes et portraits](VISUALS.md). Le transport et les domaines autorisés restent sous contrôle du noyau.
