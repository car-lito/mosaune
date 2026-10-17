# Le futur lecteur Mosaune

Décision de préparation — 11 octobre 2026. Aucun nouveau lecteur, analyseur de pistes ou outil de conversion intégré dans la 0.0.6. Le porteur demande de poursuivre d’abord la découverte et les listes.

## Ce que nous voulons construire

Un lecteur dont Mosaune maîtrise l’expérience : commandes, recherche dans la vidéo, reprise, sélection audio/sous-titres avec langues et libellés, épisodes suivants, chapitres, puis introductions/génériques, téléchargements et synchronisation. Les commandes TV doivent être pensées pour la télécommande, celles du mobile pour le tactile. Une identité visuelle commune ne signifie pas reproduire les mêmes boutons aux mêmes endroits sur tous les appareils.

Cela dépasse un habillage du lecteur HTML. Mais développer depuis zéro les décodeurs H.264/HEVC/AV1, les formats audio, le démultiplexage de chaque conteneur, la synchronisation audio/vidéo et l’accélération graphique multiplierait les années de travail et les risques sans servir notre objectif. Ces briques éprouvées restent à réutiliser.

## Découpage envisagé, à valider lors du chantier

| Partie | Responsabilité |
|---|---|
| Serveur Go | Décrire les pistes du fichier, créer les sessions, conserver progression/préférences, servir le média et choisir un mode de lecture adapté au client |
| Modèle partagé | Identifiants de pistes stables, langue, codec, canaux, piste par défaut/forcée, états de lecture, erreurs, capacités du client |
| Client web | Interface Mosaune et orchestration ; moteur du navigateur avec streaming adapté si nécessaire |
| Android / Android TV | Interface Mosaune ; moteur natif tel que Media3/ExoPlayer, avec commandes tactile/télécommande |
| iOS éventuel | Interface cohérente et API commune ; moteur natif tel qu’AVFoundation |

La découverte des pistes doit venir d’une analyse du fichier côté serveur, à préparer ultérieurement avec un outil éprouvé comme ffprobe. Le choix dans l’interface utilisera cette description. Il ne dépendra donc pas uniquement de ce que Chromium expose dans sa barre de lecture.

Un moteur web personnalisé ne peut pas ajouter à lui seul tous les codecs manquants du navigateur. Selon le fichier et l’appareil : lecture directe, changement de conteneur, ou conversion ciblée. Il faudra mesurer ces options sur le Pi et ne jamais supposer qu’il peut convertir toutes les vidéos en temps réel. Les sous-titres texte et graphiques demandent également des stratégies différentes.

Nous pouvons partager les contrats, la logique de session, les comportements et une partie du code. Nous ne promettons pas un moteur unique ni 100 % du code d’interface commun entre navigateur, Android TV et iOS. Go reste le serveur ; Rust reste une option pour une brique spécialisée si des mesures la justifient.

## Validation future

- [ ] Décrire et sélectionner les pistes audio et sous-titres sur les fichiers d’essai.
- [ ] Éprouver démarrage, déplacement, changement de piste, synchronisation et reprise sur web et Chromecast.
- [ ] Mesurer ressources et limites des modes de lecture sur Pi 5.
- [ ] Tester formats de sous-titres, pistes forcées, télécommande et accessibilité.
- [ ] Réutiliser le contrat de session dans les applications mobiles et le hors-ligne.

Références officielles consultées : [Media3 / ExoPlayer](https://developer.android.com/media/media3/exoplayer), [AVFoundation](https://developer.apple.com/av-foundation/), [Media Source Extensions](https://developer.mozilla.org/en-US/docs/Web/API/Media_Source_Extensions_API). Le choix final des moteurs sera vérifié lors du chantier, avec nos fichiers et appareils réels.
