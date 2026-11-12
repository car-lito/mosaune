# Réglages des connecteurs — adaptateur hôte

Ce paquet est exécuté par le serveur, jamais dans un programme Wasm. Il raccorde la capacité d’acquisition au stockage commun `configuration.SQLStore`. Les formulaires restent décrits par le composant installé ; les contrôles de destinations et les autorisations restent côté serveur.

- Propriétaires réservés : `mosaune.acquisition.radarr` et `mosaune.acquisition.sonarr`. Un programme ne choisit pas son propriétaire SQL. Remplacer l’implémentation conserve la connexion de la capacité.
- Champs ordinaires : adresse, activation, profil, destination et bibliothèque. Clé dans l’espace secret. Le profil est stocké en texte dans le schéma commun, mais reste numérique dans l’API.
- `Load` lit adresse et clé dans un seul instantané SQL. `Save` et `Pause` contrôlent la révision ; une écriture concurrente est refusée. `PrepareConfiguration` ne réutilise la clé qu’à adresse normalisée et connecteur identiques.
- L’adaptateur HTTP utilise la révision persistante avec l’empreinte du programme et du formulaire. Après sauvegarde, il renvoie une nouvelle empreinte ; un ancien onglet doit recharger ses réglages.
- Migration 46 : transaction pour les deux connecteurs, conflits et données invalides refusés sans effacement, suppression des anciennes copies de clés dans `acquisition_settings`. Cette table reste vide pour les migrations historiques, sans lecture ni écriture de configuration au fonctionnement normal. Les sauvegardes historiques peuvent naturellement contenir les anciennes données.

La pause reste possible sans composant actif. Les configurations des deux connecteurs sont séparées et ne sont pas exposées par l’API générique de configuration des modules. Aucun accès supplémentaire n’est donné au Wasm ; acquisition 0.7.0 et familles 14.x restent inchangés. Les clés sont isolées des réponses publiques mais restent sans chiffrement au repos dans SQLite.

Le retrait définitif du propriétaire lorsque tous ses consommateurs ont disparu reste à raccorder au cycle de désinstallation. Les prévisualisations, le rapprochement local et l’adaptation à de nouveaux types de connecteurs restent également à terminer. Ce raccordement n’est pas une revendication d’indépendance complète des connecteurs.
