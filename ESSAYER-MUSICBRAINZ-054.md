# Mosaune 0.0.54 — les recherches musicales dans l’extension

Le même composant de Musique prépare désormais les recherches MusicBrainz et interprète leurs réponses. Cela couvre la recherche globale, la correction d’association par titre/artiste, les fiches d’album, les discographies et les prochaines sorties. Aucun bouton ou écran supplémentaire.

Après avoir fermé votre ancienne version et lancé la 0.0.54 :

1. Cherchez un album depuis Musique, puis ouvrez sa fiche.
2. Ouvrez l’un de ses artistes pour afficher sa discographie.
3. Sur un album local, ouvrez **Corriger l’association** et recherchez son titre avec son artiste.

Ces parcours doivent fonctionner comme avant. Aucun téléchargement n’est demandé par ces vérifications.

Le composant officiel 0.1.0 fourni par la 0.0.53 est mis à jour vers 0.2.0 s’il est encore installé et intact. Son état désactivé reste désactivé ; un composant retiré n’est pas réinstallé ; un remplaçant choisi reste choisi. Les données de la bibliothèque sont conservées.

Les auteurs de remplaçants doivent maintenant implémenter le protocole de préparation des recherches décrit dans le kit et annoncer Mosaune 0.0.54 minimum. Un ancien remplaçant ne possédant que le décodage reste stocké, mais ses recherches affichent une demande de mise à jour, sans reprise silencieuse par le noyau. Les anciens exemples de familles 4.x restent dans le dépôt ; utilisez les nouveaux 5.x avec cette version.

Le fournisseur MusicBrainz n’est pas encore entièrement indépendant : le serveur gère toujours le transport réseau, sa cadence, le cache et les sources d’images. Aucun déploiement NAS n’est effectué.
