---
title: "OpenClaw Infrastructure"
category: ai
technologies: [Node.js, Express, Electron, Claude API, OpenAI API, SQLite, systemd, Docker]
featured: true
in_progress: true
github_language: JavaScript
catchphrase: "Infrastructure multi-bot Claude : proxy API, installer desktop, licence server"
---

## Description

Infrastructure de production pour une flotte de 6 bots IA Discord connectés à Claude (Anthropic) via un proxy OpenAI-to-Anthropic custom. Proxy Express.js qui convertit les requêtes OpenAI format vers l'API Anthropic Messages, avec streaming SSE, routing par clé API et rotation de comptes. Installer desktop cross-platform (ClawKit) en Electron avec obfuscation bytecode V8, licence server SQLite, fingerprinting device et auto-update. Déploiement VPS avec services systemd isolés par bot, monitoring et détection de bans automatisée.
