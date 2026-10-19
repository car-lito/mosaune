# Stockage des réglages — 0.0.27

Le formulaire déclaratif et son stockage sont deux contrats distincts. `SQLStore` conserve les réglages dans la base de Mosaune, avec un propriétaire par extension, une révision, le schéma accepté et deux espaces pour les valeurs ordinaires et secrètes. Il ne crée pas une base de métadonnées par fournisseur.

## Raccordement côté hôte

L’installateur vérifie d’abord l’identité, la signature et la compatibilité du paquet. Dans la même transaction que l’archive et le registre, il appelle `Register(ctx, owner, schema)`. Le schéma ne choisit ni table, ni fichier, ni requête SQL. Limites : 128 propriétaires, 32 champs par schéma et 256 Kio par enregistrement. Ce n’est pas un stockage de fichiers ou de caches.

`For(owner)` fournit un accès lié à un propriétaire. Cet accès est réservé au code de confiance de Mosaune : il n’est pas transmis au module Wasm, ni construit à partir d’un propriétaire annoncé par le module. Un auteur de paquet ne reçoit aucun accès aux autres propriétaires en déclarant un champ.

- `Read` renvoie `State`, avec `revision`, `schema`, `values` et `secretsConfigured`. Les valeurs secrètes ne sont jamais incluses, même pour l’administrateur.
- `Apply` valide tous les champs avant une écriture atomique. Les absences conservent les valeurs ; `clearSecrets` retire explicitement une clé. Une révision devenue ancienne provoque `ErrConflict`, sans modification partielle.
- `Secret` est uniquement destiné à un adaptateur de service de confiance. TMDB reçoit un service de requête borné ; le code invité ne reçoit pas la clé. Aucun service générique de lecture de secrets n’est accordé aux extensions.
- `Remove` retire les réglages du seul propriétaire, avec contrôle de révision. L’installateur doit vérifier qu’aucune implémentation restante ne les utilise et effectuer le retrait dans sa transaction. Désactiver un module ne supprime aucun réglage.

Les secrets restent stockés dans SQLite, **sans chiffrement au repos ajouté dans cette étape**. Le masquage de l’API et l’isolation entre extensions ne protègent pas contre quelqu’un qui lit directement la base ou ses sauvegardes. Le chiffrement avec gestion de clé et restauration reste un chantier distinct ; ne pas présenter ce stockage comme un coffre chiffré.

## Mises à jour et retours arrière

Une modification des textes ou l’ajout de champs est possible. Une valeur renseignée doit conserver son type et satisfaire les nouvelles contraintes sans normalisation destructive. Retirer un champ renseigné, transformer un secret en texte ou invalider une valeur provoque `ErrIncompatible`. La même règle s’applique à un retour arrière.

Aucune migration arbitraire fournie par un paquet n’est exécutée. Les migrations explicites des données privées et leur aperçu utilisateur restent à construire. Une erreur de schéma ou d’écriture annule aussi l’installation du paquet lorsque l’hôte utilise la transaction commune.

## API et premier utilisateur réel

`GET` et `PUT /api/v1/extensions/{id}/configuration` sont réservés à l’administrateur. Un module installé doit être présent dans le registre commun et posséder un schéma enregistré. Une ligne de stockage isolée ne suffit pas. L’ancien chemin `/extensions/bundled/{id}/configuration` reste accepté. Le corps d’une modification ne peut pas redéfinir le propriétaire.

Le paquet TMDB et le fournisseur intégré partagent explicitement `mosaune.tmdb-search`. La migration au schéma 16 déplace la clé existante puis enlève l’ancienne copie, dans la même transaction et après sauvegarde du catalogue. Les deux interfaces de configuration et les deux moteurs utilisent cette même clé. Retirer le paquet rétablit le formulaire intégré et garde la clé, puisque le fournisseur intégré reste présent.

Les formulaires envoient la révision affichée. En cas de conflit, l’administrateur peut recharger les réglages avant de recommencer. Le schéma de paquet pilote reste limité au champ TMDB existant ; cette étape ne rend pas encore installables toutes les familles d’extensions.

## Vérifications

Les tests utilisent SQLite et des clés factices : propriétaires isolés, réouverture de la base, réponses sans secrets, refus des changements de type, corruption conservée plutôt que réinitialisée, concurrence, écritures sans effet partiel, migrations réussies et refusées, échec de mise à jour annulant archive et registre, conservation de la clé lors du retrait du paquet et routes génériques réservées à l’administrateur.
