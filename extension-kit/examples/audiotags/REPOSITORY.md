# Exemple de lecteur musical indépendant

Ce dépôt de démonstration contient deux versions du même lecteur ID3v1. Il ne remplace pas le lecteur officiel par défaut. Il lit uniquement les 128 derniers octets des fichiers et ne prend pas en charge ID3v2, FLAC ou les pochettes.

Dans Réglages → Extensions, importer le paquet ou ajouter ce dépôt GitHub public. Vérifier l’accès au fichier en cours, installer, activer puis choisir « Utiliser pour les scans ». Ajouter un nouveau MP3 ID3v1 à une bibliothèque de test. Les fichiers inchangés ne sont pas réanalysés.

Le paquet utilise le contrat media.audio-tags.v1 et le service media.file.read.v1/range. Aucun accès réseau, chemin arbitraire ou secret n’est accordé. La confirmation d’installation autorise ce service précis ; l’état actif reste un choix séparé. Une mise à jour avec d’autres services ou opérations est refusée.

Les versions 1.0.0 et 1.1.0 exécutent le même code : elles servent à vérifier les mises à jour et restaurations, pas à annoncer des différences fonctionnelles.
