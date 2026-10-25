## Informations musicales (0.0.33)

Le [contrat audio-tags](capabilities/audiotags/v1/README.md) et son programme WASI séparent la lecture des tags du noyau. Accès par plages bornées, cycle de vie propre et données conservées. L’import externe de ce rôle reste fermé.

## Reconnaisseurs dans les dépôts (0.0.32)

Les paquets externes sont installables depuis le catalogue GitHub, avec les mêmes contrôles que les fichiers locaux. Le [dépôt d’exemple](examples/recognition/REPOSITORY.md) est produit par la construction et publié sur une branche séparée. [Guide utilisateur](../ESSAYER-LES-EXTENSIONS-032.md). Les autres rôles et familles médias restent à extraire.

## Paquets externes de reconnaissance (0.0.31)

Un [exemple contributeur](examples/recognition/README.md) peut être construit, importé et choisi sans modifier le serveur. Le format local de paquet (schéma 2) sépare les déclarations du manifeste de la politique d’exécution de l’hôte. Installation, activation, sélection, mise à jour, restauration et retrait sont disponibles ; les familles complètes restent à extraire.

## Cycle des reconnaisseurs officiels — 0.0.30

Les trois reconnaisseurs utilisent désormais le registre partagé et un stockage de programmes installés. Ils peuvent être désactivés, retirés puis réinstallés depuis la distribution officielle, avec conservation du catalogue et arrêt des analyses non enregistrées. [Contrat et limites](capabilities/recognition/v1/README.md). L’import externe, les alternatives et les familles médias complètes restent à généraliser.

## Première extraction des fichiers — 0.0.29

Le [contrat media.recognition.v1](capabilities/recognition/v1/README.md) est utilisé par les programmes Films, Séries et Musique pendant le scan réel. Ils sont compilés séparément, sans accès aux fichiers ni au réseau. Livraison embarquée uniquement pour cette étape ; installation des familles média et extraction de leurs écrans encore à réaliser.

## Autorisations persistantes — 0.0.28

Le [contrat des autorisations](authorization/README.md) distingue les demandes du manifeste des opérations accordées par l’hôte. TMDB est le premier adaptateur raccordé à la révocation et à l’interface administrateur ; les autres familles restent à intégrer.

## Réglages persistants — 0.0.27

Le [stockage commun des réglages](configuration/STORAGE.md) isole les propriétaires, masque les secrets dans les réponses et contrôle les modifications concurrentes. TMDB utilise ce stockage pour ses deux moteurs. Ce contrat ne donne pas de lecture directe des clés aux modules invités et n’ajoute pas de chiffrement au repos.

# Laboratoire des extensions Mosaune

## Dépendances partagées — travail du 19 octobre 2026

Le moteur prend désormais en charge les [dépendances par capacités](dependencies/README.md) : choix explicite entre fournisseurs, protection contre le retrait d’un fournisseur utilisé, aperçu des consommateurs et désactivation en cascade. Le [registre durable 0.0.26](registry/README.md) conserve ces choix, restaure les modules avec états réels et raccorde le paquet TMDB à une transaction commune. Les installateurs et l’interface ne sont pas encore généralisés. [Cible lecteurs remplaçables et thèmes](../LECTEURS-ET-EXTENSIONS.md).

## Distribution depuis un dépôt — 0.0.20

[Contrat des dépôts GitHub](DEPOTS-GITHUB.md) : index v1, chemins et empreintes, paquets vérifiés puis remis aux installateurs existants. Catalogue local et dépôts publics ajoutables ; aucune confiance d’exécution accordée par l’ajout d’une source. [Guide utilisateur](../ESSAYER-LE-CATALOGUE-020.md).

**0.0.19 :** [paquets de fournisseurs signés](PAQUETS-FOURNISSEURS.md), contrôle de démarrage versionné et publication indépendante. Deux versions TMDB installables depuis l’interface, remplacement atomique et restauration. Ce pilote accepte une clé de développement reconnue et l’adaptateur TMDB ; autres éditeurs/adaptateurs et budgets globaux restent à construire.


