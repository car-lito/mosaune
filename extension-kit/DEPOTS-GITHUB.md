Catalogue officiel publié : https://github.com/car-lito/mosaune-extensions. Intégré à partir de la 0.0.21, actualisé uniquement sur demande. Les dépôts communautaires suivent le même contrat ci-dessous.

# Dépôts d’extensions — contrat 1, Mosaune 0.0.20

Un dépôt annonce des paquets ; il ne définit ni leur format ni leurs droits d’exécution. Le lecteur de dépôt remet les octets vérifiés aux installateurs existants. Les modèles de médias, comptes et réglages restent indépendants de la source de téléchargement.

## Structure

Le fichier `mosaune-repository.json` est à la racine d’un dépôt GitHub public. Les paquets ZIP sont suivis dans le même dépôt. Pas de GitHub Releases, de liens arbitraires, de scripts d’installation, d’identifiants GitHub ou de jetons dans ce contrat.

```json
{
  "schema": 1,
  "name": "Mon catalogue",
  "packages": [
    {
      "id": "auteur.theme",
      "name": "Mon thème",
      "publisher": "Auteur",
      "description": "Description identique à celle du manifeste.",
      "version": "1.0.0",
      "kind": "theme",
      "minCore": "0.0.15",
      "path": "packages/theme-1.0.0.mosaune.zip",
      "sha256": "REMPLACER_PAR_LES_64_CARACTERES_HEXADECIMAUX_DU_SHA256_DU_ZIP"
    }
  ]
}
```

Cet exemple est un modèle : sa valeur `sha256` doit être remplacée. L’export disponible dans l’application fournit un dépôt complet avec des empreintes réelles.

`kind` accepte `theme` ou `provider`. Chaque couple identifiant/version est unique dans un catalogue ; plusieurs versions d’un paquet sont possibles. Les cartes affichent la dernière version et les fiches proposent les versions précédentes. Les versions sont des triplets numériques, sans suffixe. Les champs inconnus sont rejetés afin de ne pas ignorer silencieusement une future exigence.

L’identifiant, le nom, l’éditeur déclaré, la description, la version et le minimum noyau doivent être **identiques au manifeste du paquet**. L’empreinte est le SHA-256 des octets du **ZIP complet**, distinct de l’empreinte interne utilisée pour identifier une installation. Exemple PowerShell : `Get-FileHash -Algorithm SHA256 -LiteralPath .\packages\theme-1.0.0.mosaune.zip` ; convertir le résultat en minuscules dans le catalogue.

## Lecture et limites

- Adresse source : `https://github.com/auteur/depot`, référence facultative. Nom de branche par défaut si omis. Pas d’URL contenant des identifiants, un port, un chemin de fichier, des paramètres ou une ancre.
- Requêtes exclusivement à `https://api.github.com/repos/{auteur}/{depot}/contents/{chemin}`, avec le type brut `application/vnd.github.raw+json` et `ref` encodé séparément. Aucune redirection suivie, aucun jeton ou cookie de Mosaune transmis.
- Délai réseau 30 secondes ; flux de réponse borné même sans Content-Length. Catalogue 256 Kio, thème 256 Kio, fournisseur 9 Mio ; 128 entrées par source, 20 sources ajoutées.
- Chemins relatifs composés de segments ASCII alphanumériques, tiret, soulignement et point. Segments vides, `.` et `..`, antislashs, schémas d’URL et encodages `%` refusés.
- Les champs de texte sont rendus comme texte, sans HTML fourni par un dépôt. La description d’un dépôt n’accorde aucune identité officielle.
- Actions réseau explicites seulement ; opérations réseau de dépôt sérialisées par serveur. Pas de scrutation ni de téléchargement à chaque affichage du catalogue.

Référence de l’API : [GitHub — Repository contents](https://docs.github.com/en/rest/repos/contents). Les limites GitHub peuvent bloquer temporairement les requêtes ; l’interface explique l’erreur sans perdre le dernier catalogue valide.

## Persistance et installation

Schéma SQLite 13, table `extension_repositories` : URL canonique, référence, index validé et date du dernier succès. L’identité locale est dérivée de l’URL et de la référence. Une actualisation invalide ne remplace pas l’index. Retirer la source ne touche pas aux installations. Au redémarrage, les sources et leur liste sont conservées ; aucune requête réseau implicite.

Une carte contient l’empreinte attendue ; le serveur refuse une sélection devenue périmée. Il télécharge le ZIP, vérifie son empreinte, applique le validateur du type de paquet et compare le manifeste à la carte. Les octets ainsi vérifiés sont présentés à l’administrateur puis envoyés à l’installateur local existant. Celui-ci revérifie le paquet et la version installée attendue pour empêcher un écrasement concurrent. Le catalogue ne change donc pas le cycle de restauration TMDB ni les droits des thèmes.

Toutes les routes sont réservées aux administrateurs. Les mutations exigent le mécanisme de protection existant. Les utilisateurs ordinaires ne peuvent pas provoquer des requêtes réseau par ces routes.

## Confiance et prochaines familles

Les thèmes restent déclaratifs et sans code. Le pilote fournisseur reste limité à l’adaptateur TMDB et à la clé de développement épinglée ; un catalogue externe ne peut pas élargir cette politique. La signature ne garantit pas l’absence de bogues. Les budgets globaux de compilation/processus et les éditeurs communautaires exécutables restent à traiter avant leur ouverture.

L’index est un contrat de distribution séparé. D’autres hébergeurs, familles ou services pourront être ajoutés par leurs adaptateurs et des versions de contrat explicites, sans modifier les bases de médias ni permettre au catalogue d’accéder directement à celles-ci.
