# Mosaune 0.0.12 — demander un film ou une série

Le suivi des fichiers et l’identification en arrière-plan sont expliqués dans [ESSAYER-LE-SCAN.md](ESSAYER-LE-SCAN.md). La demande Radarr et le démarrage dans Transmission ont été validés par l’utilisateur en 0.0.9 ; l’arrivée finale sur son NAS reste à vérifier.

Les boutons des fiches peuvent maintenant envoyer une demande à **Radarr** pour les films ou à **Sonarr** pour les séries. La musique conserve son bouton « bientôt » : son raccordement viendra séparément.

## 1. Lancer la mise à jour

1. Arrêter l’ancien serveur Mosaune avec **Ctrl+C** dans sa fenêtre.
2. Copier **tout le dossier data**, serveur arrêté, pour conserver une sauvegarde.
3. Relancer **DEMARRER-WINDOWS.cmd**, puis recharger la page du navigateur.
4. Vérifier **v0.0.12** dans la barre latérale. La base passe au format 8 ; gardez la sauvegarde avant mise à jour.

Le serveur reste un prototype accessible depuis ce PC uniquement. Il peut joindre les services de ton NAS ; ce n’est pas encore le déploiement de Mosaune sur le NAS.

## 2. Relier les services

Dans **Réglages → Téléchargements**, un guide dépliable est également disponible.

1. Ouvrir Radarr dans son interface habituelle. Dans **Settings → General → Security**, copier sa **API Key**.
2. Dans le bloc **Radarr** de Mosaune, saisir son adresse accessible depuis le PC. Exemple fictif : `http://192.168.1.50:7878`. Si tu utilises un préfixe comme `/radarr`, le conserver. Ne pas ajouter de clé dans l’URL.
3. Coller la clé dans le champ prévu. Elle reste côté serveur après enregistrement et n’est pas renvoyée dans les réponses de réglages. Inutile de la partager dans le chat.
4. Cliquer **Tester la connexion**. Ce test lit les réglages du service et ne demande aucun téléchargement.
5. Choisir un **profil qualité** et un **dossier de destination** parmi ceux déjà configurés dans Radarr.
6. Choisir le **dossier correspondant dans Mosaune**. Si la liste est vide, cliquer **Ajouter un dossier de films** (ou de séries), coller son chemin accessible depuis ce PC, puis **Ajouter et sélectionner**. Terminer par **Enregistrer et activer**.
7. Répéter pour Sonarr, avec sa propre clé et son propre port (souvent 8989).

Le dossier de Radarr/Sonarr est celui vu par leur conteneur, par exemple `/movies` ou `/tv`. Le dossier Mosaune est celui que ton PC peut lire, par exemple un partage réseau `\\NAS\Medias\Films`. Les chemins peuvent être différents, mais ils doivent désigner les mêmes fichiers. S’il manque dans la liste, le bouton **Ajouter un dossier** sous celle-ci permet maintenant de l’ajouter directement ; son type Films/Séries est choisi automatiquement. Le même ajout reste possible dans **Réglages → Vos dossiers**. Un dossier d’essai local sans rapport avec le NAS ne permettra pas de détecter les arrivées sur le NAS.

Les œuvres déjà présentes dans le service conservent leur destination et leur profil existants. Si elles se trouvent ailleurs, le dossier Mosaune choisi doit aussi permettre de les lire. Un connecteur a une seule bibliothèque associée dans cette première version.

Les clés sont conservées dans le catalogue SQLite local, comme la clé TMDB : protège sa sauvegarde. Changer l’adresse d’un service impose de saisir à nouveau sa clé. Une clé laissée vide est conservée uniquement si l’adresse reste identique.

## 3. Faire une première demande

1. Rechercher un film absent de ta collection et ouvrir sa fiche TMDB.
2. Cliquer **Demander le téléchargement**.
3. Lire le récapitulatif : service, qualité, destination et bibliothèque Mosaune.
4. Cliquer **Confirmer la demande** pour autoriser l’ajout, la surveillance et la recherche dans ta stack existante.
5. Ouvrir **Demandes** dans le menu de gauche pour suivre le résultat.

Pour une série : **Demander la série** concerne toutes ses saisons, les spéciaux et les futures saisons. Pour un essai plus petit, cliquer sur une **affiche de saison**, puis **Demander cette saison**. Seule cette saison est ajoutée à la surveillance ; les autres réglages déjà actifs dans Sonarr restent tels quels.

