---
title: "IndeedOutreach"
category: ai
technologies: [Python, Playwright, Claude API, Skyvern, SQLite, SMTP, asyncio]
featured: true
in_progress: false
github_language: Python
catchphrase: "Pipeline de prospection freelance automatisé : scraping → scoring IA → candidature auto"
---

## Description

Un freelance traite réalistement 10-20 candidatures par jour avec une prospection de qualité. Ce pipeline gère l'entonnoir complet automatiquement : scraping des offres, scoring IA, recherche d'emails via vérification SMTP, génération d'emails froids personnalisés avec Claude, et auto-candidature via un agent browser vision qui remplit les formulaires comme un humain.

## Description Technique

Architecture dual-mode : emails froids avec relances automatiques J+7 et J+14, plus candidatures browser avec CV générés par IA par offre. Chaque candidature reçoit un CV unique avec sélection de thème et réécriture des expériences via LLM, plus une lettre enrichie par recherche d'entreprise en temps réel. Bypass Cloudflare via profils browser persistants. Vérification SMTP sans API payante. Remplissage de formulaires par vision Skyvern. Architecture ABC modulaire.

## Description Fonctionnelle

Automatisation complète de la prospection freelance : de la découverte d'offres jusqu'à l'envoi de la candidature personnalisée. Deux modes : outreach email froid avec CV attaché, ou candidature directe sur la plateforme via agent browser IA.

## Stats

- **2 544** lignes Python
- **22** modules
- **40+** profils de compétences
- **2** modes d'outreach (email + browser)
