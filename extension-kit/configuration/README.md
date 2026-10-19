# Réglages déclaratifs v1

Livrés dans Mosaune 0.0.18 pour le module TMDB fourni. Ce contrat décrit un formulaire ; il ne gère pas l’installation du code. Depuis la 0.0.19, un installateur distinct accepte les paquets TMDB signés reconnus. Le fichier réel est [providers/tmdb/settings.json](../providers/tmdb/settings.json).

## Responsabilités

L’auteur décrit les champs, les textes d’aide, les étapes de configuration et les liens HTTPS. Mosaune construit les contrôles avec les styles du thème, valide les modifications côté serveur et les transmet à un adaptateur de stockage autorisé. Le fichier ne contient ni HTML, JavaScript, SQL, chemin de fichier ni adresse d’API de configuration à exécuter.

Chaque capacité peut évoluer séparément : ce formulaire n’impose pas aux futurs modules leur disposition complète ni une liste fermée de fonctionnalités. Pour une interaction plus riche, une autre capacité d’interface versionnée sera nécessaire ; elle n’est pas livrée ici.

## Exemple de description

```json
{
  "version": 1,
  "title": "Mon fournisseur",
  "description": "Configurez les informations utilisées par ce module.",
  "fields": [
    {"id":"apiToken","label":"Clé API","type":"secret","minLength":20,"maxLength":2048,"format":"token"},
    {"id":"label","label":"Nom affiché","type":"text","maxLength":100},
    {"id":"enabled","label":"Activer cette option","type":"boolean"},
    {"id":"region","label":"Région","type":"select","options":[{"value":"be","label":"Belgique"},{"value":"fr","label":"France"}]}
  ]
}
```

L’exemple illustre le contrat : ces options supplémentaires ne sont pas des fonctions TMDB déjà branchées. Limites : 32 champs, 32 choix par sélection, 12 instructions, 8 liens HTTPS sans identifiants dans l’URL. Les longueurs de texte sont validées en octets UTF-8 côté serveur. Le navigateur borne aussi la saisie, mais la validation serveur fait autorité. `format: token` retire les espaces en début/fin et interdit les espaces/tabulations/retours internes.

## Lecture et modification

La réponse fournit `schema`, `values` pour les valeurs ordinaires et `secretsConfigured` pour la présence des clés. Une clé existante n’est jamais renvoyée au navigateur, y compris à un administrateur. Un formulaire vide conserve les secrets ; leur retrait utilise `clearSecrets` et une confirmation visible.

```json
{"values":{"label":"Ma source","enabled":false},"clearSecrets":[]}
```

Champ absent = conserver. Valeur inconnue, mauvais type, `null`, choix non déclaré, secret trop court ou modification et retrait simultanés du même secret = rejet de toute la demande avant écriture. Une erreur ne contient jamais la valeur reçue.

Depuis 0.0.27, les routes sont `GET` et `PUT /api/v1/extensions/{id}/configuration`, réservées à l’administrateur ; l’ancien chemin avec `/bundled/` reste compatible. L’hôte choisit le propriétaire à partir de l’identité enregistrée ; un fichier ne peut pas choisir sa table ou accéder aux réglages d’une autre extension. TMDB utilise le [stockage commun des réglages](STORAGE.md), avec un propriétaire partagé entre fournisseur intégré et paquet. Désactiver le module conserve cette clé et les fiches ; il n’y a pas de base parallèle. Le formulaire envoie la révision lue pour éviter d’écraser une modification plus récente.

## Accès aux services

Le descripteur du module annonce par exemple :

```json
{"name":"tmdb.api.v1","operations":["get"]}
```

Le moteur prend l’intersection des opérations déclarées et des services accordés par l’hôte à chaque invocation. Une déclaration ne donne aucune autorisation. Sans déclaration, aucun service ne passe, même si un appelant en propose un. Les contraintes précises de route, œuvre, langue et volume restent appliquées par l’adaptateur hôte. Les descriptions visibles des accès viennent du registre de l’hôte.

## Prochaines étapes

En 0.0.27, le stockage des réglages est générique et raccordé au paquet TMDB signé ; le registre durable des dépendances reste distinct. Restent les migrations privées explicites, le chiffrement des secrets au repos, les politiques d’autorisation des autres familles et les budgets globaux du moteur. Un paquet tiers ne peut pas déclarer son propre stockage exécutable à travers ce formulaire. Voir le [registre commun](../registry/README.md) et le [contrat de stockage](STORAGE.md) pour leur périmètre actuel.
