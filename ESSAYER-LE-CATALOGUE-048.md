# Familles depuis le catalogue — 0.0.48

Plus besoin de télécharger puis de sélectionner un ZIP : Mosaune récupère l’archive choisie depuis GitHub, vérifie son contenu, puis présente le même aperçu que l’import local.

## Essayer

1. Lancez Mosaune **0.0.48**, puis ouvrez **Réglages → Extensions → À télécharger**.
2. Cliquez sur **Actualiser le catalogue officiel** (ou **Charger le catalogue officiel**).
3. Dans **Type de paquet**, choisissez **Familles médias**. Ouvrez **Famille Films · exemple**.
4. Pour essayer une mise à jour, choisissez d’abord **2.0.0**, puis **Vérifier ce paquet**. Lisez les changements prévus et cliquez sur **Installer ce paquet**.
5. La fiche Films s’ouvre. Les nouveaux composants restent en pause : vous pouvez activer la famille depuis cette fiche.
6. Revenez au catalogue, sélectionnez **2.1.0**, vérifiez puis installez la mise à jour. Vos choix, états et autorisations existants sont conservés.

Si vous avez déjà installé les exemples 1.x en 0.0.47, leur version est reconnue et les exemples 2.x sont proposés comme mise à jour. Une version déjà installée ou plus ancienne n’est pas proposée à l’installation depuis le catalogue. Pour réparer une même archive après un retrait partiel, l’import local reste disponible.

Les exemples Séries et Musique sont également proposés. Ils regroupent les **composants déjà extraits et leur écran**, sans ajout de lecteur et sans prétendre que toutes les fonctions métier sont sorties du noyau.

## Ce qui est préservé

- Actualiser le catalogue ne met rien à jour automatiquement.
- Les composants partagés présents et les remplaçants explicitement choisis sont conservés.
- Une erreur de téléchargement ou une archive différente de l’empreinte annoncée bloque l’installation.
- Un aperçu périmé est refusé : vérifiez de nouveau avant de confirmer.
- Retirer un dépôt personnalisé conserve les familles installées.
- Les médias, comptes et clés ne sont pas envoyés au dépôt GitHub. Seuls le catalogue et les archives demandés sont téléchargés.

L’ancien catalogue officiel reste publié pour les versions précédentes. Mosaune 0.0.48 utilise un index distinct du même dépôt. Le mot « officiel » identifie la source ; les exemples de familles ne sont pas signés et l’éditeur déclaré n’est pas authentifié par une signature.

Restent notamment la restauration globale d’une famille, les dépendances entre familles, les contributions cumulables et l’extraction des services métier. Aucun déploiement NAS dans cette étape.
