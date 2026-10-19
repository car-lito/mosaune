# Lecteurs remplaçables et thèmes — décisions du 19 octobre 2026

Ce document fixe la cible demandée par l’utilisateur. Il ne décrit pas des lecteurs déjà livrés. Terminer d’abord le système d’extensions, construire ensuite les lecteurs, puis effectuer une installation neuve sur CasaOS.

## Distribution et remplacement

- Films et Séries installent automatiquement une même dépendance officielle de lecture vidéo. Une seule copie par installation ; retirer Films ne retire pas le lecteur encore utilisé par Séries.
- Musique fournit de la même manière son lecteur audio par défaut. Il doit fonctionner sans les modules vidéo.
- Les modules demandent une capacité de lecture, pas une implémentation imposée. Un lecteur alternatif compatible peut être sélectionné, sous réserve des droits du compte et des plateformes annoncées.
- L’installation d’un deuxième lecteur ne détourne pas la lecture en cours. Un changement de fournisseur doit être explicite ; l’interface indique si la session doit être arrêtée/reprise. Aucune promesse de remplacement transparent pendant une lecture.
- Un lecteur utilisé ne disparaît pas silencieusement. Montrer les modules concernés avant de désactiver leurs dépendances ; conserver les données partagées et les progressions. La collecte des paquets devenus inutiles doit être une opération distincte et explicite.
- Les lectures en cours, la file et la progression appartiennent à un service de session commun, avec accès contrôlé. Le moteur et l’interface du lecteur consomment ce contrat ; le catalogue ne dépend pas d’un lecteur particulier.

## Séparer les responsabilités

| Partie | Responsabilité |
|---|---|
| Module média | Catalogue, règles du média, action Lire et dépendance de lecture par défaut |
| Session de lecture | Élément courant, file, état, progression, commandes et événements autorisés |
| Moteur de lecture | Chargement du flux, décodage disponible sur la plateforme, pistes, sous-titres et erreurs |
| Interface du lecteur | Bandeau, vue agrandie, plein écran, commandes et accessibilité |
| Thème | Couleurs, typographie, dimensions, arrondis, états et variations déclarées |

La réutilisation porte d’abord sur les contrats, les états et le comportement. Le navigateur et les futurs clients Android/TV peuvent utiliser des moteurs différents. Ne pas contraindre un moteur vidéo natif à s’exécuter comme un petit fournisseur de métadonnées WASI : les capacités et les modes d’exécution sont distincts.

## Vidéo : références Plex, identité Mosaune

Références utilisateur : première capture du 19 octobre (bandeau en bas pendant la navigation), deuxième capture (lecture agrandie). Elles guident les usages, sans reproduction à l’identique.

- [ ] Bandeau persistant en bas, petit aperçu vidéo en cours, titre et informations utiles, position et durée.
- [ ] Lecture/pause, déplacement dans la lecture, sauts avant/arrière, volume, pistes audio, sous-titres et accès aux réglages adaptés aux possibilités du moteur.
- [ ] Navigation dans Mosaune sans interrompre la vidéo ; clic sur l’aperçu pour ouvrir la vue agrandie.
- [ ] Vue agrandie et plein écran avec commandes lisibles, masquage lorsqu’elles sont inutilisées, retour au bandeau sans recommencer la lecture.
- [ ] Précédent/suivant et file lorsqu’ils ont un sens ; commandes absentes ou indisponibles expliquées selon les capacités réelles.

## Musique : références Spotify, ouverture par la pochette

Références utilisateur : troisième capture pour le bandeau, quatrième et cinquième pour la vue musicale agrandie.

- [ ] Bandeau persistant sur toutes les pages : pochette, titre/artistes, lecture/pause, précédent/suivant, progression, durée, volume, aléatoire, répétition et file.
- [ ] Cliquer la pochette ouvre la vue musicale immersive ; afficher une indication d’agrandissement au survol et au focus clavier. Ne pas imposer un bouton supplémentaire comme seul chemin vers cette vue. Sur écran tactile, toucher la pochette fonctionne directement.
- [ ] Grande pochette, titre et artistes, fond harmonisé avec le thème ; commandes accessibles dans cette vue et retour sans coupure au bandeau.
- [ ] Prévoir des emplacements extensibles pour crédits, file à suivre, paroles ou autres contributions ; ne pas afficher des fonctions fictives en l’absence d’extension fournissant les données.

## Thèmes et accessibilité

Les deux lecteurs utilisent le même système déclaratif que le reste de Mosaune : aucune palette figée dans le lecteur. Définir des réglages documentés pour bandeaux, commandes, curseurs, menus de pistes, états actifs/inactifs, arrière-plans et vues immersives. Les contributions d’un lecteur tiers suivent ce contrat ou déclarent leurs propres réglages nommés avec valeurs de repli.

Le survol ne doit jamais être indispensable : focus clavier visible, libellés accessibles, cibles tactiles utilisables et commandes actionnables au clavier. Distinguer la vue agrandie dans l’application du plein écran du navigateur ; prévoir un repli si ce dernier est indisponible.

## Choix du moteur et compatibilité

L’utilisateur autorise une base open source. Le choix sera fait au chantier lecteurs après vérification des plateformes, licences, maintenance, pistes/sous-titres et contraintes ARM. Une base open source ne garantit pas tous les codecs dans tous les navigateurs. Distinguer conteneur de fichier et codecs, favoriser la lecture directe, puis un changement de conteneur ou une conversion si nécessaire et supportée. Mesurer les coûts avant toute promesse de conversion vidéo sur le Pi 5.

## Vérification de l’architecture avant livraison

- [ ] Films seul, Séries seul, puis les deux : un seul lecteur vidéo par défaut et une seule session active selon la politique retenue.
- [ ] Musique seule, sans installer les composants vidéo.
- [ ] Installer et sélectionner une alternative ; expliquer incompatibilités et dépendances manquantes.
- [ ] Désactiver/retirer un module sans casser les autres ni effacer leurs données.
- [ ] Changer de thème dans les vues normale, réduite et immersive sans couleurs résiduelles.
- [ ] Tester formats/pistes réellement supportés et afficher une erreur utile pour les autres, sans succès simulé.
