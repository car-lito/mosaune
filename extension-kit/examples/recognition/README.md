# Un reconnaisseur externe en quelques fichiers

Cet exemple n’est pas lié au serveur et n’est jamais installé automatiquement. Il utilise le kit facultatif `modules/recognizer` pour répondre au protocole ; un autre langage capable de produire du WASI peut implémenter le même contrat.

Le manifeste `extension.json` déclare l’identité stable du module, sa version, le minimum de version serveur, sa capacité et sa contribution. L’auteur choisit son propre préfixe d’identifiant : `mosaune.*` est réservé. Le nom d’éditeur est déclaratif et ne vaut pas signature.

Depuis le dossier `extension-kit`, avec Go installé :

```powershell
$env:GOOS = 'wasip1'
$env:GOARCH = 'wasm'
go build -trimpath -ldflags='-s -w' -o examples/recognition/module.wasm ./examples/recognition/source
Remove-Item Env:GOOS, Env:GOARCH
go run ./cmd/package-module examples/recognition/extension.json examples/recognition/module.wasm example-recognition.mosaune.zip
```

Sur Linux :

```sh
GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags='-s -w' -o examples/recognition/module.wasm ./examples/recognition/source
go run ./cmd/package-module examples/recognition/extension.json examples/recognition/module.wasm example-recognition.mosaune.zip
```

La commande de paquet calcule `codeSha256`. Le ZIP contient exactement `extension.json` et `module.wasm`, sans extraction sur le disque à l’import. Taille maximale : 9 Mio comprimés, 8 Mio de programme, 64 Kio de manifeste. Le format local n’ajoute aucune signature ; ne présentez pas un hash comme une preuve d’authenticité.

Importez l’archive dans les paramètres, activez le module, puis choisissez-le pour les scans Films. Il reconnaît les `.mkv` et `.mp4`, ajoute « Exemple — » au titre et ne contacte aucun fournisseur. Utilisez une bibliothèque de test : changer le reconnaisseur affecte les résultats des futurs scans.

Pour essayer une mise à jour, changez `module.version` de `1.0.0` à `1.1.0`, reconstruisez le ZIP puis importez-le. La fiche propose ensuite de restaurer `1.0.0`. Ne changez pas l’identifiant ni le type de média pour une mise à jour.

## Contrat et évolution

Le format d’enveloppe est dans `packageformat/executable.go`. Les champs `module.services` et `module.dependencies` reprennent les contrats communs du moteur. Le lecteur de paquets valide leur syntaxe ; l’hôte décide séparément quelles contributions il sait exécuter. En 0.0.31, les reconnaisseurs doivent avoir exactement la capacité `media.recognition.v1`, une contribution vers `films`, `series` ou `music`, aucun service et aucune dépendance. Un refus explicite évite qu’une permission déclarée soit interprétée comme une permission accordée.

Le contrat de reconnaissance est décrit dans [recognition/v1](../../capabilities/recognition/v1/README.md). L’application teste les formats à l’installation dans le moteur borné ; la prévisualisation ne lance pas de code. Chaque résultat de scan est ensuite validé. Le choix reste explicite et persistant, même si le paquet choisi est désinstallé. Les résultats d’une session annulée ne peuvent plus être enregistrés après changement de programme.

Ce mécanisme remplace uniquement la reconnaissance, pas encore les catalogues, interfaces ou lecteurs. Les prochains rôles auront leurs propres contrats versionnés plutôt qu’un accès libre aux tables de Mosaune.
