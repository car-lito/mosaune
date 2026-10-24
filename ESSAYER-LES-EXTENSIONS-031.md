# Modules externes — préversion 0.0.31

Mosaune peut maintenant charger un reconnaisseur créé hors du serveur, depuis un fichier `.mosaune.zip`. Le choix du programme est indépendant de son installation et de son activation.

## Essayer sans toucher à sa collection

Utilisez une installation de test et un dossier contenant quelques copies de fichiers. Le module de démonstration ajoute volontairement « Exemple — » aux titres : ce n’est pas un remplaçant destiné à votre collection principale.

1. Dans **Réglages → Extensions → À télécharger**, ouvrez **Importer un fichier depuis cet appareil**.
2. Sous **Importer un module de reconnaissance**, choisissez `example-recognition.mosaune.zip`, produit dans `build/` par la construction du projet. Il accompagne aussi les exécutables dans l’archive de vérification GitHub.
3. Relisez le nom, l’éditeur déclaré, le type de média et la version minimale de Mosaune. Confirmez l’installation.
4. Sous **Sur le serveur**, ouvrez **Reconnaisseur de démonstration**. Activez-le, puis cliquez sur **Utiliser pour les scans**.
5. Ajoutez une bibliothèque de test de type Films avec un MKV ou MP4 : son titre commence par « Exemple — ». Le programme installé a réellement reconnu le fichier.
6. Pour revenir au programme fourni, ouvrez **Reconnaissance des films**, activez-le au besoin et choisissez-le pour les scans.

Plusieurs modules peuvent être installés et activés, mais un seul est choisi par type de média. Désactiver ou retirer le module choisi suspend les nouveaux scans : Mosaune ne choisit pas un remplaçant silencieusement. Le choix persiste au redémarrage.

## Mettre à jour et revenir en arrière

Importez un paquet ayant le même identifiant, le même type de média et une version supérieure. La mise à jour conserve l’activation et le choix. Sa version précédente reste disponible sur la fiche via **Restaurer la version…**. Une seule version précédente est conservée ; la restauration échange les deux versions.

Le retrait supprime le programme installé, son archive et sa version précédente. Les médias, métadonnées et progressions communs restent présents. Les modules externes se réinstallent en réimportant leur fichier ; les trois modules fournis restent disponibles hors ligne.

## Limites assumées

- L’import est réservé à l’administrateur. L’éditeur d’un fichier local n’est pas authentifié : son nom est déclaratif. L’empreinte vérifie les octets, pas l’identité de l’auteur.
- Les reconnaisseurs reçoivent uniquement des chemins relatifs, sans accès aux fichiers, au réseau ou aux secrets. Les mêmes limites d’exécution et validations de réponses s’appliquent aux modules officiels et externes.
- Ce format contient un manifeste commun avec capacités, contributions et dépendances. Le serveur accepte actuellement une contribution de reconnaissance pour Films, Séries ou Musique, sans service ni dépendance externe. Les déclarations non prises en charge sont refusées explicitement.
- Le téléchargement de ces nouveaux paquets depuis les dépôts, les contributions d’écrans et l’extraction complète des familles médias restent à terminer. Les dépôts existants pour thèmes/TMDB conservent leur fonctionnement.
- Le schéma de base passe à 19, avec sauvegarde préalable. Les anciennes installations conservent leurs modules absents/désactivés et leur comportement. Aucun déploiement sur le NAS n’est requis.

Pour créer le paquet : [exemple contributeur](extension-kit/examples/recognition/README.md).
