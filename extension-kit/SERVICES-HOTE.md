# Services hôte v1 et pilote TMDB

## Déclarations obligatoires — Mosaune 0.0.18

Le descripteur comprend maintenant services (nom versionné et opérations). Au chargement, le moteur en prend une copie ; à chaque invocation il intersecte cette liste avec les grants fournis par l’hôte. Une déclaration seule n’accorde rien ; un service absent de la déclaration ne s’exécute jamais, même proposé par l’hôte. Le pilote TMDB 0.3.0 déclare tmdb.api.v1/get. Les politiques exactes par œuvre/recherche décrites ci-dessous restent inchangées.

Les anciens descripteurs sans services restent valides, avec zéro service accessible. Les programmes du laboratoire qui appellent un service doivent le déclarer. Ce contrat expérimental concerne les modules fournis ; il ne constitue pas encore un manifeste installable de politique réseau. [Réglages déclaratifs et secrets](configuration/README.md).


## Extension du pilote dans Mosaune 0.0.17

Le module TMDB 0.2.0 conserve son identifiant mosaune.tmdb-search et annonce aussi la capacité facultative **metadata.video-details.v1**. Le contrat séparé capabilities/video/v1 contient Selection et Details ; les identifiants sont opaques, les tableaux et textes bornés. Aucune écriture SQL, provenance ou correction manuelle n’est acceptée du programme.

Pour une fiche, provider_tmdb_extension_details.go autorise une fois chacune : la route principale movie/id ou tv/id, sa route videos facultative, et uniquement l’épisode sélectionné si demandé. L’hôte fixe et compare langue, append_to_response et include_video_language. Une autre œuvre, saison, URL, requête ou un en-tête arbitraire est refusé. Le budget général reste quatre appels, mais ces droits n’ouvrent que trois routes au maximum.

Le programme normalise acteurs, saisons, épisode, genres, durée, note et bandes-annonces. La langue demandée passe en premier ; la consultation vidéo de repli est facultative. Une erreur de l’épisode demandé fait échouer la fiche. L’hôte vérifie à nouveau les données, les identifiants TMDB, les références d’images et les liens YouTube, puis construit les URL d’assets et attribue la provenance. Les octets des images et les suggestions spécialisées restent natifs.

Recherche et détails partagent le même moteur, verrou et cycle de désactivation. Pas de migration de catalogue ni de réidentification globale. La gestion de paquets exécutables autonomes reste à faire. Les sections 0.0.16 ci-dessous décrivent le premier pilote de recherche.


État du 15 octobre 2026. Contrat expérimental, partagé entre le moteur et le SDK ; pas encore ouvert aux paquets exécutables installés par l’utilisateur.

## Une autorisation attachée à une invocation

`Engine.Call` ne donne aucun service. `Engine.CallWithServices` reçoit une table de fonctions hôte autorisées pour cette seule demande. Cette table est copiée ; le nom d’un service dans le programme ou le manifeste ne confère aucune permission.

Le programme peut importer `mosaune_services_v1.read(inputPointer, inputLength, outputPointer, outputCapacity) -> i32`. Les pointeurs sont des adresses dans sa propre mémoire linéaire. Le résultat positif indique le nombre d’octets écrits ; zéro indique un refus ou une erreur de transport. Les autres imports non WASI restent refusés, ainsi que les sockets WASI. Les langages autres que Go peuvent reproduire cette ABI ; seul le SDK Go est testé ici.

Requête JSON :

```json
{"service":"tmdb.api.v1","operation":"get","params":{"path":"/search/movie","query":{"query":"Voyage","include_adult":"false","language":"fr-FR"}}}
```

Réponse JSON : `{"data":{...}}` ou `{"error":"service_denied"}`. Les autres erreurs génériques sont `service_failed` et `service_invalid_response`. Les erreurs internes du service ne passent pas dans la mémoire de l’extension : elles peuvent contenir des secrets. L’adaptateur applicatif peut présenter à l’utilisateur une erreur contrôlée du fournisseur.

Limites générales : requête 8 Kio, corps de résultat 2 Mio, enveloppe 2 Mio + 1 Kio, quatre appels au maximum par invocation. Les appels invalides consomment le budget. Les plages mémoire sont vérifiées avant lecture et écriture. Les réponses JSON sont validées. Désactivation et délai annulent également le contexte du service. Chaque service hôte est du code de confiance : il doit respecter le contexte, borner ses opérations et valider ses paramètres ; le moteur ne peut interrompre arbitrairement une fonction Go qui ignore le contexte.

## Adaptateur TMDB livré dans Mosaune 0.0.16

Le programme `providers/tmdb/source` choisit recherche film/série et normalise les résultats. Il demande `tmdb.api.v1 / get`. L’adaptateur `provider_tmdb_extension.go` autorise **une seule consultation**, exactement pour le type, titre, année et langue de la recherche en cours. Il compare la route et tous les paramètres ; aucune URL complète, en-tête, autre route, clé ou requête différente n’est accepté.

Le service utilise le client TMDB existant de Mosaune : destination fixée à api.themoviedb.org, redirections refusées, clé injectée côté hôte, cache commun avant injection du secret, temporisation entre appels, délai HTTP et réponse bornée. Les tests utilisent un faux transport, sans TMDB ni NAS réels. La langue reste celle de l’application (actuellement fr-FR), pas encore une préférence multilingue par compte.

L’extension renvoie `metadata.search.v1` : candidats avec identifiant, titre, titre original facultatif, année et résumé. L’hôte valide ce modèle puis les identifiants TMDB numériques. Le contrat partagé garde des identifiants opaques pour les autres fournisseurs. La provenance reste `tmdb`, ce qui préserve les associations existantes. Le programme ne possède aucune base ; l’écriture des fiches riches et des corrections reste dans le catalogue.

Le moteur applicatif est chargé à la demande, limité à un module, une exécution à la fois, 64 Mio de mémoire linéaire et quinze secondes par invocation réseau. Les autres limites du laboratoire s’appliquent. Ce n’est pas une limite globale de RAM du processus. La préférence est administrateur, persistante, désactivée par défaut et neutralisée par le mode de secours. La fermeture du moteur annule ses travaux en cours. Les détails/images restent dans le fournisseur natif, sans repli silencieux après une recherche WASM échouée.

## Évolution

Le pont de services n’est pas propre aux métadonnées : d’autres services nommés et versionnés pourront fournir stockage dédié, actions ou ressources, avec leurs propres politiques. Un nouveau fournisseur demandera une politique réseau explicite, des droits et réglages déclarés, pas l’ajout de domaines libres à la requête. Il reste à concevoir ces politiques dans les manifestes installables, le cycle de mise à jour, les budgets de compilation/processus et les validations ARM. Le présent pilote ne prétend pas fournir cette gestion complète.
