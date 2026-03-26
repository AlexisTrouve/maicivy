---
title: "Grimorium"
category: ai
technologies: [Python, FastAPI, ChromaDB, Claude API, MCP, Docker, sentence-transformers, Qwen]
featured: true
in_progress: false
github_language: Python
catchphrase: "Archiviste IA RAG : bibliothèque de scriptwriting connectée à Claude via MCP"
links: []
---

## Description

Les auteurs utilisent Claude pour développer des histoires et NotebookLM pour consulter des livres de théorie — les deux outils ne communiquent pas. Grimorium résout ça avec un agent archiviste qui vit directement dans Claude : posez une question narrative, il cherche dans toute votre bibliothèque de scriptwriting et retourne les passages pertinents avec leurs sources.

## Description Technique

Pipeline d'ingestion PDF en deux passes : pdfplumber pour l'extraction + Qwen3 14b pour la structure narrative + Haiku pour l'extraction de concepts. Stockage vectoriel ChromaDB avec embeddings sentence-transformers, recherche sémantique (pas par mots-clés). Agent archiviste Claude Haiku (~7k tokens de contexte) exposé via serveur MCP HTTP/SSE. Metadata livres/scripts en SQLite. Déploiement Docker Compose sur VPS.

## Description Fonctionnelle

Outil de développement narratif pour auteurs et scénaristes. Recherche sémantique dans une bibliothèque de livres de scriptwriting, retour des passages pertinents avec sources exactes, directement accessible depuis Claude Code ou tout client MCP. Développé pour Tuco Benedicto Entertainment.
