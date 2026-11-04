# Transport des fournisseurs

**Depuis la 0.0.57**, le manifeste signé TMDB 0.7.0 contient un profil `transport` v1 et son formulaire `settings`. Les exemples sources sont `extension-kit/providers/tmdb/transport.json` et `transport-settings.json` ; l’outil `package-provider -transport` les inclut dans le paquet. Les sources nommées `api`, `images` et `trailers` déclarent leur URL de base, format, taille maximale, délai, cadence et durée de cache. Seule la source API peut demander le secret. L’installateur vérifie les destinations contre les autorisations existantes ; le module ne reçoit jamais le secret. Une absence de déclaration sur un ancien paquet conserve son comportement historique.

Le schéma de transport est réutilisable ; les adaptateurs de fournisseur et les permissions de nouvelles origines restent à extraire. MusicBrainz utilise encore la politique du serveur décrite ci-dessous.

Les programmes d’extension définissent les recherches et interprètent les réponses. Le noyau garde la responsabilité des connexions, des secrets et du stockage. TMDB, MusicBrainz, Cover Art Archive et les sources de portraits passent désormais par `fetchProviderHTTP`, dans `provider_http.go`.

## Ce que le service garantit

- Une destination doit être autorisée avant toute lecture du cache ou connexion. Chaque redirection repasse par la politique du client et la liste des destinations. Sans politique explicite, aucune redirection n’est suivie.
- Une attente de cadence peut être annulée pendant la file comme pendant le délai. Les fournisseurs gardent des cadences séparées.
- Les réponses sont bornées et validées avant d’être conservées. JSON TMDB/MusicBrainz : 2 Mio ; JSON des portraits : 512 Kio ; images : 8 Mio. Les limites d’exécution Wasm ne changent pas.
- Le cache commun reste séparé par identifiant de fournisseur et clé publique. Il conserve ses durées existantes et son plafond de 16 Mio ; les images persistantes sont protégées du nettoyage des réponses expirées.
- La désactivation en cours d’opération est vérifiée avant la sauvegarde, sous le verrou du cycle de vie concerné. Une réponse annulée ne devient pas une nouvelle entrée de cache.
- La clé ou le jeton TMDB est ajouté à une requête indépendante, après le calcul de la clé publique du cache. Les paramètres reçus ne sont jamais modifiés. Les erreurs réseau ne révèlent pas l’URL authentifiée.

## Ce qui reste à découper

`providerHTTPCall` est une interface **interne de l’hôte**, pas un service librement invocable par un programme Wasm. Les origines, clients, limites, cadence, fonctions d’authentification et espaces de cache sont encore choisis par les adaptateurs de confiance du serveur. Les programmes externes ne peuvent pas fournir leur propre politique d’autorisation.

Les fichiers `provider_tmdb.go`, `provider_musicbrainz.go` et `provider_musicbrainz_visual.go` relient les contrats existants à ce service. La suite doit distribuer leur configuration et leurs déclarations de sources avec les paquets, en les reliant aux autorisations communes. Ajouter un fournisseur entièrement nouveau demande encore du travail dans le serveur : cette étape ne prétend pas avoir terminé cette indépendance.

Aucune nouvelle carte d’extension, migration de données ou version de famille n’est nécessaire pour ce changement interne. Les familles 6.0/6.1 et le composant musical 0.3 restent compatibles.

## Vérification

`provider_http_test.go` couvre le cache, les frontières d’origine, les redirections, la révocation, l’annulation des attentes et la confidentialité des clés. Les tests existants TMDB, musique et images vérifient les parcours complets avec des réponses simulées, sans clé personnelle ni accès au NAS.

Il n’y a pas de changement visuel à tester pour cette étape.
