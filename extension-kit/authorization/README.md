# Autorisations des services — expérimental, 0.0.28

`Descriptor.Services` décrit les services demandés, pas ceux accordés. Le stockage
`authorization.SQLStore` conserve deux listes distinctes par propriétaire :
`requested` et `granted`, avec une révision. Il n'accorde aucun accès à l'installation.

## Contrat de l'hôte

1. Initialiser la table durant une migration du catalogue.
2. Appeler `Register` avec le manifeste vérifié, dans **la transaction** qui
   remplace le paquet, le registre et son schéma de configuration.
3. Présenter les opérations demandées et accordées à l'administrateur. Décrire
   les services avec des textes contrôlés par l'hôte. Un service inconnu ou sans
   adaptateur ne doit pas devenir un bouton donnant une fausse autorisation.
4. `Apply` reçoit la révision vue par l'administrateur et la liste complète des
   opérations approuvées. Une révision périmée produit `ErrConflict`. Les
   opérations non demandées, répétées ou invalides sont refusées sans écriture.
5. L'adaptateur vérifie **à la fois** le consentement, les droits de l'utilisateur,
   le périmètre précis de la requête et ses limites. Ne pas exposer ce stockage
   comme un service invité : un module ne choisit ni son propriétaire ni ses droits.
6. La révocation doit être sérialisée avec les écritures des résultats : enregistrer
   le changement, annuler les opérations en cours et interdire leurs résultats
   tardifs, y compris après une nouvelle autorisation. Si l'écriture échoue, ne
   pas modifier l'état effectif. Une simple lecture du stockage à l'entrée d'un
   callback **ne suffit pas** à garantir ces propriétés.

`Register` conserve uniquement l'intersection des anciens accords et des nouvelles
demandes. Une opération ajoutée reste sans accord ; une opération retirée puis
réintroduite doit être approuvée à nouveau, y compris lors d'une restauration.
Une signature valide ne change pas cette règle. Les lignes illisibles ne sont
jamais réinitialisées avec des droits permissifs. Maximum : 128 propriétaires,
16 services et 16 opérations par service ; aucun processus supplémentaire au repos.

## Raccordement actuel

TMDB possède un adaptateur effectif : `mosaune.tmdb-search`, service
`tmdb.api.v1`, opération `get`. Son accord couvre les consultations en ligne du
fournisseur, y compris ses images gérées par l'hôte. Les deux moteurs, natif et
Wasm, partagent le même accord. Les sous-requêtes Wasm restent limitées à l'œuvre,
la langue et les chemins attendus ; elles ne reçoivent pas la clé.

Le catalogue passe au schéma 17, avec la sauvegarde préalable habituelle. Pour le
seul fournisseur TMDB livré par Mosaune, la migration conserve une fois son accès
historique (également activé par défaut sur une installation neuve). Cela ne
réactive ni un fournisseur désactivé ni une autorisation déjà enregistrée comme
retirée. Ce choix de l'hôte n'est **pas** le comportement générique de `Register`.

Les routes administrateur `GET/PUT /api/v1/extensions/{id}/authorizations` ne sont
exposées que pour cet adaptateur complet. La révocation utilise le cycle de vie
commun des opérations et leurs verrous d'enregistrement. Elle suspend les
identifications sans compter d'échec, garde les métadonnées et images conservées,
et persiste au redémarrage. Changer de moteur, retirer son paquet optionnel ou
passer en mode de secours ne contourne pas le refus.

Les autres familles doivent encore raccorder leur adaptateur et leur cycle de
révocation avant d'exposer ces routes. Ce stockage ne constitue pas un système
universel de permissions de fichiers, de réseau arbitraire ou de tâches.
