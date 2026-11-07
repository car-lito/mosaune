# Formulaires des connecteurs — 0.0.62

Dans **Réglages → Téléchargements**, ouvrez le guide de Radarr ou Sonarr. Le titre, les instructions, les champs et leurs aides viennent maintenant du composant de demandes choisi dans Films/Séries.

Votre configuration est conservée. « Tester la connexion » lit uniquement les profils et dossiers du service. Une clé laissée vide conserve celle enregistrée pour la même adresse. La mise en pause reste accessible si le composant est désactivé ou trop ancien.

Pour un contrôle visuel : vérifiez que les deux champs de connexion puis les trois choix de destination sont lisibles avec votre thème. Aucun nouveau téléchargement n’est nécessaire.

Les contributeurs peuvent modifier `extension-kit/modules/acquisition/source/settings.json`, recompiler le programme et installer leur propre paquet. Les champs sont ordonnés dans chacune des deux étapes (connexion, destination). Les contrôles utilisent les couleurs et styles du thème actif ; aucun HTML arbitraire n’est exécuté. Le bouton « Recharger les réglages » recharge le formulaire après un remplacement.

Ce lot ne termine pas l’indépendance des connecteurs : les formats des objets Radarr/Sonarr et le stockage de leur configuration restent dans le serveur. Aucun lecteur ni déploiement NAS n’est inclus.