**0.0.18 :** [réglages déclaratifs v1](configuration/README.md), formulaire TMDB généré par Mosaune, secrets gérés par l’hôte. Descripteurs enrichis de services/opérations ; le moteur les intersecte avec les autorisations par invocation. Module TMDB fourni 0.3.0. L’import de fournisseurs exécutables externes reste fermé.


**0.0.17 :** le pilote TMDB 0.2.0 fournit recherche et fiches vidéo riches. Nouvelle capacité facultative dans capabilities/video/v1 ; transport et capacités précédentes conservés. Activation dans Réglages → Extensions → TMDB. Voir [les services hôte](SERVICES-HOTE.md) pour les droits par œuvre et les responsabilités du catalogue. Les images et suggestions spécialisées restent natives.


**15 octobre — intégré au pilote Mosaune 0.0.16 :** le kit fournit aussi un service hôte contrôlé et un programme de recherche TMDB. Il s’active depuis les paramètres du serveur ; [guide utilisateur](../ESSAYER-TMDB-EXTENSION.md), [contrat technique](SERVICES-HOTE.md). La démonstration fictive ci-dessous reste indépendante. Les mesures du 14 octobre concernent uniquement cette démonstration, pas le serveur ni TMDB.

Le laboratoire initial du 14 octobre était distinct du serveur **0.0.15**. Il exécute une petite extension WebAssembly et vérifie le contrat d’échange, le cycle de vie et plusieurs limites de ressources. Depuis 0.0.16, le serveur importe aussi ce kit pour le pilote TMDB, activable dans ses paramètres.

## Essayer sans programmer

Double-clique sur **TESTER-LE-MOTEUR-EXTENSIONS.cmd**, dans le dossier Mosaune. Une fenêtre affiche les étapes réussies et le temps des demandes. Tu n’as aucun fichier à écrire, compte à créer ni clé à fournir. Tu peux laisser ton serveur habituel ouvert.

La démonstration cherche « Voyage », reçoit la fiche fictive « Voyage sous les étoiles », désactive l’extension, la réactive puis la retire du laboratoire. Elle ne lit pas le dossier data de Mosaune, ne se connecte pas au NAS et n’ajoute rien à ta collection. Ce n’est pas un test de téléchargement ou de TMDB.

## Découpage pour les créateurs

| Élément | Responsabilité |
|---|---|
| protocol | Enveloppes de messages, version du protocole, identité de la demande et erreurs |
| capabilities/metadata/v1 | Modèles et validations pour recherche et fiche simple de métadonnées |
| sdk | Aide facultative pour écrire une extension en Go |
| runtime | Chargement et exécution WASI, annulation et limites choisies par l’hôte |
| services | Contrat d’accès aux fonctions hôte explicitement autorisées |
| providers/tmdb/source | Programme de recherche TMDB ; clé et réseau gérés par l’hôte |
| examples/demo/source | Fournisseur fictif, compilé séparément du moteur |
| cmd/lab | Petit hôte qui utilise le fournisseur et vérifie ses résultats |

