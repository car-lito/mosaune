# Activer les affiches et fiches TMDB

**Depuis la 0.0.7 :** ouvrir **Réglages → Métadonnées → Affiches et fournisseurs** pour trouver cette configuration. MusicBrainz fonctionne sans clé et sert aux albums ; TMDB reste le fournisseur films/séries.

Mosaune fonctionne sans compte : les fichiers restent lisibles, les épisodes sont classés à partir de leur nom et les informations/pochettes musicales sont extraites sur place. TMDB ajoute les affiches, titres, résumés, genres et une première liste d'acteurs pour les films et séries.

## Créer votre accès

Sur votre PC, créez un compte sur [TMDB](https://www.themoviedb.org), puis ouvrez [Paramètres → API](https://www.themoviedb.org/settings/api). Demandez un accès développeur pour votre application personnelle. Le [guide officiel](https://developer.themoviedb.org/docs/getting-started) explique l'inscription et les conditions à accepter.

Pour présenter le projet : nom **Mosaune** ; usage **bibliothèque multimédia personnelle auto-hébergée** ; description **récupération des titres, affiches, résumés et distributions des films et séries d'une bibliothèque locale**. L'application tourne à l'adresse http://127.0.0.1:47831. Si le formulaire exige une URL publique et refuse cette adresse, ne renseignez pas un domaine que vous ne possédez pas : nous traiterons ce point ensemble.

L'API est gratuite pour les usages non commerciaux avec attribution. Mosaune inclut le logo officiel et la mention demandée dans **À propos des métadonnées**. [FAQ TMDB](https://developer.themoviedb.org/docs/faq).

## Configurer Mosaune

1. Dans les paramètres API de TMDB, copiez le **API Read Access Token**, seul, sans le préfixe `Bearer`. La clé API v3 est également acceptée. [Authentification officielle](https://developer.themoviedb.org/docs/authentication-application).
2. Dans Mosaune, dépliez **Affiches et fournisseurs**.
3. Collez la valeur dans **Jeton de lecture ou clé API**, puis cliquez sur **Enregistrer**. Ce message confirme la sauvegarde ; la validité est vérifiée au premier appel au fournisseur.
4. L’identification démarre automatiquement si elle n’est pas en pause. Son avancement et ses commandes sont dans **Réglages → Métadonnées → Affiches et fiches**.

Ne collez pas votre clé dans le chat ou dans le code. Elle est conservée dans la base locale, sans chiffrement au repos ; protégez donc aussi les sauvegardes du catalogue. L'interface ne relit jamais sa valeur après enregistrement. **Déconnecter** retire la clé et conserve les fiches déjà enregistrées.

## Vérifier et corriger

Ouvrez la carte d'une série, puis **Fiche** à côté d'un épisode. Après identification, vous devriez voir les informations disponibles chez TMDB, demandées en français.

En cas de doute, Mosaune garde la fiche locale. Dans **Fiche → Identifier ce média**, recherchez le titre original ou français, puis choisissez le bon résultat. La correction s'applique à ce fichier, pas à toute la série. Elle est conservée lors des analyses suivantes tant que le fichier ne change pas de taille/date. Pour une fiche déjà identifiée, la même action apparaît sous **Corriger la correspondance**.

Depuis 0.0.11, l’identification continue en arrière-plan, un fichier à la fois. La pause et l’avancement sont conservés après redémarrage. Les erreurs sont espacées et limitées à trois essais par fichier, puis relançables. Les correspondances ambiguës attendent une sélection dans la fiche. Aucun nouveau scan du dossier n’est nécessaire pour reprendre les fiches en attente. Voir [ESSAYER-LE-SCAN.md](ESSAYER-LE-SCAN.md).

## Données et limites

Mosaune envoie les titres recherchés, l'année connue, puis les identifiants d'œuvre et numéros d'épisodes nécessaires. Il n'envoie ni chemins de dossiers ni fichiers audio/vidéo. Les requêtes passent par le serveur.

Les réponses sont mises en cache sept jours (plafond de 16 Mio). Les images sont réduites à 640 pixels maximum, dédupliquées et conservées localement (plafond de 128 Mio). Une image invalide, trop grosse ou inaccessible est ignorée ; les informations textuelles restent utilisables. Pour retenter une affiche manquante, refaites une identification dans la fiche.

L'API réelle reste à essayer avec votre clé ; le connecteur a été testé avec des réponses simulées. Cette étape ne corrige pas les pistes audio que le navigateur ne sait pas lire.
