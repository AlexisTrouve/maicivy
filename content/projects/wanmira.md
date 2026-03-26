---
title: "WanMira"
category: ai
technologies: [Python, Claude API, Gitea API, ProjectMind, Tweepy, LinkedIn API]
featured: true
in_progress: true
github_language: Python
catchphrase: "Pipeline automatisé : commits Git → articles blog + tweets + posts LinkedIn via Claude"
---

## Description

Pipeline de content automation qui transforme le travail de développement en présence publique. Collecte l'activité depuis Gitea et ProjectMind, analyse les patterns localement (zéro LLM à cette étape), puis génère du contenu via un pipeline Claude trois stages avant de publier sur blog, X et LinkedIn.

## Description Technique

Architecture multi-stage inspirée de Demiurgos : Stage 1 Grounding (Haiku, ~$0.001) valide les faits et propose la structure, Stage 2 Génération (Sonnet, ~$0.05) produit JSON structuré avec 5+ few-shot exemples et liste de mots interdits, Stage 3 Validation locale vérifie longueur tweets, cohérence factuelle, structure JSON. Filtre de confidentialité : exclusion automatique des tags privés (awen, proxy, credentials) et redaction des IPs/tokens. Tweets schedulés via JSON + cron horaire.

## Description Fonctionnelle

Automatisation complète de la présence publique d'un développeur : chaque cycle génère un article blog (EN, 800-1500 mots), 5-7 tweets/semaine, et 1-2 posts LinkedIn/semaine (FR, format storytelling) — sans intervention manuelle au-delà du déclenchement du pipeline.
