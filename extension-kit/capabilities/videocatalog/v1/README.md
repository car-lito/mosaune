# Catalogue vidéo — contrat v1

Capacité metadata.video-catalog.v1, disponible à partir de Mosaune 0.0.43. Les opérations discover, shelf, profile et posters partagent Query et Result (types.go). Ce contrat transporte des fiches légères, un profil et des références d’affiches ; il ne donne pas accès aux comptes, à la base, au disque ou au réseau.

- Entrée : type, recherche ou identifiant source, page, langue ; sélection et région pour shelf.
- Sortie : au plus 20 œuvres par page, 50 pages, 100 affiches. Chaque œuvre porte son type et son identifiant. Un profil mélange films et séries ; les autres listes respectent le type demandé.
- Le fournisseur renvoie des références d’images opaques. L’hôte valide ces références et construit ses propres liens, ajoute la provenance et conserve les données communes.
- L’adaptateur TMDB n’autorise qu’une requête tmdb.api.v1/get par appel, correspondant exactement à la demande, sans clé ni URL libre dans le module.
- Langue et région sont explicites. Le classement des affiches, l’extraction des crédits, la déduplication et la pagination sont réalisés par le module. Un profil est limité aux 1000 premières œuvres distinctes valides.
- Les contrôles d’accès et de révocation s’appliquent avant le cache et avant toute sauvegarde. Un ancien paquet sans cette capacité échoue sans substitution native.

TMDB 0.5.0 est un consommateur de ce contrat, distribué en archive signée et exécutable WASI. L’installateur de fournisseurs reste limité au pilote TMDB signé ; ce contrat ne prétend pas ouvrir tous les fournisseurs communautaires. Le transport d’images et l’identifiant externe nécessaire à Sonarr restent des adaptateurs hôtes.

Depuis Mosaune 0.0.44, le moteur TMDB natif de remplacement est supprimé. Une pause ou un retrait bloque les consultations ; réactiver le module fourni demande une action explicite. Les services hôtes restent spécifiques au pilote.
