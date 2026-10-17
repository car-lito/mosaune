# Essayer les thèmes de Mosaune 0.0.15

Cette version remplace le petit habillage de neuf couleurs par un fichier de thème commun à l’interface : couleurs, polices, tailles, arrondis et états des composants. Les paramètres utilisent maintenant le thème choisi. Une rubrique ne change d’apparence que si l’auteur a déclaré une exception.

## Installer la version

1. Arrête le serveur avec **Ctrl+C** dans sa fenêtre.
2. Garde une copie du dossier **data**, puis relance **DEMARRER-WINDOWS.cmd** et recharge la page.
3. Vérifie **v0.0.15** en bas du menu. Tu conserves comptes, bibliothèques et réglages. Le format de base reste 10 ; aucune nouvelle migration depuis 0.0.14.

## Mettre à jour Ambre

1. Ouvre **Réglages → Extensions**, avec ton compte administrateur.
2. Télécharge l’exemple Ambre. Le serveur fournit maintenant **Ambre 2.0.0**. Le même ZIP se trouve dans [examples/ambre-v2](examples/ambre-v2/atelier-ambre-2.0.0.mosaune.zip).
3. Sélectionne ce ZIP dans **Paquet à vérifier**, puis installe la mise à jour (ou le paquet si tu n’avais pas Ambre).
4. Clique **Activer**, puis **Utiliser pour moi** si nécessaire. Une mise à jour conserve les autorisations et le choix des comptes, mais attend une réactivation.
5. Passe de Films à Musique puis aux Réglages : le menu doit rester brun. Teste aussi le mode sombre dans **Mon compte**.

Observe les boutons, champs, badges « Chez vous » / « À découvrir », fenêtres de correction et la file d’écoute. Leurs couleurs et arrondis viennent également du fichier. La lecture en cours ne redémarre pas lorsqu’on change de thème.

Les anciens paquets v1 restent acceptés. Ils utilisent les nouveaux réglages par défaut pour les éléments qu’ils ne savent pas personnaliser.

## Un thème commun, avec des exceptions choisies

L’exemple Ambre utilise les mêmes réglages dans toutes les rubriques. Un auteur peut cependant demander un menu violet uniquement dans Musique, par exemple. Les éléments non précisés héritent du thème commun ; sortir de Musique les rétablit automatiquement.

Dans Extensions, l’administrateur peut télécharger le **modèle JSON complet** et le **catalogue des réglages**. Le guide [CREER-UNE-EXTENSION.md](CREER-UNE-EXTENSION.md) explique leur rôle, avec un exemple d’exception.

Pour revenir à Mosaune, utilise **Rétablir pour moi**. Désactiver Ambre garde le paquet et les préférences ; le désinstaller retire ses références sans toucher aux médias. **DEMARRER-SECOURS-WINDOWS.cmd** reste disponible pour démarrer sans thème externe. La connexion garde le thème Mosaune, avant de connaître le compte.

Cette étape concerne l’apparence web. Déplacer librement les blocs, ajouter des menus, installer des langues ou exécuter des fournisseurs externes demandera d’autres points d’extension ; ce n’est pas annoncé comme livré ici.
