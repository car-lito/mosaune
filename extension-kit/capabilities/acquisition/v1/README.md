# Préparation des demandes

Le rôle `acquisition.plan.v1` cible `video` et nécessite Mosaune 0.0.59. Une seule implémentation est sélectionnée pour Films et Séries. Le module fourni est dans `modules/acquisition/source` et fait partie des archives de familles 8.0/8.1.

Le programme répond à `describe` avec une version de contrat égale à 1. L’opération `plan` reçoit le connecteur, l’identifiant distant, la présence d’un fichier, le périmètre des saisons et le type de série. Elle renvoie un `Plan`. Les structures et leur validation sont définies dans `acquisition.go`.

Un plan indique la création, la mise à jour ou l’absence d’écriture, les saisons à surveiller et les recherches à déclencher. Le serveur vérifie que les choix respectent la demande confirmée. Il applique le plan à la ressource complète pour conserver les champs distants inconnus. Les dossiers et profils existants sont protégés. Les clés, chemins et réponses HTTP complètes restent dans le serveur.

Le fichier `policy.json` fourni contient `searchMode`. La valeur `auto` déclenche la recherche habituelle et `none` active uniquement la surveillance. Pour créer un remplaçant, modifiez le programme ou cette politique, compilez le module pour `wasip1/wasm` et empaquetez-le avec `cmd/package-module`. Le manifeste doit déclarer le rôle ci-dessus avec la cible `video`, un identifiant propre et aucune dépendance ou service.

L’installation utilise le cycle de vie commun des programmes. L’activation et la sélection sont explicites. Le retrait conserve les demandes enregistrées. Une confirmation préparée avant le remplacement doit être vérifiée à nouveau. La désactivation annule les connexions actives et bloque les nouvelles connexions du connecteur. Le plan ne rejoue aucune opération après une erreur d’envoi.

Le type de paquet de dépôt est `acquisition`. Les archives de familles peuvent inclure ce composant sous l’emplacement `acquisition` avec `shared` activé. Les adaptateurs HTTP, l’interprétation du suivi et les formulaires constituent les prochaines étapes de l’extraction.
