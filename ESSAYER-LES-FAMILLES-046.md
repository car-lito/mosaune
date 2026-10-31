# Essayer les familles d’extensions — 0.0.46

Dans **Réglages → Extensions → Sur le serveur**, Films, Séries et Musique regroupent désormais leurs composants fournis et leur écran. Les fournisseurs, connecteurs, thèmes et compléments restent distincts.

## Un essai simple

1. Ouvrez **Films**. Son état indique si les composants sont prêts, désactivés ou incomplets.
2. Cliquez sur **Désactiver la famille**. L’aperçu indique les programmes concernés et les composants partagés conservés. Confirmez : la rubrique Films disparaît du menu, sans supprimer vos fichiers ni vos fiches. Séries et Musique restent disponibles.
3. Revenez dans **Réglages → Extensions → Films**, puis choisissez **Installer / activer la famille**. Vérifiez l’aperçu et confirmez : l’écran et les composants choisis reviennent ensemble.
4. Ouvrez **Composants et remplacements · réglages avancés** pour retrouver les réglages individuels, les remplaçants et les accès de l’écran.

**Retirer la famille** désinstalle les composants exclusifs choisis et son écran. Un remplaçant externe sélectionné fait aussi partie du retrait : il faudra le réimporter. Les composants partagés et les alternatives non sélectionnées restent installés. Les données de collection sont conservées. Un redémarrage ne réinstalle pas la famille retirée.

## Catalogue et remplacements

L’onglet **À télécharger** propose les trois familles. Les composants techniques fournis ne multiplient plus les cartes par défaut ; cochez **Afficher les composants techniques des familles** pour les retrouver dans les dépôts, notamment pour une mise à jour individuelle.

Activer explicitement un écran alternatif le choisit aussi pour sa famille. Le regroupement respecte ce choix et ses permissions. S’il a été désinstallé, Mosaune demande de le réinstaller ou de choisir explicitement un autre écran ; il ne le remplace pas en silence. Un écran servant plusieurs rubriques ne peut pas être retiré par le bouton d’une seule famille.

## Ce que cette étape ne termine pas

Cette version regroupe réellement les opérations des composants **déjà extraits** et des écrans, avec une seule validation en base. Elle ne constitue pas encore trois extensions médias totalement indépendantes : recherche, profils, associations, playlists et plusieurs services restent à extraire. Masquer une rubrique n’interdit pas à lui seul les autres accès autorisés aux médias.

Les manifestes de regroupement fournis sont versionnés, mais l’import d’une archive de famille complète, ses mises à jour coordonnées et les contributions cumulables restent à construire. Les composants partagés restent conservés même si toutes les familles sont retirées, car des fonctions du noyau les utilisent encore. Aucun nouveau lecteur ni déploiement NAS dans cette étape.
