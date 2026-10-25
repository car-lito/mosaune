# media.audio-tags.v1

Contrat versionné de lecture des informations embarquées. Implémentation officielle : `modules/audiotags/source`, programme WASI séparé utilisant dhowden/tag. Le module de reconnaissance musicale des noms reste indépendant.

## Invocation

Capacité `media.audio-tags.v1`, opération `extract`, charge JSON `{"size":12345}`. Aucun nom, chemin ni identifiant de compte. Réponse : `matched`, puis `title`, `artist`, `album`, `albumArtist`, `genre`, `year`, `track`, `disc`, `picture` (octets encodés en base64 JSON). La structure et les limites sont définies dans contract.go et vérifiées des deux côtés.

Un fichier sans tags reconnus renvoie `{"matched":false}` et garde le nom fourni par le reconnaisseur. Une défaillance du moteur, du service ou du protocole interrompt le scan avant toute écriture du catalogue. Une corruption de tags que le parseur peut traiter renvoie simplement aucune correspondance. Le noyau conserve la propriété et la validation des données, de leur provenance et des images décodables.

## Service accordé par l’hôte

Le manifeste demande `media.file.read.v1`, opération `range`. Paramètres : `{"offset":0,"length":65536}`. Réponse : `{"data":"base64"}`. L’hôte fournit ce service uniquement pendant l’analyse du fichier qu’il a déjà ouvert. Aucun autre chemin, répertoire, URL, secret ou environnement n’est accessible au programme. Les champs inconnus sont rejetés.

Limites : 64 Kio par bloc, 16 Mio lus par fichier, 512 appels hôte maximum, 20 000 opérations de lecture/déplacement dans l’adaptateur officiel, pochette brute de 8 Mio maximum. Le moteur est créé pour un scan musical puis fermé ; aucun moteur de tags ne reste en mémoire au repos. Budget mémoire linéaire WASM de 128 Mio et délai de 10 secondes par fichier : ce n’est pas un plafond de RAM totale du processus. La sortie est bornée à 12 Mio pour l’encodage JSON d’une pochette. Les autres capacités gardent leurs limites par défaut (dont 4 appels de service).

Les images normalisées sont préparées dans un fichier temporaire privé du dossier de données, plafonné à 128 Mio par scan, puis incorporées dans la transaction. Au-delà de ce plafond, les pochettes supplémentaires ne sont pas mises en cache ; les informations textuelles restent analysées. L’enregistrement n’a lieu qu’après réussite des programmes et vérification que leurs sessions sont toujours autorisées.

## Cycle de vie et compatibilité

Schéma catalogue 21 (20 pour la première extraction). Installation initiale ou migration : paquet officiel activé une seule fois. Les choix ultérieurs sont persistants. Désactivation/retrait annulent les sessions ; réinstallation hors ligne désactivée par défaut. Les données déjà intégrées au catalogue sont partagées et conservées.

Depuis 0.0.34, import local et dépôts GitHub sont disponibles via le paquet exécutable commun (schéma 2), contribution media.audio-tags.v1 ciblant music et type de catalogue audio-tags. Le service range ci-dessus est le seul accepté, sans dépendance supplémentaire. Installation confirmée, activation et sélection sont distinctes. Les archives peuvent être mises à jour ou restaurées avec les mêmes contrôles que les reconnaisseurs. Exemple : examples/audiotags.
