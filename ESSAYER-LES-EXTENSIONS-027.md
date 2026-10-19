# Extensions — étape 0.0.27

Cette version généralise le stockage des réglages et le raccorde à TMDB. Elle ne change pas la disposition principale de l’interface et ne termine pas encore le système d’extensions.

Les paramètres de chaque extension ont leur propre espace. Les clés ne sont pas renvoyées à l’interface. Les modifications concurrentes sont détectées : une vieille fiche ne peut pas écraser une configuration plus récente sans rechargement.

La clé TMDB existante est reprise au premier lancement, après sauvegarde du catalogue. Le moteur intégré et le paquet installé continuent de la partager ; il n’est pas nécessaire de la ressaisir.

## Essai facultatif

1. Lance la version 0.0.27 puis ouvre **Réglages → Extensions → TMDB**.
2. Vérifie que la mention **Clé enregistrée** apparaît si une clé était déjà configurée.
3. Continue tes recherches normalement. Désactiver puis réactiver TMDB ne doit pas retirer la clé.

Les tests automatiques vérifient les conflits entre fenêtres, les échecs de mise à jour et l’absence de clés dans les réponses. Tu n’as pas besoin de modifier ta vraie clé pour ces essais.

Le catalogue passe au schéma 16. Pour revenir à 0.0.26, restaure la sauvegarde créée avant cette migration. Aucun essai NAS n’est demandé.

Suite : autorisations et services des autres familles, choix des dépendances dans l’interface, puis extraction de Films, Séries et Musique. Les lecteurs viennent ensuite. Contrat développeurs : [Stockage des réglages](extension-kit/configuration/STORAGE.md).
