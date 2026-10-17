# Essayer Découvrir — Mosaune 0.0.6

**Guide historique 0.0.6.** Depuis 0.0.7, Découvrir est remplacé par la recherche en haut, Ma liste/Favoris et les suggestions dans les grandes fiches. Lire ESSAYER-LA-REFONTE.md pour le parcours actuel.

## Mettre à jour

1. Arrêter le serveur avec **Ctrl+C** dans sa fenêtre.
2. Copier le dossier **data** ailleurs pour garder un point de retour, pendant que le serveur est arrêté.
3. Double-cliquer sur **DEMARRER-WINDOWS.cmd**, puis recharger la page. Vérifier **v0.0.6** dans la barre latérale.

Le catalogue passe au format 4. Vos dossiers, progressions, métadonnées, configuration fournisseur et playlists sont conservés. Les anciens exécutables refusent ce nouveau format : pour revenir en arrière, restaurer la copie du dossier data correspondante, serveur arrêté.

## Découvrir une œuvre absente

1. Ouvrir **Découvrir** dans la navigation, puis **Rechercher**.
2. Choisir **Films** ou **Séries**, puis entrer un titre, par exemple « Iron Man ».
3. Ouvrir une fiche. **À découvrir** signifie qu’aucun fichier identifié comme cette œuvre n’est présent dans le catalogue disponible.
4. Cliquer sur **Garder dans À voir**, et éventuellement **Ajouter aux favoris**.
5. Fermer la fiche et ouvrir **À voir** ou **Favoris**. Recharger, puis redémarrer le serveur : les œuvres choisies doivent toujours être là.
6. Cliquer à nouveau sur un bouton activé dans la fiche pour retirer l’œuvre de cette liste. Cela ne touche pas aux fichiers.

TMDB doit être configuré pour chercher de nouvelles œuvres ou obtenir des suggestions : voir **CONFIGURER-TMDB.md**, ou le guide dans « Affiches et fournisseurs ». La clé déjà enregistrée reste utilisable. Ne pas la partager dans le chat. Les listes enregistrées sont accessibles sans fournisseur connecté ; une fiche incomplète explique alors pourquoi elle ne peut pas être enrichie.

## Suggestions et œuvres locales

- **Pour vous** propose un point de départ issu des favoris ou des œuvres regardées identifiées, puis de la collection. Choisir une œuvre et cliquer sur **Trouver des suggestions**. À l’ouverture, un favori ou une œuvre regardée peut déclencher directement la première page.
- La raison apparaît au-dessus des résultats : « Parce que vous aimez… », « Parce que vous avez regardé… », ou simplement « À partir de… ». Avoir un fichier ne signifie pas automatiquement l’aimer. Un épisode marqué vu suffit pour utiliser sa série comme point de départ.
- Depuis la bibliothèque, **Fiche → Favoris, À voir et suggestions** ouvre la fiche de l’œuvre. Si ce bouton manque, identifier d’abord le média via son fournisseur.
- **Dans votre bibliothèque** permet de lancer le film. Pour une série, le nombre correspond aux **fichiers d’épisode disponibles** ; la fiche présente saison, épisode et commande Lire/Reprendre. Cela ne promet pas que la série est complète. Deux éditions d’un épisode sont deux fichiers.
- **Précédent / Suivant** change la page des résultats. Le lecteur musical reste monté pendant la navigation.

## Limites de cette étape

Découverte films/séries uniquement, listes mono-utilisateur. Recommandations de proximité fournies par TMDB à partir d’une œuvre ; pas encore de classement mêlant tous vos goûts, de recommandations musicales ni de suivi de personnes. Un fichier non identifié n’est pas rapproché d’une fiche externe par une simple ressemblance de titre. Il peut donc être affiché absent jusqu’à son identification et son actualisation.

Recherche : 20 résultats maximum par page, 50 pages maximum par requête. Cache des fiches non sauvegardées limité à 1 000 œuvres ; une fiche ancienne peut demander de relancer sa recherche. Les favoris et À voir sont préservés. Les très grandes collections et la pagination des listes locales restent à travailler avant la bêta. Images téléchargées à la demande, normalisées et mises en cache ; métadonnées TMDB publiques en cache sept jours. Les listes restent sur votre serveur ; les titres recherchés et les identifiants utilisés pour obtenir fiches/suggestions sont transmis au fournisseur.

Sonarr/Radarr, acquisition automatique et notifications ne sont pas encore raccordés. Le chantier du lecteur vidéo et de ses pistes reste reporté. Cette version reste un prototype local sur PC.
