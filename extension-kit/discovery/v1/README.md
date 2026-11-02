# Surface de découverte v1 — Mosaune 0.0.51

Une déclaration de données, sans HTML, JavaScript, URL de service, clé API ou accès à la base. Elle annonce les recherches et sélections d’un fournisseur. Elle ne crée pas l’implémentation de ces fonctions ni leurs permissions.

Exemple :

```json
{"version":1,"search":[{"kind":"films","label":"Cinéma","placeholder":"Rechercher une œuvre…","similar":true}],"shelves":[{"id":"festival","kind":"films","label":"Les films du festival","selector":"kind"}]}
```

search accepte au plus 16 entrées uniques par type, avec libellé de 80 octets et texte de recherche de 200 octets. shelves accepte au plus 32 couples type/identifiant uniques, avec libellé de 120 octets. Textes UTF-8 sans contrôles ; identifiants bornés à 80 caractères ASCII (lettre minuscule initiale, puis lettres/chiffres/point/tiret/soulignement). Les types déclarés doivent appartenir à ceux du fournisseur. Le noyau reste actuellement limité à Films/Séries/Musique.

similar annonce l’acceptation d’une œuvre de référence, résolue et contrôlée par le serveur. selector vaut kind (l’adaptateur reçoit le type) ou contributors (il reçoit un identifiant de contributeur). Le client choisit au plus trois identifiants distincts dans les œuvres visibles, locales ou favorites, du même fournisseur et du même type. La déclaration n’autorise ni URL ni requête HTTP libre.

Les recherches interrogent jusqu’à six couples fournisseur/type actifs, configurés et compatibles ; les échecs sont isolés par source. Les résultats sont dédoublonnés par leurs identités existantes. Une sélection inconnue est refusée côté serveur avant d’appeler le fournisseur, même si le client forge sa requête. Les droits de compte, l’état des modules, les secrets et l’annulation des appels restent gérés par les services existants.

Un écran ui.pages.v1 peut employer un bloc feed avec source remote, provider et shelf nommés :

```json
{"type":"feed","source":"remote","provider":"org.example.source","shelf":"festival","view":"films","title":"Festival","display":"shelf"}
```

Pour un nouvel identifiant de source/sélection, annoncez minCore 0.0.51 dans le paquet écran. Installer cet écran ne crée pas sa source : l’adaptateur actif doit déclarer la sélection. Une source absente, désactivée ou non configurée ne provoque aucun appel depuis le bloc.

Compatibilité : un adaptateur sans interface DiscoverySurfaceProvider conserve sa recherche legacy s’il implémente DiscoveryProvider ; ses sélections nécessitent une déclaration. Une déclaration explicite vide interdit les opérations correspondantes ; une déclaration invalide ne déclenche jamais de repli legacy.

Depuis Mosaune 0.0.52, le manifeste signé TMDB accepte un champ optionnel discovery contenant cette Surface. L'outil package-provider prend -discovery chemin.json et fixe minCore à 0.0.52. TMDB 0.6.0 l'utilise ; la signature couvre la déclaration comme le reste du manifeste. Le fichier JSON de départ reste dans le dossier du fournisseur.

Le serveur privilégie DiscoverySurfaceContext(ctx) pour les adaptateurs qui chargent leur déclaration depuis un paquet installé, puis l'interface historique DiscoverySurfaceProvider pour les autres. TMDB relit l'empreinte persistée et ne garde en cache que la petite déclaration validée. Le contrôle des opérations est répété sous le verrou du moteur afin qu'une mise à jour ne puisse pas changer la déclaration entre son contrôle et l'appel Wasm.

Pour l'adaptateur tmdb.v1 actuel, les sélections restent limitées aux routes qu'il implémente : now_playing/upcoming pour films et popular pour series, avec selector kind. Une déclaration peut réduire ces possibilités et changer les libellés, sans élargir ses permissions réseau. discovery absent dans un ancien paquet doté de metadata.video-catalog.v1 conserve la surface historique ; discovery présent mais vide interdit toute recherche et sélection. L'absence du paquet ne réactive rien. Les métadonnées déjà stockées restent indépendantes de ces déclarations.

État de migration : MusicBrainz garde encore sa déclaration dans le serveur. Le transport WASI générique des adaptateurs et leur cycle de vie complet restent à finaliser. Ce format ne permet pas encore d'installer un fournisseur arbitraire ni un nouveau type de média.
