# Identifiants entre fournisseurs

Depuis Mosaune 0.0.71, un fournisseur décrit ses correspondances dans le champ `identifiers` de son manifeste signé. La déclaration fait partie de la signature et suit l’installation, la mise à jour et le retour arrière. Le noyau ne contient plus le chemin TMDB ni le champ de réponse servant à retrouver une série dans TVDB.

```json
{
  "identifiers": [
    {
      "kind": "series",
      "namespace": "tvdb",
      "source": "api",
      "path": "/tv/{id}/external_ids",
      "field": "tvdb_id"
    }
  ]
}
```

`IdentifierQuery` transporte le type de média, son identifiant chez le fournisseur et l’espace d’identifiants demandé. Le contrat ne connaît ni Sonarr ni Radarr. L’adaptateur de demandes impose ensuite le format numérique accepté par ces services. Un autre fournisseur peut implémenter `MetadataIdentifierProvider.ResolveIdentifier` sans changer cet adaptateur.

Une déclaration autorise au maximum une requête GET par résolution, sur une source JSON déjà autorisée du profil de transport. Le profil conserve la responsabilité de l’origine réseau, du secret, des délais, de la taille maximale et du cache. Aucun en-tête ni adresse complète n’est fourni par cette déclaration. Le chemin contient exactement un `{id}`, les identifiants sont bornés et les chemins relatifs dangereux sont refusés. Le champ cible est un champ JSON de premier niveau : chaîne ou entier positif, sans arrondi. Une correspondance absente ou invalide bloque la demande avant tout ajout au gestionnaire de téléchargements.

Limites actuelles : 32 déclarations, une déclaration par couple type/espace d’identifiants, noms de 48 caractères maximum, identifiants de 128 caractères et chemins de 256 caractères. Les réponses sont bornées à 2 Mio au plus, et au budget inférieur éventuel de la source. Les recherches avec plusieurs étapes, champs imbriqués ou paramètres supplémentaires ne sont pas couvertes par ce premier contrat.

Le paquet officiel TMDB 0.8.0 est créé avec `package-provider -transport -identifiers` et requiert le cœur 0.0.71. Les anciens paquets restent installables ; sans déclaration, une conversion distante affiche une demande de mise à jour, sans repli intégré. L’identifiant propre du fournisseur, déjà enregistré dans la fiche, reste utilisable sans réseau, même lorsque le fournisseur est désactivé. Les appels distants, y compris ceux servis par le cache, respectent les permissions, la désactivation et le retrait.

Cette extraction ne rend pas encore les fournisseurs arbitraires installables : le format signé reste limité à l’adaptateur TMDB et aux origines déjà approuvées. Les autres opérations et l’interface fournisseur complète restent à généraliser.
