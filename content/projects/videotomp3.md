---
title: "VideoToMP3 Transcriptor"
category: backend
technologies: [Node.js, Express, OpenAI Whisper, GPT-4o, ffmpeg, SSE, yt-dlp]
featured: true
in_progress: false
github_language: JavaScript
catchphrase: "Pipeline de transcription vidéo/audio à 97% moins cher que les services classiques"
---

## Description

Les équipes de contenu dépensent 15-50€ par heure de vidéo en services de transcription. Cette API fait la même chose pour 0,44€ — une réduction de 97%. VideoToMP3 Transcriptor est un pipeline complet qui prend n'importe quelle URL YouTube et livre un texte transcrit, traduit et résumé, en gérant automatiquement tout depuis le téléchargement jusqu'au traitement IA.

## Description Technique

Intégration de trois modèles de transcription IA (GPT-4o, Whisper, Faster-Whisper) avec sélection intelligente selon la qualité requise et le budget. Streaming Server-Sent Events pour le suivi en temps réel. Système de gestion de cookies pour contourner la détection de bots YouTube. Support des playlists pour le traitement en batch de 50+ vidéos. Découpage automatique des gros fichiers, sortie multi-format (TXT, SRT, VTT, JSON), interface CLI et API REST.

## Description Fonctionnelle

Outil d'automatisation de transcription pour équipes de contenu : transcription, traduction et résumé automatiques à partir d'URLs YouTube ou de fichiers locaux. Interface CLI pour usage direct, 12 endpoints REST pour intégration dans des workflows existants.

## Stats

- **12** endpoints API
- **3** modèles IA (GPT-4o, Whisper, Faster-Whisper)
- **50+** langues supportées
- **97%** de réduction de coût vs services classiques
