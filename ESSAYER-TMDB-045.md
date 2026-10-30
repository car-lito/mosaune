# TMDB devient un paquet installé — 0.0.45

Au premier démarrage, Mosaune installe le paquet signé TMDB 0.5.0 disponible dans le catalogue fourni. Son programme est chargé uniquement à la demande, depuis le paquet installé. Le serveur ne possède plus de chemin d’exécution vers un module de secours embarqué.

## Essai simple

1. Relancer Mosaune avec DEMARRER-WINDOWS.cmd, puis ouvrir **Réglages → Extensions → TMDB**.
2. Le panneau **Paquet installé · 0.5.0** propose **Désinstaller le paquet**. Un paquet plus ancien déjà installé garde sa version.
3. Après confirmation, le panneau affiche **TMDB non installé**. L’activation du module est bloquée : elle ne réinstalle rien.
4. Redémarrer Mosaune : TMDB reste absent. Les médias et les fiches déjà enregistrées restent présents. Les autres fournisseurs peuvent toujours proposer leurs propres résultats.
5. Choisir **Voir les paquets disponibles**, puis TMDB 0.5.0 dans le catalogue **Inclus avec Mosaune** ou le dépôt officiel ; vérifier et installer le paquet.
6. La réinstallation reste en pause. Activer explicitement le module dans **Module de traitement**. Le fournisseur et son accès en ligne doivent également être autorisés pour reprendre les consultations.

L’essai est facultatif : les tests automatiques utilisent des bases temporaires, sans vos médias ni votre clé TMDB.

## Passage depuis une ancienne version

- Un paquet déjà installé, sa version et sa version de retour sont conservés.
- L’ancien module fourni actif devient une installation du paquet signé, une seule fois.
- Si aucun paquet n’était installé et que le module était en pause dans la 0.0.44, Mosaune le laisse absent : cet état pouvait aussi correspondre à une désinstallation. Il faudra le réinstaller explicitement si vous le souhaitez.
- Désactivation du fournisseur, refus d’accès réseau et réglages de connexion sont conservés.
- Retirer un paquet ne supprime ni les fiches, ni les affiches conservées, ni les corrections, ni les médias.

## Limite du découpage actuel

Le catalogue inclus conserve les archives installables pour permettre une réinstallation hors ligne ; elles ne sont jamais exécutées directement. TMDB n’est pas encore un fournisseur entièrement indépendant du noyau : son adaptateur réseau, ses réglages et la correspondance d’identifiants pour Sonarr restent à extraire. Ces consommateurs expliquent la conservation des réglages partagés après retrait. Aucun nouveau lecteur ni déploiement NAS dans cette étape.
