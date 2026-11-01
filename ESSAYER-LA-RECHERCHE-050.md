# Recherche locale pilotée par les extensions — 0.0.50

Les règles de recherche locale font maintenant partie des extensions de catalogue de Films, Séries et Musique, dans leurs familles habituelles. Aucune nouvelle carte d’extension à gérer.

Après avoir arrêté l’ancien serveur, conservé une copie de data et relancé DEMARRER-WINDOWS.cmd :

1. Dans Musique, recherchez un album, un artiste ou un morceau. L’album et ses morceaux peuvent apparaître séparément, avec leur type et une pochette carrée.
2. Dans Séries, recherchez le titre d’un épisode : le résultat peut proposer la série et l’épisode lui-même, avec une vignette horizontale pour l’épisode.
3. Essayez un titre sans ses accents. Les titres exacts passent avant les résultats trouvés seulement dans les mots-clés.

Les résultats en ligne continuent de venir des fournisseurs actifs. Ce lot extrait l’indexation locale ; il ne rend pas encore toute la recherche distante ou la découverte indépendantes du noyau.

Les catalogues officiels déjà installés passent de 0.1 à 0.2 sans réactiver ceux qui étaient en pause ni réinstaller ceux qui avaient été retirés. Un remplaçant externe choisi reste choisi. Les anciennes extensions sans index de recherche continuent de fonctionner pour l’organisation du catalogue ; seule une recherche simple par nom de fichier reste disponible pour leurs éléments. Une extension peut aussi déclarer explicitement qu’elle ne veut proposer aucun résultat.

Si vous utilisez les familles d’exemple du catalogue, les nouvelles archives 3.0/3.1 contiennent cet index et nécessitent Mosaune 0.0.50. Les archives 2.x restent inchangées. Le numéro de la famille d’exemple est indépendant de celui du serveur.

Pour les contributeurs : le composant catalogue fournit les titres, mots-clés, priorités, libellés et formats des vignettes. Il peut demander l’ouverture du groupe ou du fichier concerné, sans accès supplémentaire aux fichiers, au réseau ou aux comptes. L’exemple build/example-catalogue.mosaune.zip fournit des « Pistes démo » circulaires, retrouvables avec « collection démo ». Son installation et son choix se font dans les composants avancés de Musique, comme les autres remplaçants.

L’index est produit au chargement du catalogue et suit ses droits d’accès. La saisie filtre cet index dans le navigateur, sans charger un moteur WASI à chaque lettre. Aucun rescannage ni déploiement NAS nécessaire.

Le premier chargement doit calculer cet index. Sur le poste Windows de développement, le test synthétique de 1 000 films a pris entre 24 et 37 secondes selon le test, puis entre 70 et 90 ms avec le cache ; ce n’est pas une mesure sur Raspberry Pi. Une modification des métadonnées peut refaire ce calcul. L’optimisation du premier chargement et du cache par portions reste à prévoir avant une validation des performances sur NAS.
