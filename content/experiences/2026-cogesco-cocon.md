---
title: "Développeur Système SEO — CoconSystem"
company: "Cogesco"
category: backend
start_date: 2026-02
end_date: 2026-03
technologies: [Rust, Actix-web, VBA, Access, WordPress, Claude API, Docker, Prometheus]
tags: [backend, automation, ai, api, rust, seo]
featured: true
catchphrase: "43 microservices Rust pour piloter un système de cocons sémantiques SEO"
---

## Description

Conception et développement d'un système complet de gestion de cocons sémantiques pour Cogesco. Architecture workspace Cargo : gateway Actix-web + 10 familles de microservices (43 modules) orchestrés depuis l'interface VBA/Access existante. Couvre l'intégralité du cycle SEO : stratégie mots-clés → génération contenu IA → scoring → publication WordPress.

## Description Technique

Gateway HTTP Actix-web avec rate limiting (governor), métriques Prometheus et proxy vers les microservices. 43 services Rust spécialisés (tokio async) : génération de contenu via Claude API (Anthropic), analyse et scoring SEO, publication vers WordPress/WooCommerce via REST API. Bases Access satellites par cocon (Cockpit_Master.accdb + Cocon_xxx.accdb) pilotées depuis VBA. Déploiement Docker multi-conteneurs.

## Description Fonctionnelle

Cockpit de pilotage SEO permettant à Cogesco de planifier des cocons sémantiques (pages N1/N2/N3/N4), générer automatiquement du contenu optimisé, mesurer la couverture par mot-clé et publier vers WordPress — le tout piloté depuis l'interface Access déjà en place, sans rupture de workflow pour les équipes.
