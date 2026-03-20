---
title: "Etheryale"
category: ai
technologies: [Node.js, Express, SQLite, Stripe, Flutter, Claude API, JWT]
featured: true
in_progress: true
github_language: JavaScript
catchphrase: "Plateforme de revente de tokens Claude API avec billing multi-modèle"
---

## Description

Backend d'une plateforme de revente d'accès à l'API Claude (Anthropic). Trois systèmes de facturation isolés : gacha (packs de tokens one-shot), prepaid (crédits développeur consommés par appel API) et subscription (abonnement mensuel récurrent). Gestion d'API keys multi-environnement (dev/prod/staging), tracking d'usage par requête (tokens, latence, modèle), rate limiting par utilisateur et IP. Paiements Stripe (PaymentIntent + Subscriptions), système de referral avec commissions, dashboard admin Flutter Web. Proxy intelligent avec rotation de comptes Claude et refresh automatique de tokens.
