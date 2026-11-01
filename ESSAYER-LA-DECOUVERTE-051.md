# Recherche et découverte — 0.0.51

Cette étape change surtout la façon de brancher les fournisseurs. Elle ne crée pas de nouvelle extension à gérer.

Après avoir conservé une copie de data, arrêté votre ancien serveur et relancé DEMARRER-WINDOWS.cmd :

1. Dans Films ou Séries, recherchez un titre : les résultats locaux et ceux des fournisseurs disponibles se mélangent toujours.
2. Si plusieurs fournisseurs compatibles sont présents, ils participent à la recherche, au lieu de retenir seulement le premier.
3. Désactivez TMDB : les films locaux restent trouvables et le message indique la source réellement indisponible. Une autre source compatible peut continuer à répondre.
4. Dans Musique, les prochaines sorties utilisent les identifiants des contributeurs de vos albums identifiés ou favoris, comme auparavant. Aucun appel n’est lancé si aucun contributeur n’est connu.

Les textes de recherche et les types de sélections viennent maintenant de déclarations communes. Les thèmes et les grandes familles Films/Séries/Musique restent en place.

Pour les contributeurs, voir extension-kit/discovery/v1/README.md. Les tests utilisent une source fictive indépendante pour vérifier que les écrans et le serveur ne nécessitent pas une nouvelle condition spécifique à son nom.

Limite actuelle : le nouveau contrat est relié aux adaptateurs du serveur. Il faudra encore permettre à tous les paquets fournisseurs externes de livrer leur adaptateur et leur déclaration. MusicBrainz n’est pas encore un paquet indépendant. Cette version ne prétend donc pas terminer la découpe de la découverte, ni ajouter la prise en charge d’un type de média supplémentaire.