**Ma liste** et les favoris ne déclenchent aucun téléchargement. Les demandes de cette version passent par une confirmation explicite. TMDB doit pouvoir fournir l’identifiant TVDB de la série pour Sonarr ; s’il manque, Mosaune refuse l’envoi plutôt que de choisir une autre série par son nom. Certains découpages de saisons TMDB/TVDB peuvent différer : vérifier le numéro dans le récapitulatif et dans Sonarr.

## 4. Comprendre les états

| État | Ce qu’il signifie |
| --- | --- |
| En attente | Demande acceptée ; aucune activité de téléchargement détectée pour le moment. La disponibilité dépend des indexeurs et des règles du service. |
| En téléchargement | Une entrée correspondante figure dans sa file. Le pourcentage concerne la file active, pas toute la série. |
| À vérifier | Radarr/Sonarr signale un blocage : ouvrir sa file d’activité pour en voir la cause. |
| Importé · dossier à vérifier | Le service possède le fichier, mais Mosaune ne peut pas encore le lire et l’identifier dans la bibliothèque associée. |
| Partiellement disponible | Certains épisodes sont lisibles ; la saison ou la série n’est pas complète. Le décompte inclut les épisodes futurs déjà référencés par Sonarr. |
| Disponible | Les fichiers attendus sont identifiés et lisibles dans Mosaune. Cela ne garantit pas que leurs codecs sont pris en charge par le navigateur actuel. |
| Envoi à vérifier | Le service n’a pas confirmé toutes les étapes. Vérifier directement dans Radarr/Sonarr ; Mosaune ne renvoie pas la recherche automatiquement. |
| Service inaccessible / Suivi en pause | Mosaune ne peut pas actualiser le suivi. Une demande déjà envoyée continue d’être gérée par le service. |

Le serveur vérifie les demandes environ une fois par minute, tant qu’il reste ouvert. Les demandes déjà entièrement disponibles sont revérifiées moins souvent (environ 15 minutes). Après import, les dossiers concernés sont analysés au maximum une fois toutes les cinq minutes. Un gros dossier, plusieurs demandes ou un fournisseur lent peuvent allonger l’attente. **Actualiser le suivi** relit les services immédiatement ; le délai de cinq minutes entre analyses du même dossier reste appliqué. Le bouton **Actualiser** du dossier permet aussi une analyse manuelle.

Les nouveaux fichiers sont identifiés avec le mécanisme TMDB existant, par lots. Si une identification manque ou reste ambiguë, actualiser le dossier et utiliser **Corriger l’association**. Une petite notification dans Mosaune annonce les nouvelles disponibilités lorsque l’application est ouverte et visible ; le contenu apparaît alors dans sa bibliothèque. Ce n’est pas une notification système ou mobile.

Un double clic ne crée pas une deuxième demande identique. Les demandes et leur destination sont conservées au redémarrage. Après une coupure pendant l’envoi, le suivi recherche l’œuvre dans le service, sans répéter une commande potentiellement déjà exécutée. Pour relancer une recherche incertaine, le faire dans Radarr/Sonarr après vérification.

## Limites de cette tranche

- API v3 de Radarr et Sonarr (Sonarr 4 visé) ; ta configuration NAS réelle reste à tester. Les essais de développement utilisent des services simulés.
- Pas encore de webhook entrant, de demandes musicales, de demandes automatiques depuis les abonnements, de suppression/annulation distante ni de relance de recherche depuis Mosaune.
- Mettre le suivi en pause ne stoppe pas les téléchargements du NAS. Les arrêter ou changer leur surveillance se fait dans Radarr/Sonarr.
- Changer l’adresse du connecteur met les anciennes demandes en pause pour éviter de suivre une autre installation par erreur. Restaurer l’adresse d’origine reprend leur suivi. La bibliothèque associée à une demande est mémorisée au moment de l’envoi.
- Maximum 1 000 demandes conservées ; pas encore d’archivage. La file distante est lue par pages, jusqu’à 2 000 entrées. Pas de garantie de performances sur une grande bibliothèque ou sur le Pi : ces mesures viendront avec le déploiement.
- Aucune nouvelle dépendance serveur, aucun transcodage et aucun changement du lecteur vidéo dans cette version.

Références des contrats : [API Radarr](https://radarr.video/docs/api/), [API Sonarr](https://sonarr.tv/docs/api/), [identifiants externes TMDB](https://developer.themoviedb.org/reference/tv-series-external-ids).
