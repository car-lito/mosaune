# Registre durable des extensions — 0.0.26

Le registre conserve les modules, leurs descripteurs, l’empreinte du code vérifié, leur activation souhaitée et leurs choix de fournisseurs. Il utilise la base SQLite de l’hôte ; aucune base de métadonnées supplémentaire n’est créée par fournisseur.

## Intégration actuelle

Le paquet TMDB installé est le premier utilisateur réel du registre. Installation, mise à jour, restauration, choix de moteur et désinstallation mettent à jour l’archive et le registre dans la même transaction SQLite. Le démarrage à la demande vérifie la correspondance avec le paquet authentifié et utilise les choix enregistrés. La fiche du paquet affiche « Dépendances et démarrage ».

La migration du catalogue vers le schéma 15 est précédée de la sauvegarde habituelle. Elle reprend les paquets TMDB existants dont la signature est vérifiée et leur choix de moteur, sans lancer de code. Un paquet illisible/non authentifiable n’est pas exécuté ni enregistré comme valide ; il reste visible via son installateur pour réparation. Les thèmes et fonctions intégrées n’utilisent pas encore ce registre. Le pilote TMDB refuse toujours les dépendances externes : l’intégration de plusieurs fournisseurs, des gros modules média et du choix interactif des alternatives reste à faire.

## Préparer et enregistrer une opération

`Snapshot` contient `schema`, `revision` et `entries`. Chaque entrée conserve un descripteur, `codeSha256`, `enabled` (activation souhaitée) et `choices` (capacité → identifiant du fournisseur).

`Preview(snapshot, change)` est sans effet de bord. Les opérations sont `install`, `update`, `restore`, `enable`, `disable`, `choose`, `remove`. Le résultat donne le prochain état et les modules à désactiver. Une installation démarre désactivée. Une mise à jour doit augmenter la version. `restore` est réservé à un installateur qui a déjà authentifié une version de retour et vérifié son origine ; le registre n’est pas une preuve de confiance ni un historique d’archives.

Une capacité demandée doit être satisfaite par un fournisseur actif compatible. À la première activation, une résolution non ambiguë devient un choix explicite enregistré. Activer ensuite une alternative ne modifie pas ce choix. Modifier un choix actif nécessite de désactiver le consommateur. Une référence manquante, une incompatibilité ou un cycle empêche la validation. Les dépendances désactivées peuvent conserver une préférence même si le fournisseur a été retiré ; leur activation reste alors bloquée.

Un retrait/désactivation refuse de casser un consommateur actif. L’option explicite `cascade` désactive ses consommateurs dans le plan ; elle ne désinstalle pas leurs paquets. Retirer Films ne retire pas automatiquement un lecteur partagé avec Séries. Les données du catalogue et fichiers média sont hors de la responsabilité de ce registre.

`SQLStore.Commit(ctx, expectedRevision, change)` relit l’état, recalcule le plan et écrit par comparaison de révision. Un aperçu ancien reçoit `ErrConflict`. Le client ne peut pas soumettre directement un état final inventé. `SQLStore` accepte une connexion ou une transaction existante : l’installateur peut rendre atomiques l’archive, les réglages d’activation et le registre. Ce composant ne crée aucun endpoint public ni permission supplémentaire.

## Restauration et état réel

`Restore` crée un nouveau moteur, démarre les fournisseurs avant leurs consommateurs et renvoie un `Report` exploitable par une interface. Les modules désactivés ne sont pas compilés. Un moteur déjà utilisé n’est jamais reçu en paramètre : le propriétaire doit fermer l’ancien moteur avant de le remplacer.

La source de code fournie par l’hôte doit authentifier à nouveau les paquets et vérifier permissions/compatibilité. Le registre compare ensuite le descripteur et l’empreinte du code. Une empreinte conservée dans SQLite ne suffit pas à déclarer un paquet sûr.

Les états du rapport sont `active`, `disabled`, `suspended`, `blocked`, `failed`, avec des codes stables (`safe_mode`, `artifact_unavailable`, `artifact_mismatch`, `dependency_unavailable`, `load_failed`, `activation_failed`). L’état souhaité reste distinct de l’état réel. Un fournisseur indisponible bloque ses consommateurs sans empêcher un module indépendant de démarrer et sans basculer silencieusement vers une alternative. Le mode de secours n’appelle pas la source et ne crée aucun moteur. Les messages détaillés des chargeurs ne sont pas exposés dans le rapport.

Le pilote TMDB conserve son démarrage paresseux et son contrôle de santé existants, en consultant le registre ; la restauration commune de plusieurs modules est testée mais n’a pas encore remplacé tous les gestionnaires intégrés. Le registre ne réalise pas à lui seul une mise à jour à chaud, des migrations privées ou l’autorisation d’appeler un autre module.

## Vérifications

- Conservation des choix après fermeture/réouverture de SQLite et création d’un nouveau moteur, puis appel réel d’un module WASI.
- Dépendance partagée, sélection d’alternative, cycles, conflits et mise à jour cassant une capacité.
- Modifications simultanées : une seule révision acceptée ; erreur d’écriture sans état partiel.
- Code ou descripteur différents, fournisseur absent, indépendance des autres modules et mode de secours.
- Installateur TMDB : erreur d’écriture du registre annulant aussi le remplacement de l’archive, l’activation ou le retrait ; restauration de l’ancienne version ; migration avec sauvegarde.

La suite porte sur l’enregistrement des autres familles, les réglages/secrets privés génériques, les permissions de leurs services, puis les écrans de sélection des fournisseurs. Les lecteurs restent une étape ultérieure.
