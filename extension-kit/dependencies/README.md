# Dépendances par capacités — première intégration au moteur

État au 19 octobre 2026 : ce contrat est intégré au moteur WASI et testé avec plusieurs modules réels de démonstration. La 0.0.26 ajoute le [registre durable](../registry/README.md), les plans d’opérations et le raccordement du paquet TMDB installé. Les installateurs génériques et l’écran de choix entre fournisseurs restent à faire. Le pilote de paquets TMDB continue à refuser les dépendances externes. Aucun lecteur n’est ajouté par ce changement.

## Déclarer le besoin, choisir l’implémentation

Un descripteur peut ajouter un champ facultatif `dependencies` :

```json
{
  "id": "atelier.films",
  "version": "1.0.0",
  "capabilities": ["media.films.v1"],
  "dependencies": ["playback.video.v1"]
}
```

Le fournisseur du besoin peut être n’importe quel module actif annonçant exactement `playback.video.v1`. Les noms vidéo de cet exemple sont un scénario de test ; le contrat des opérations de lecture reste à définir. Une dépendance n’autorise aucun accès aux fichiers, secrets, services ou données d’un autre module. Les contrôles par invocation restent nécessaires.

`dependencies.Resolve` est indépendant de WASI, du catalogue et du type de média. Il reçoit un descripteur, les candidats et les choix de l’hôte. Il renvoie les associations capacité → identifiant, ou un problème structuré : `missing`, `disabled`, `ambiguous`, `invalid_choice`. Plusieurs fournisseurs actifs nécessitent un choix explicite, jamais une priorité fondée sur l’ordre d’installation. Les candidats d’une erreur sont triés pour un affichage stable. La version de l’API est dans le nom de capacité ; cela ne remplace pas les futures contraintes de versions des paquets.

## Cycle de vie du moteur

- `Load` valide et copie le descripteur, sans activer ses dépendances.
- `Enable` sélectionne l’unique fournisseur actif compatible. `EnableWithProviders` reçoit les choix explicites de l’hôte. Les dépendances doivent être activées d’abord ; un cycle ne peut donc pas devenir actif.
- `Bindings` retourne une copie des associations actives. L’ajout d’un autre fournisseur ne les modifie pas. Pour changer une association, désactiver le consommateur, puis le réactiver avec le nouveau choix.
- `Dependents` donne les consommateurs actifs directs et indirects pour un aperçu d’impact, les plus éloignés en premier.
- `Disable` et `Unload` refusent de casser les consommateurs actifs et renvoient `DependentsError` sans modifier l’état.
- `DisableCascade` est une opération explicite de l’hôte qui annule les générations de tous les consommateurs avant celle du fournisseur, sous le même verrou. Elle ne désinstalle aucun paquet et n’efface aucune donnée.
- Retirer un consommateur ne retire jamais ses fournisseurs partagés. Un consommateur désactivé peut rester chargé sans fournisseur ; sa prochaine activation est bloquée jusqu’à résolution du besoin.

Les opérations recalculent leurs conditions sous verrou : un aperçu n’est pas un jeton d’autorisation. L’hôte doit encore vérifier les permissions et gérer le consentement pour l’impact d’une action. Le moteur ne télécharge pas automatiquement un fournisseur, ne stocke pas la préférence et ne route pas implicitement des appels entre invités.

Les anciens descripteurs sans `dependencies` gardent leur comportement. Le moteur connaît ce champ en 0.0.26, mais un paquet installable l’utilisant devra annoncer la première version du noyau dont l’installateur accepte sa famille et ses dépendances ; cette étape n’est pas encore livrée. Les messages JSON restent stricts et un ancien hôte doit refuser les nouveaux champs plutôt que les ignorer.

## Prochaines intégrations

1. Registre persistant commun aux familles de paquets et choix de fournisseurs, avec aperçu des dépendances avant mutation.
2. Installation et mise à jour transactionnelles, contraintes de versions, retour arrière et migrations.
3. Distribution automatique des dépendances officielles une seule fois, sans exécuter un paquet avant validation et autorisation.
4. Réglages/stockage génériques, extraction des modules et contributions UI ; lecteurs après ce travail.

Cible produit : [Lecteurs et extensions](../../LECTEURS-ET-EXTENSIONS.md).
