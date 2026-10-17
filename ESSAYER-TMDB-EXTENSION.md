# Essayer TMDB par extension — 0.0.17

**Version actuelle 0.0.19 :** le fournisseur TMDB peut maintenant être installé depuis un paquet, mis à jour et restauré sans remplacer le serveur. [Suivre ce nouveau guide](ESSAYER-LES-FOURNISSEURS-019.md). La connexion et les fiches existantes sont conservées.


**Version actuelle 0.0.18 :** la fiche Extensions → TMDB contient désormais un formulaire construit depuis la description du module. La clé existante est conservée ; laisser son champ vide la conserve aussi lors d’une sauvegarde. Retirer la clé est une action distincte avec confirmation. [Guide de mise à jour](ESSAYER-LA-018.md). Les étapes 0.0.17 ci-dessous décrivent le pilote précédent.


Le moteur traite maintenant la recherche et les fiches détaillées des films, séries et épisodes. Le résultat rejoint le catalogue habituel de Mosaune.

## Mettre à jour et essayer

1. Arrêtez Mosaune, sauvegardez **data**, relancez **DEMARRER-WINDOWS.cmd**, puis rechargez le navigateur. Vérifiez **0.0.17**.
2. Avec un compte administrateur, ouvrez **Réglages → Extensions**, puis la carte **TMDB**.
3. Activez le module. La clé déjà configurée est réutilisée ; le panneau de connexion est aussi disponible dans cette fiche.
4. Ouvrez un film ou une série puis **Corriger l’association** et recherchez son titre. Les propositions passent par le module. Annulez si vous ne souhaitez rien changer.
5. Sur une fiche à corriger, sélectionnez la bonne correspondance : les informations détaillées passent aussi par le module. Vérifiez résumé, acteurs, saisons et bandes-annonces selon ce que TMDB fournit.
6. Revenez à la carte TMDB et désactivez le module : Mosaune retrouve le fournisseur intégré. Les fiches et corrections restent conservées.

L’activation ne réidentifie pas toute la collection et ne force pas le renouvellement des fiches déjà enregistrées. Le réglage est partagé par le serveur et conservé au redémarrage. Le module est désactivé par défaut. Sa première utilisation charge le moteur et peut prendre un peu plus de temps.

## Limites de cette étape

La base reste au format 11 de la 0.0.16. Une installation plus ancienne est migrée avec sauvegarde SQLite préalable ; gardez votre copie de data pour revenir en arrière.

Le serveur conserve la clé, le cache, les fichiers images et le catalogue. Le module reçoit uniquement les réponses des consultations autorisées pour l’œuvre demandée. La langue actuelle reste fr-FR ; la préférence multilingue par compte n’est pas encore livrée.

Les suggestions spécialisées restent dans le fournisseur intégré. Les fournisseurs exécutables externes, leurs mises à jour et la boutique GitHub ne sont pas encore installables. La consommation sur Pi reste à mesurer ; la limite de mémoire de l’instance Wasm ne borne pas toute la mémoire du serveur.

Si nécessaire, désactivez le module. Si les réglages sont inaccessibles, arrêtez le serveur et lancez **DEMARRER-SECOURS-WINDOWS.cmd** : les thèmes externes et le module TMDB sont neutralisés sans effacer les préférences.
