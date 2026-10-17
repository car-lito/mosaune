# Mosaune 0.0.18 — interface et réglages d’extensions

## Mettre à jour

1. Arrête Mosaune avec **Ctrl+C** dans la fenêtre du serveur.
2. Fais une copie du dossier **data** pour garder une sauvegarde.
3. Relance **DEMARRER-WINDOWS.cmd**, puis recharge la page. Vérifie **v0.0.18** dans la barre latérale sur ordinateur.

La base reste au format 11. Les comptes, médias, affiches, clés, thèmes et préférences existants sont conservés.

## Les deux corrections visuelles

- Les interrupteurs éteints utilisent désormais une couleur de fond plus contrastée issue du thème. La petite bulle reste visible en clair et en sombre. Le créateur peut toujours modifier ce réglage dans son thème.
- Les barres de défilement de l’interface sont masquées, sans barre de remplacement. La molette, le pavé tactile, le clavier et le défilement tactile gardent leur comportement natif. Les flèches des rangées de médias restent disponibles.

Ouvre **Réglages → Extensions** et fais défiler la colonne de gauche, puis le contenu de droite. Les deux zones restent indépendantes. Tu peux aussi vérifier une rangée de films sur l’accueil.

## La suite du moteur d’extensions

Dans **Réglages → Extensions → TMDB**, le panneau de connexion est maintenant généré par Mosaune à partir d’un fichier de description fourni par le module. Ce fichier contient les champs, leurs contraintes et le guide d’installation. L’objectif est que les futurs auteurs puissent ajouter leurs réglages sans devoir fabriquer un écran dans le noyau.

Ta clé existante est réutilisée : tu n’as rien à recopier. Le champ est volontairement vide même quand une clé est enregistrée. **Enregistrer les réglages** sans saisir de nouvelle clé conserve l’ancienne. **Retirer la clé** ouvre une confirmation ; annuler ne change rien. Tu n’as pas besoin de retirer ta vraie clé pour tester la version.

La rubrique **Accès du module** explique les consultations autorisées. Le moteur vérifie désormais que les services utilisés sont à la fois déclarés par le module et autorisés par Mosaune pour la demande en cours. Le module TMDB ne reçoit toujours pas la clé et n’accède pas directement au catalogue.

Cette étape prépare les fournisseurs installables ; elle n’ouvre pas encore l’import de code communautaire ni la boutique GitHub. Films/Séries/Musique restent des fonctions intégrées.
