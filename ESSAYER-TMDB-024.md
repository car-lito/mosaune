# TMDB : un véritable interrupteur

La version 0.0.24 sépare deux réglages : **Activer TMDB** et le **moteur de traitement**. Le premier coupe le fournisseur ; le second choisit comment il fonctionne quand il est actif.

## Vérification simple

1. Lancez la nouvelle version et ouvrez **Réglages → Extensions → Sur le serveur** avec le compte administrateur.
2. Coupez l’interrupteur de la carte **TMDB**. La carte indique **Désactivé**.
3. Dans **Films**, recherchez un titre : la recherche dans votre collection fonctionne encore, mais TMDB ne propose plus de nouveaux résultats en ligne. Un message explique pourquoi.
4. Ouvrez un film déjà identifié : sa fiche et ses affiches conservées restent disponibles. Les images jamais récupérées ne sont pas téléchargées. Les suggestions TMDB et les lignes de sorties en ligne sont suspendues.
5. Fermez puis relancez Mosaune : TMDB reste désactivé. Réactivez-le pour retrouver la recherche en ligne et permettre la reprise des fiches en attente, si l’identification n’est pas mise en pause et si sa clé est configurée.

MusicBrainz continue à fournir les résultats musicaux. Vos favoris et votre liste peuvent toujours contenir des films absents du serveur que vous aviez déjà enregistrés : ces fiches conservées ne sont pas de nouvelles recherches TMDB.

## Un choix de moteur séparé

Ouvrez la carte TMDB, puis **Moteur de traitement · avancé** pour choisir le moteur intégré ou le moteur extension. Changer ce choix, installer un paquet, le retirer ou utiliser le mode de secours ne réactive pas un fournisseur désactivé.

Les requêtes déjà en cours sont annulées. Une réponse arrivée tardivement est ignorée et ne peut pas enrichir une fiche après la désactivation. Un arrêt volontaire ne compte pas comme un échec d’identification.

## Lors de la mise à jour

TMDB reste initialement actif, car les versions précédentes utilisaient effectivement TMDB même lorsque le moteur extension était coupé. **Coupez une première fois le nouvel interrupteur si vous voulez arrêter le fournisseur.** Votre clé, vos fiches, vos corrections et vos préférences sont conservées.

La base passe au format 14, avec une sauvegarde préalable. Pour revenir à un ancien programme, utilisez sa sauvegarde compatible ; l’ancienne version ne sait pas ouvrir le nouveau format. Les médias ne sont pas modifiés.
