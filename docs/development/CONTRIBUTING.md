# Contribuer ou reprendre le développement

Lire d’abord [AGENTS](../../AGENTS.md), [STATUS](../STATUS.md), [DECISIONS](../product/DECISIONS.md) et [ROADMAP](../ROADMAP.md). Ce parcours vaut pour un humain comme pour une IA découvrant le dépôt.

## Choisir et livrer un lot

1. Vérifier l’état Git et le dernier résultat de CI. Prendre un objectif cohérent dans la roadmap ; ne pas recréer un mécanisme déjà décrit comme livré.
2. Repérer contrats, implémentations, migrations et tests concernés. Distinguer primitive hôte et règle métier avant d’ajouter du code au serveur.
3. Écrire une proposition concrète si le lot modifie un choix produit, un format de stockage ou une frontière de confiance. Pour les corrections ordinaires, conserver les conventions établies.
4. Implémenter et tester les comportements utiles, y compris absence/désactivation du composant, permissions et réponse périmée lorsque concernés. Ne pas modifier une base personnelle pour obtenir une démonstration.
5. Mettre à jour documentation et historique, vérifier liens et arborescence. Décrire résultat, tests et limites dans la proposition de modification.

## Conventions

- [Arborescence obligatoire](../architecture/OVERVIEW.md). Tests adjacents, données de test synthétiques, aucun fichier « essai-version » supplémentaire à la racine.
- Contrats versionnés : bornes explicites, validation des réponses, compatibilité et migration documentées. Respecter les états attendus avant une écriture.
- Une désactivation doit arrêter les appels concernés et empêcher leurs résultats tardifs d’être publiés. Une réactivation ne valide pas un ancien appel.
- L’UI et les providers utilisent des capacités déclarées et validées, pas une reconnaissance de nom. Un fournisseur ne choisit pas librement sa clé de confiance ou ses permissions réseau.
- Les fichiers dans `resources/packages/` sont signés : ne pas les modifier ni les recomprimer pour « nettoyer ». Les anciens paquets et migrations nécessaires restent conservés.
- Versions du serveur (`server/main.go`) et du produit (`package.json`, verrou npm) cohérentes. La version d’un composant ou paquet évolue selon son propre contrat, pas à chaque version du serveur.
- Français dans les explications produit ; conserver les conventions de code existantes. Ne pas ajouter de jargon technique dans les parcours utilisateur.

## Vérification avant envoi

`node scripts/project.mjs check` reconstruit les ressources puis exécute Svelte, Node, tests Go et vet des deux modules. `build` construit les trois cibles ; `ci` combine les deux. `node scripts/test-minimal.mjs` vérifie le parcours depuis un catalogue temporaire vide. `node scripts/check-repository.mjs` contrôle les liens locaux, les fichiers racine et les chemins interdits à la publication.

Pour une modification étroite, des tests ciblés suffisent pendant le développement ; la CI complète reste la référence avant fusion. Pour une réorganisation, faire une construction depuis une copie propre des fichiers suivis : les anciennes sorties locales peuvent masquer un oubli de chemin.

## Publication et données privées

Le propriétaire autorise les publications de préversions sur son dépôt ; cela n’implique aucun déploiement sur son NAS. La CI peut publier une image GHCR après validation. Ne jamais ajouter credentials, médias, bases SQLite, clés privées, fichiers `.env` réels ou rapports issus d’un catalogue personnel.

Un rapport de problème doit contenir version, plateforme, étapes et erreur expurgée. La licence globale reste à décider : le dépôt public seul ne vaut pas autorisation open source. Les contributions et dépendances doivent conserver leurs notices.
