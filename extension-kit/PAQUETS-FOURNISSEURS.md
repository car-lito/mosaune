# Paquets de fournisseurs — pilote 0.0.19

## Archive et confiance

Un paquet contient exactement trois fichiers à sa racine : `extension.json`, `provider.wasm` et `signature.ed25519`. Maximum 9 Mio pour l’archive, 8 Mio pour le programme, 64 Kio pour le manifeste. Les chemins, fichiers supplémentaires, doublons, liens et tailles décompressées excessives sont refusés ; aucune extraction n’est effectuée sur le système de fichiers.

Le manifeste est signé avec Ed25519 et contient le SHA-256 du programme. L’hôte vérifie la signature sur les octets exacts du manifeste avant son interprétation, puis vérifie les déclarations et l’empreinte du Wasm avant toute compilation. La clé publique reconnue est `provider-package-key.pub`. Elle correspond à une clé de développement propre à ce projet, pas à une identité vérifiée par un service extérieur.

La signature autorise des mises à jour provenant de cette clé ; elle n’est pas une preuve d’absence de bogue. Les paquets signés par d’autres clés restent refusés. Les budgets globaux de compilation et de processus doivent encore être renforcés et mesurés sur ARM avant l’ouverture au code communautaire non fiable. Les plafonds actuels de mémoire linéaire ne représentent pas la consommation totale du processus.

## Contrat du pilote

`packageformat.Manifest` sépare le nom, l’éditeur, la version minimale du serveur, l’adaptateur hôte, le descripteur du programme et son schéma de configuration. Pour ce premier adaptateur, seules les déclarations suivantes sont admises :

- Adaptateur `tmdb.v1`, identité `mosaune.tmdb-search`.
- Capacités `metadata.search.v1`, `metadata.video-details.v1` et `system.health.v1`.
- Service `tmdb.api.v1`, opération `get` ; contraintes de route et d’œuvre contrôlées par l’hôte à chaque appel.
- Configuration `apiToken`, secret de 20 à 2048 octets, format `token`, stocké dans la connexion TMDB commune.

L’archive ne peut pas sélectionner une autre table, accéder aux fichiers, changer les domaines réseau, demander des secrets supplémentaires ou injecter du JavaScript. Les nouvelles familles de fournisseurs nécessiteront des adaptateurs et des contrats explicites ; ce pilote ne prétend pas être un chargeur universel.

## Contrôle de démarrage et remplacement

Avant toute installation, le candidat est compilé, activé dans une instance temporaire puis appelé avec `system.health.v1/check`. Il n’a aucun service ni secret pour ce contrôle et doit répondre avec sa version exacte et le fournisseur `tmdb`. Le contrôle dispose de trois secondes ; la compilation a son délai propre. Ce contrôle vérifie le démarrage et l’identité de version, pas toute la qualité sémantique des métadonnées futures.

Le remplacement attend la fin de l’invocation courante. Les archives actuelle et précédente sont enregistrées dans une seule transaction SQLite, avec comparaison des empreintes vues dans l’aperçu. Le processus ne bascule sur le candidat qu’après validation et validation de la transaction. Échec de compilation, de contrôle, d’enregistrement ou aperçu devenu périmé : la version actuelle reste inchangée.

Première installation désactivée ; mise à jour conservant l’état courant. Une version inférieure n’est admise que par l’action explicite de restauration de la version conservée. Cette restauration est testée de la même manière et échange les deux versions. Une seule version de retour est conservée ; une sauvegarde du dossier data reste nécessaire pour un historique plus long.

Au redémarrage, le programme est lu depuis l’archive installée, sa signature vérifiée à nouveau et son moteur chargé à la demande. Le mode de secours neutralise l’exécution. Désinstaller supprime les deux archives et désactive le module ; les données communes et la clé utilisée par le fournisseur intégré sont conservées.

## Construire une publication indépendante

Depuis la racine du projet, avec Go installé :

```powershell
./scripts/build-provider.ps1 -Key 'C:/chemin-prive/mosaune-provider.private' -Version '0.4.2'
```

Le script compile le programme du fournisseur avec sa propre version et signe `packages/tmdb-0.4.2.mosaune.zip`. Il ne reconstruit pas le serveur. La clé privée existante est requise ; son contenu n’est jamais affiché ni embarqué dans le paquet. La clé publique seule peut être versionnée. Conserver une sauvegarde privée de la clé ; la perdre exige une rotation explicite de confiance. Le script refuse de remplacer silencieusement une clé publique déjà différente.

Pour un nouveau projet sans clé publique, l’option `-NewSigningKey` initialise une paire. Elle n’autorise pas la rotation d’un serveur existant. Ne pas publier les fichiers `.private` : ils sont aussi exclus par le `.gitignore` du projet.

La reconstruction ordinaire du serveur utilise `scripts/build-app.ps1` et les deux exemples signés déjà présents. Elle ne nécessite pas la clé privée. Les deux exemples 0.4.0 et 0.4.1 ont des versions embarquées distinctes et permettent un essai complet d’installation/mise à jour/restauration dans l’interface.
