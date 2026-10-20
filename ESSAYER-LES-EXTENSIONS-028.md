# Essayer les autorisations — 0.0.28

Cette étape poursuit le système d'extensions. Les lecteurs et le découpage des
modules média restent à terminer avant l'installation sur le NAS.

Avec un compte administrateur, ouvrez **Réglages → Extensions → TMDB**, puis
descendez jusqu'à **Accès du module**.

1. Décochez **Autoriser les consultations en ligne de TMDB**, puis cliquez sur
   **Enregistrer les accès**.
2. Faites une recherche depuis Films : TMDB ne doit plus proposer de nouvelles
   réponses en ligne. Les films locaux et leurs fiches déjà enregistrées restent
   disponibles. Les résultats musicaux restent possibles dans une recherche globale :
   ils utilisent un autre fournisseur.
3. Rechargez la page : la case reste décochée. Le choix reste aussi conservé
   après un redémarrage du serveur.
4. Cochez de nouveau la case et enregistrez pour reprendre les consultations.

La clé n'est ni supprimée ni affichée par ce réglage. Les recherches en cours
sont annulées au retrait d'accès et leurs résultats tardifs sont refusés. Les
identifications interrompues restent en attente. Les affiches déjà conservées
restent disponibles ; une image jamais récupérée ne peut plus être téléchargée.

Cet accès est commun au fournisseur intégré et à son paquet TMDB : désinstaller
le seul paquet ou sélectionner un autre moteur ne rétablit pas l'autorisation.
Il est indépendant du bouton d'activation du fournisseur : les deux doivent
permettre la consultation pour que TMDB fonctionne.

TMDB conserve son accès historique lors de cette mise à jour et reste autorisé
par défaut sur une installation neuve. Le contrat générique des extensions, lui,
n'accorde aucun nouvel accès automatiquement. Une mise à jour conserve les
autorisations existantes uniquement pour les opérations toujours demandées.

Si un autre administrateur a modifié les accès, Mosaune refuse d'écraser son
choix et propose **Recharger les accès**. Les utilisateurs ordinaires ne peuvent
pas modifier ces autorisations du serveur.

Le premier raccordement complet concerne TMDB. Les autorisations des autres
familles, les migrations privées et l'extraction Films/Séries/Musique restent
dans la feuille de route. [Contrat pour les auteurs](extension-kit/authorization/README.md).
