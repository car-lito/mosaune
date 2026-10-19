# Extensions — étape 0.0.26

Cette version poursuit le découpage de Mosaune. Elle ne termine pas le système d’extensions et n’ajoute pas encore les lecteurs personnalisés.

L’essentiel est interne : un registre commun conserve les dépendances et les choix de fournisseurs. Le paquet TMDB installé est le premier raccordé à ce registre. L’archive et son enregistrement évoluent ensemble lors d’une installation, mise à jour, restauration ou désinstallation. Les essais automatiques couvrent aussi le redémarrage de plusieurs modules avec une dépendance partagée.

## Ce qui est visible

Si tu veux vérifier l’ajout dans l’interface :

1. Arrête le serveur précédent, lance la version 0.0.26 puis recharge Mosaune.
2. Ouvre **Réglages → Extensions → TMDB**.
3. Si un paquet TMDB est installé, ouvre **Dépendances et démarrage**, sous la présentation du paquet.
4. Ce pilote indique qu’il ne nécessite pas d’autre extension. Le choix du moteur enregistré est également rappelé. Le choisir ne réactive pas un fournisseur TMDB désactivé.

Si aucun paquet n’est installé, le moteur fourni continue de fonctionner comme avant ; tu n’as pas besoin d’en installer un uniquement pour tester cette tranche.

Le premier démarrage sauvegarde un catalogue existant avant le passage au schéma 15. Les fiches, reprises, comptes et réglages sont conservés. Pour revenir au serveur 0.0.25, il faut restaurer la sauvegarde antérieure : cette ancienne version ne sait pas ouvrir le nouveau schéma.

## Ce qui reste à faire

Les choix entre plusieurs fournisseurs et les installations automatiques de dépendances sont préparés dans le moteur, mais leur interface générale n’est pas encore disponible. Films, Séries et Musique font encore partie du programme. Les réglages et autorisations propres à chaque paquet doivent être généralisés avant leur extraction.

Aucun déploiement NAS n’est nécessaire : nous terminerons les extensions et les lecteurs avant de tester une installation neuve sur CasaOS.
