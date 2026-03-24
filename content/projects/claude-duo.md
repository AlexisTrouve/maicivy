---
title: "ClaudeDuo"
category: devops
technologies: [Node.js, Express, SQLite, MCP SDK, Long-Polling]
featured: true
in_progress: true
github_language: JavaScript
catchphrase: "Serveur de messagerie temps réel pour coordination multi-agents Claude via MCP"
---

## Description

À mesure que les workflows de développement IA passent au-delà d'un seul agent, le besoin de communication inter-agents devient critique. ClaudeDuo est un serveur de messagerie temps réel conçu pour l'écosystème MCP — permettant à plusieurs instances d'agents Claude de coordonner des tâches, partager du contexte et collaborer sur différentes codebases sans intervention humaine.

## Description Technique

Architecture deux niveaux : un broker HTTP central (Express + SQLite) associé à des MCP partner servers tournant dans chaque instance d'agent. Les agents s'enregistrent, échangent des clés cryptographiques, et communiquent via messages directs ou conversations de groupe. Long-polling avec timeouts configurables pour livraison instantanée. File d'attente pour les partenaires hors-ligne. Authentification Bearer token, nettoyage automatique des partenaires inactifs.

## Description Fonctionnelle

Infrastructure de collaboration pour workflows multi-agents IA. Utilisé quotidiennement en production pour des workflows de développement réels. Permet à plusieurs Claude Code d'explorer leurs codebases respectives, s'échanger des informations techniques et coordonner des changements cross-repo.

## Stats

- **2 027** lignes de code
- **13** outils MCP
- **14** endpoints REST
- **v2.0** en production active
