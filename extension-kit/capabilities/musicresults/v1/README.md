# Interprétation des réponses musicales — v1

Le composant `metadata.music-results.v1` interprète les réponses du fournisseur musical sans accès au réseau, aux fichiers ou à la base. Le premier module fourni interprète le JSON MusicBrainz : listes d’albums, édition détaillée, artistes et identité de l’artiste d’un album de référence.

Le serveur reste responsable des requêtes MusicBrainz, de leur cadence, du cache, des permissions et des images. Ce composant ne constitue donc pas encore un fournisseur indépendant complet.

Un paquet exécutable schema 2 annonce une contribution de rôle `metadata.music-results.v1`, de cible `music` et `minCore: 0.0.53`. Aucun service ni dépendance ne sont acceptés. Les identifiants `mosaune.*` sont réservés ; un auteur emploie son propre identifiant. Importation, sélection, désactivation, retrait, remplacement et restauration utilisent le gestionnaire commun. Le composant appartient à Musique, emplacement `music-results`.

Le programme répond à `describe` avec `{"Version":1}`, puis à `decode` avec les types Query/Result de contract.go. Operation vaut releases, release, artist, artists ou seed. Body contient une réponse JSON de 512 Kio maximum ; ID identifie la requête pour release/artist/seed, Page est comprise entre 1 et 50. Aucun secret n’est transmis. Les autres champs de la réponse MusicBrainz peuvent être ignorés par le module.

Résultats : au plus 20 albums, 200 pistes par édition, 30 contributeurs, pages bornées à 50, identités MusicBrainz vérifiées, texte et présentation bornés. L’identité de l’édition retournée doit correspondre à celle demandée. Les références de pochettes sont des identifiants ; aucune URL fournie par le module ne devient une requête arbitraire. Le serveur valide à nouveau tous les résultats.

Limites dédiées : enveloppe de 1 Mio, sortie de 512 Kio, mémoire linéaire de 64 Mio, 4 secondes par appel. Les plafonds globaux du runtime et du SDK restent inchangés. Les réponses dépassant la limite sont refusées explicitement ; les métadonnées existantes sont conservées. Un seul traitement musical par instance de fournisseur ; son moteur est fermé après l’opération, sans moteur conservé au repos. Le transport réseau garde ses délais propres.

Le code Go du décodeur est dans `modules/musicresults`, son point d’entrée Wasm et son descripteur dans `modules/music-results`. Seuls le binaire et les types du contrat sont intégrés au serveur, jamais l’implémentation Go du décodeur. Les archives d’exemple des familles 4.0/4.1 incluent ce composant pour Musique. Les versions publiées antérieures restent inchangées.