Le format des messages n’est pas lié à Go ni à un média particulier. L’exemple est écrit en Go ; d’autres langages capables de produire une commande WASI peuvent implémenter le même échange. Nous n’avons pas encore validé un exemple dans chaque langage. Le moteur Go choisi pour ce prototype est [wazero 1.12.0](https://github.com/wazero/wazero/releases/tag/v1.12.0), sous licence Apache 2.0, sans CGO. La compilation Go vers WASI est [documentée officiellement](https://go.dev/wiki/WebAssembly).

## Contrat expérimental

Une invocation reçoit exactement un objet JSON sur l’entrée standard et émet exactement une réponse JSON sur la sortie standard. Les diagnostics utilisent la sortie d’erreur, limitée séparément. Aucun fichier partagé ne sert de canal implicite.

```json
{
  "protocol": 1,
  "id": "demande-42",
  "capability": "metadata.search.v1",
  "operation": "search",
  "payload": { "kind": "films", "title": "Voyage", "language": "fr-FR" }
}
```

```json
{
  "protocol": 1,
  "id": "demande-42",
  "result": {
    "items": [{ "id": "atelier:etoiles", "title": "Voyage sous les étoiles", "year": 2024, "overview": "Une fiche fictive." }]
  }
}
```

Une erreur remplace result par error, par exemple `{"code":"not_found","message":"Aucune fiche."}`. Les deux ne peuvent pas coexister. L’hôte vérifie version, identifiant, format et modèle de résultat avant utilisation. Les messages ne constituent pas une API réseau et ne donnent aucune autorisation utilisateur.

Capacités de l’exemple :

- **metadata.search.v1 / search** : kind, title, year facultatif et language facultatif ; résultat items, tableau de zéro à cinquante candidats.
- **metadata.details.v1 / details** : id opaque ; résultat candidat avec id, title, year et overview.

Le candidat limite les identifiants à 160 octets, les titres à 500, le résumé à 16 000 et l’année à 0–3000 (0 = inconnue). Une recherche n’accepte pas deux fois le même identifiant. Les limites sont vérifiées côté hôte, pas simplement suggérées au créateur. Les modèles de fiches riches existants du serveur ne sont pas encore remplacés par ce petit contrat de laboratoire.

## Évoluer sans tout figer

Le transport ne contient pas une liste fermée de tous les médias ni une gigantesque interface obligatoire. Les capacités sont nommées et versionnées séparément. Une extension peut implémenter la recherche sans proposer des photos, des bandes-annonces ou des recommandations.

Une nouvelle capacité, par exemple `atelier.waveforms.v1`, peut définir ses propres données sans modifier le transport. Pour être utile dans Mosaune, un consommateur doit toutefois savoir l’appeler, vérifier sa réponse et afficher son résultat. Déclarer un nom ne crée pas automatiquement un écran, un droit ou un service dans le noyau.

Les changements compatibles peuvent ajouter des champs facultatifs selon le contrat concerné. Un changement de sens ou de champs obligatoires demande une nouvelle version de capacité. La version du transport reste indépendante. Le SDK facilite l’écriture ; il n’est pas imposé à tous les auteurs.

Pour les extensions plus importantes, on distinguera leurs calculs privés, leurs données privées, leurs contributions d’interface et les données incorporées à un catalogue partagé. Les services communs — accès réseau autorisé, secrets, images, stockage et tâches — devront être fournis par Mosaune via des contrats distincts. Le moteur ne remettra pas sa connexion SQL ou tous les fichiers du serveur à un paquet. Les thèmes gardent leur propre contrat déclaratif ; ils n’ont pas besoin de ce moteur exécutable.

## Limites effectivement présentes

| Ressource | Profil du laboratoire |
|---|---|
| Code Wasm | 8 Mio maximum avant compilation |
| Modules compilés présents | 4 maximum |
| Travail simultané | 1 chargement ou invocation ; appel supplémentaire refusé comme occupé |
| Mémoire linéaire du module | 64 Mio maximum |
| Invocation | Délai de 2 secondes ; annulation lors de désactivation, retrait ou fermeture |
| Compilation | Contexte de 15 secondes ; ce n’est pas une interruption OS garantie du compilateur |
| Message entrant | 64 Kio |
| Réponse / diagnostics | 256 Kio / 8 Kio |

Les limites sont choisies par l’hôte, pas augmentées par le manifeste. Le module commence désactivé. Chaque demande utilise une nouvelle instance ; un compteur global de l’extension ne persiste donc pas entre deux appels. Une mémoire durable nécessitera un service de stockage explicite. Désactiver annule la génération en cours ; une réactivation ne valide pas une ancienne réponse annulée.

Aucun dossier, environnement du processus, argument personnel ou socket n’est fourni au module. Les imports hors WASI sont refusés sauf `mosaune_services_v1.read`, dont chaque service doit être autorisé par l’hôte pour l’invocation. Les imports socket restent refusés. Le laboratoire fictif ne reçoit aucun service ; le pilote TMDB reçoit seulement sa recherche contrôlée. Les tests utilisent des fichiers sentinelles temporaires pour vérifier les refus d’accès.

**La limite de 64 Mio ne borne pas toute la mémoire du processus.** Compilation, code machine, piles, tables et allocations du moteur ont aussi un coût. Un délai d’exécution n’est pas un quota de CPU global. Ce prototype n’est pas encore un installateur de code communautaire inconnu : les budgets du compilateur et du processus, les imports autorisés en détail et les comportements sur Pi doivent encore être évalués. Le lanceur fourni n’accepte que l’exemple embarqué.

## Mesures Windows

Résultats du programme compilé dans [MEASURES-WINDOWS.json](MEASURES-WINDOWS.json), Go 1.27.2, Windows amd64, une exécution locale :

- Extension Go : **4,72 Mio** de Wasm, identique quel que soit le système hôte.
- Préparation du module : **1,147 seconde**.
- Première demande : **7,8 ms** ; médiane des neuf demandes suivantes : **6,8 ms**.
- Pic de mémoire résidente observé pour tout le processus de laboratoire : **104,3 Mio**, relevé toutes les 25 ms ; pas une borne universelle.
- Tas Go après collecte : environ **8,8 Mio** une fois chargé et après les demandes, puis **0,57 Mio** après fermeture. Le runtime Go garde environ **78,2 Mio** réservés ; ces mesures ne signifient pas que toute cette mémoire a immédiatement été rendue au système.

Ces chiffres concernent ce petit exemple local, sans HTTP, images ni lecture de médias. Ils ne prédisent ni TMDB ni les performances du Pi. Les exécutables Linux amd64/arm64 sont compilés, pas exécutés ici. Le serveur 0.0.15 ne charge pas ce laboratoire et son exécutable reste identique.

## Reconstruire et tester

Ce sous-dossier reste un **module Go distinct**, désormais importé par le serveur 0.0.16 via un remplacement local. Ses tests ne sont pas exécutés par le `go test ./...` du dossier parent.

Depuis extension-kit, avec Go installé :

```powershell
./scripts/build-lab.ps1
```

Le script compile les trois modules WASI (démonstration, sonde de tests, TMDB), exécute les tests du kit puis construit les trois laboratoires sans CGO. Les fichiers Wasm fournis localement sont des sorties de compilation ; pour un clone des sources, il faut les reconstruire avant de lancer directement les tests. Le paramètre `-Go` accepte un chemin explicite vers Go. Les versions sont verrouillées dans go.mod/go.sum et les licences sont dans licenses. Pour le serveur complet, utiliser aussi le script parent `scripts/build-app.ps1`.

Le laboratoire accepte `-json` pour produire les mesures brutes. Pour le Pi, utiliser le même module avec le laboratoire Linux arm64 et conserver le rapport, puis mesurer aussi la mémoire résidente avec les outils système.

## Prochain passage vers les vrais fournisseurs

1. Mesurer sur ARM64 et prévoir les limites du processus de travail, dont la compilation de code non fiable.
2. Définir un service réseau contrôlé par l’hôte, avec destinations autorisées, délais, budgets et secrets dédiés ; pas de clé globale donnée à tous les modules.
3. Relier un fournisseur réel au catalogue normalisé, en conservant les corrections manuelles et la provenance des données.
4. Étendre le format de paquet, les permissions, les dépendances et le cycle de mise à jour avant d’ouvrir l’installation dans l’interface.
5. Ajouter des points d’extension d’interface documentés et des capacités supplémentaires selon les besoins réels, puis le catalogue GitHub.

Le pilote 0.0.16 réalise une première partie des étapes 2 et 3 pour la recherche TMDB. Les autres services, les fiches riches et l’installation de fournisseurs restent ouverts.
