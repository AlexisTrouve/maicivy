---
title: "LiveConf"
category: ai
technologies: [Python, FastAPI, Flutter, Claude API, GPT-4o, WASAPI, WebSocket, Faster-Whisper]
featured: true
in_progress: true
github_language: Python
catchphrase: "Copilote IA pour calls clients : transcription dual-channel + coaching tactique en temps réel"
---

## Description

Copilote IA pour les calls clients. Audio WASAPI dual-channel (micro + loopback système) pour une attribution parfaite des locuteurs sans diarisation IA. GPT-4o transcrit le client en temps réel, Faster-Whisper local gère le micro gratuitement. Filtre VAD Silero pour prévenir les hallucinations sur bruit et silences.

## Description Technique

Claude Opus analyse de façon incrémentale — sans jamais régénérer les notes depuis zéro, uniquement des éditions chirurgicales (opérations add/remove/set sur notes structurées). Canal de conseils séparé pour coaching tactique en direct pendant le call. UI Flutter Desktop 3 panneaux : flash vert sur champs modifiés, vumètres audio, archivage complet en JSONL. Résolution des bugs drivers AMD WASAPI, corruption DirectSound et loopback zero-frame.

## Description Fonctionnelle

Outil de productivité pour les freelances et commerciaux : prise de notes automatique pendant les calls clients, suggestions tactiques en temps réel, archivage complet des sessions. Construit et déployé en 4 jours, utilisé quotidiennement en production.

## Stats

- **7 059** lignes de code
- **~0,87€** par heure d'utilisation (GPT-4o + Claude)
- **4 jours** de build
- **14** fichiers de tests
