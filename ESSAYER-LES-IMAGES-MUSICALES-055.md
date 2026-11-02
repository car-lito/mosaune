# Mosaune 0.0.55 — pochettes et portraits dans l’extension

Le composant MusicBrainz de Musique choisit désormais comment trouver une image : pochette de l’édition, autre pochette de l’album si nécessaire, ou portrait provenant des liens MusicBrainz/Wikidata/Commons. Il interprète aussi les crédits et la licence de la photo. Le noyau exécute seulement les étapes autorisées et conserve les images dans le cache partagé.

Après avoir fermé l’ancienne version et lancé la 0.0.55 :

1. Ouvrez un album à découvrir : sa pochette doit apparaître si la source en propose une.
2. Ouvrez la page d’un artiste disposant d’une photo pour vérifier son portrait et ses crédits.
3. Facultatif : désactivez le composant MusicBrainz dans les réglages avancés de Musique. Les pochettes déjà conservées restent disponibles ; de nouvelles images ne sont plus récupérées. Réactivez-le ensuite.

Une image absente chez les fournisseurs peut toujours manquer : cette version ne fabrique pas de pochettes et ne garantit pas que tous les artistes aient une photo.

Le composant fourni passe en 0.3.0. Son état désactivé et les choix de remplaçants sont conservés ; les composants retirés ne sont pas réinstallés. Les anciens remplaçants peuvent continuer leurs recherches s’ils prennent en charge le protocole précédent, mais doivent ajouter le protocole d’images pour récupérer de nouvelles pochettes et photos. Les exemples de familles 6.x l’incluent ; les archives précédemment publiées restent inchangées.

Le transport HTTP, la cadence des requêtes et les adaptateurs des fournisseurs restent dans le serveur. Ce lot n’est donc pas la fin du découpage MusicBrainz. Aucun déploiement NAS n’est effectué.
