# Transport des fournisseurs

## Déclarations des extensions

Depuis la 0.0.57, le manifeste signé TMDB 0.7.0 contient un profil transport v1 et son formulaire settings. Les exemples sont dans extension-kit/providers/tmdb. Le paquet déclare les sources api, images et trailers, leurs limites, la langue et le pays du catalogue. Les secrets sont ajoutés par le serveur au moment de la connexion.

Depuis la 0.0.58, le composant Musique 0.4.0 annonce TransportVersion 1 dans describe-requests et fournit son profil par describe-transport. Son fichier extension-kit/providers/musicbrainz/transport.json est embarqué dans le programme Wasm. Le serveur lit le profil du composant effectivement sélectionné à chaque session de recherche ou d’image, y compris après remplacement et redémarrage.

Chaque source déclare son identifiant, son URL de base, son format, sa taille maximale, son délai, sa cadence et sa durée de cache. Les sources musicales sont musicbrainz, wikidata, commons, coverart, upload et thumb. Le module peut adapter les chemins d’API et réduire les limites dans les bornes autorisées. Les noms de fichiers des images Wikimedia conservent leur encodage. Les redirections Cover Art Archive vers Archive.org sont contrôlées par le serveur.

## Compatibilité et cycle de vie

Les anciens modules musicaux qui annoncent uniquement la version de préparation des recherches conservent la politique de transport historique. Un module annonçant le nouveau contrat doit fournir un profil valide. Une déclaration absente ou invalide dans ce cas bloque son utilisation.

L’import vérifie les contrats de recherche, d’images et de transport avant installation. Les familles 7.0/7.1 nécessitent Mosaune 0.0.58. Les anciennes archives restent disponibles. La migration 38 met à jour uniquement le composant fourni 0.3.0 dont les octets stockés correspondent au registre. Elle conserve son activation et respecte les composants remplacés, modifiés ou supprimés.

La désactivation annule les sessions et leurs connexions en cours. Les fiches et images conservées restent consultables. Le cache et les données sont communs aux composants.

## Responsabilités du serveur

Le service provider_http.go contrôle les destinations avant connexion et à chaque redirection, limite les réponses, applique les cadences interruptibles et valide les données avant stockage. Il fournit le cache partagé et protège les secrets. Les extensions s’exécutent avec les limites Wasm existantes.

Les autorisations actuelles restent attachées aux fournisseurs connus. L’ajout de fournisseurs entièrement nouveaux, l’attribution générique de permissions réseau et les derniers adaptateurs métier restent à extraire.

## Vérification

Les tests de transport musical couvrent un module de remplacement qui change réellement le chemin d’API et la limite de réponse, sa sélection après redémarrage, le rejet des destinations non autorisées, les noms d’images encodés et les migrations avec composants désactivés, supprimés ou modifiés. Les tests des images et recherches couvrent également la révocation des sessions et la conservation du cache.
