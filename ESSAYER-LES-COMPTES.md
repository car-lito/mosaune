# Essayer les comptes — Mosaune 0.0.13

Cette version s’utilise encore sur ton PC. Pas besoin de NAS, d’e-mail ni de compte en ligne.

## 1. Mettre à jour et créer ton administrateur

1. Arrête l’ancien serveur avec **Ctrl+C** dans sa fenêtre. Copie le dossier **data** ailleurs par précaution.
2. Lance **DEMARRER-WINDOWS.cmd**. Une sauvegarde de la base est aussi créée automatiquement dans data avant le passage au format 9 : `mosaune-before-accounts-…db`.
3. Ouvre `http://127.0.0.1:47831`. Dans la fenêtre du serveur, copie le **code unique de configuration** et colle-le dans le formulaire.
4. Choisis ton nom d’utilisateur et un mot de passe d’au moins **12 caractères**, puis confirme-le.

Le code sert une seule fois. Il n’est pas ton mot de passe. Après un redémarrage avant configuration, utilise le nouveau code affiché.

Ton premier compte récupère les progressions, favoris, listes et playlists enregistrés sur le serveur. Les dossiers, fournisseurs, demandes de téléchargement, associations et affiches restent conservés. La file d’écoute du navigateur repart vide ; les nouvelles files sont séparées par compte.

## 2. Créer un deuxième compte

Dans **Réglages → Utilisateurs → Créer un compte** :

- Renseigne un **nom d’utilisateur** (lettres sans accents, chiffres, point, tiret ou soulignement).
- **Mot de passe vide** : Mosaune crée un lien d’invitation. La personne ouvre ce lien et choisit elle-même son mot de passe à la première connexion. Le lien dure sept jours et ne fonctionne qu’une seule fois. Tu peux le renouveler si nécessaire.
- **Mot de passe rempli** : la personne peut se connecter immédiatement avec ce nom et ce mot de passe. Elle peut ensuite changer son mot de passe dans Mon compte.
- Choisis les types de médias et les bibliothèques accessibles. Un nouveau dossier est privé par défaut pour les comptes ordinaires.

Pour tester sur ce PC, garde ton administrateur dans ton navigateur habituel et ouvre le lien dans **une fenêtre privée ou un autre navigateur**. L’invitation contient un accès au compte : transmets-la uniquement à son destinataire. Elle est affichée à la création ou au renouvellement, pas conservée en clair dans la base.

Si tu ouvres une invitation dans un navigateur déjà connecté, Mosaune te propose d’abord de te déconnecter. Tu peux aussi ouvrir une fenêtre privée pour garder ton compte administrateur ouvert.

## 3. Vérifier les permissions

| Droit | Ce qu’il permet |
|---|---|
| Voir | Afficher les fichiers de cette bibliothèque dans la collection |
| Lire | Lire les fichiers et enregistrer la progression ou l’état vu/écouté |
| Modifier | Corriger les associations et changer les affiches communes |
| Demander | Demander un téléchargement dans cette bibliothèque, si le connecteur est configuré |

Les types de médias autorisés contrôlent aussi les rubriques et la découverte. Un contenu absent du NAS peut toujours apparaître dans la découverte d’un type autorisé ; cela ne donne pas accès aux fichiers d’une bibliothèque privée. Les affiches et métadonnées des œuvres sont partagées, les progressions et listes personnelles ne le sont pas.

Essaie avec un compte autorisé à voir une bibliothèque mais pas à la lire : ses fiches restent visibles, la lecture est refusée. Autorise ensuite la lecture, reconnecte ce compte et vérifie qu’il démarre avec sa propre progression. Une modification des permissions révoque ses anciennes sessions : elle s’applique aux requêtes suivantes. Un transfert déjà commencé n’est pas interrompu rétroactivement.

Les utilisateurs ordinaires disposent de **Mon compte** ; les paramètres serveur et la gestion des comptes restent réservés aux administrateurs. Les playlists sont privées, y compris entre administrateurs. Le suivi des demandes concerne les bibliothèques accessibles et reste partagé entre leurs utilisateurs.

## 4. Mot de passe oublié et comptes désactivés

L’administrateur peut ouvrir un compte dans **Utilisateurs**, saisir un nouveau mot de passe puis enregistrer. Les anciennes sessions du compte sont révoquées. Laisser ce champ vide en modification conserve le mot de passe existant.

**Désactiver** empêche la connexion tout en conservant les données. Il n’y a pas de suppression définitive de compte dans cette première version. Un administrateur ne peut pas retirer ses propres droits ni se désactiver lui-même.

Si tu as oublié ton propre mot de passe administrateur et qu’aucun autre admin ne peut t’aider : arrête le serveur, ouvre un terminal dans le dossier du projet (clic droit → Ouvrir dans le terminal), puis lance :

```powershell
.\build\mosaune-0.0.13-windows-amd64.exe -data-dir .\data -recover-admin TON_NOM
```

Remplace `TON_NOM` par ton nom d’utilisateur. Ouvre l’adresse habituelle dans une fenêtre privée, copie le nouveau code affiché par le serveur et choisis un nouveau mot de passe. Cette récupération nécessite l’accès au serveur ; elle ne s’effectue pas par une simple demande anonyme sur le site.

## 5. Mon compte et limites de cette livraison

**Mon compte** propose le changement de mot de passe et les thèmes **clair, sombre ou selon l’appareil**. Le français reste la seule langue de l’interface. Les extensions de traduction et les permissions propres aux futurs paquets feront partie de l’étape extensions.

Les mots de passe sont hachés avec Argon2id et un sel individuel. La vérification utilise environ 19 Mio de mémoire de travail ; un seul calcul de mot de passe est effectué à la fois. Ce n’est pas une mesure de la mémoire totale du serveur ni un benchmark sur ton Pi. Les tentatives de connexion sont limitées à dix par tranche de cinq minutes sur cette installation locale, connexions réussies comprises. Les sessions expirent après sept jours.

Cette livraison **reste liée à 127.0.0.1**. Les invitations fonctionnent donc sur cet ordinateur. L’accès depuis d’autres appareils, HTTPS, les extensions installables et CasaOS seront traités aux jalons suivants.

Retour à 0.0.12 : restaurer le dossier data sauvegardé avant migration avec le serveur arrêté. Ne pas essayer d’ouvrir la base au format 9 avec l’ancien exécutable. La copie SQLite automatique permet aussi une restauration de la base seule : conserver l’original, utiliser un dossier data séparé avec la copie renommée `mosaune.db`, sans reprendre les fichiers `-wal` et `-shm` de la base plus récente.

Référence : [OWASP — stockage des mots de passe](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).
