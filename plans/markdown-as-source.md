# Plan : Markdown comme source de donnees CV

## Probleme

Les donnees CV (experiences, skills, projets) sont en dur dans PostgreSQL via `seed_data.sql`. Mettre a jour le CV = modifier du SQL + re-seeder. Les fichiers `.md` du profil freelance sont plus riches, plus a jour, et plus faciles a maintenir.

## Objectif

Les fichiers Markdown deviennent la **single source of truth** pour toutes les donnees CV. Le site web, le scoring par theme, et l'export PDF consomment tous le meme pipeline : `.md` -> parser -> modeles Go -> API/PDF.

## Architecture cible

```
content/                          # NOUVEAU - source markdown
  experiences/
    2024-indeed-outreach.md       # Une experience = un fichier
    2023-maicivy.md
    2020-moteur3d.md
    ...
  skills.md                       # Toutes les skills dans un fichier
  projects/
    maicivy.md
    indeed-outreach.md
    ...
  profile.md                      # Profil general (apercu, dispo, engagement)

backend/
  internal/
    content/                      # NOUVEAU - parser markdown
      parser.go                   # Parse les .md en modeles Go
      loader.go                   # Charge tous les fichiers au demarrage
    services/
      cv_service.go               # MODIFIE - lit depuis le content loader au lieu de la DB
      cv_scoring.go               # INCHANGE - le scoring par theme fonctionne tel quel
      pdf_service.go              # INCHANGE - recoit les memes structs
```

## Format Markdown des experiences

Chaque fichier experience utilise un frontmatter YAML :

```markdown
---
title: "Developpeur Full-Stack Freelance"
company: "Indeed Outreach Bot"
category: backend
start_date: 2025-01
end_date: null
technologies: [python, sqlite, claude-api, camoufox, browser-use]
tags: [backend, automation, scraping, ai, api]
featured: true
---

## Description
Bot de lead gen freelance base sur les offres Indeed.
Signal : boite qui cherche un dev = budget ouvert + besoin actif.

## Description Technique
Pipeline Python : scrape Indeed (Camoufox) -> scoring AI (Claude Haiku)
-> enrichissement contact (Clearbit + SMTP verify) -> email personnalise
-> candidature automatique (browser-use + Patchright)

## Description Fonctionnelle
Automatisation complete du cycle de prospection freelance,
du sourcing d'offres a l'envoi de candidatures personnalisees.
```

## Format Markdown des skills

```markdown
# Skills

## Expert
- Python | backend, automation | 10 | Automatisation, APIs, scraping, AI integration
- Node.js | backend, fullstack | 8 | Express, APIs REST, microservices
- C/C++ | cpp, systems | 10 | Moteurs 3D, outils bas niveau, performance

## Advanced
- TypeScript | fullstack, frontend | 5 | React, Next.js, type safety
- Docker | devops | 4 | Conteneurisation, compose, CI/CD
- PostgreSQL | backend, database | 6 | Modelisation, requetes complexes, optimisation

## Intermediate
- Rust | backend, systems | 2 | CLI tools, performance critique
- Kubernetes | devops | 2 | Orchestration, deployments
```

Format par ligne : `- Nom | tags (csv) | annees | description`

## Phases

### Phase 1 : Parser Markdown (backend)
- [ ] Creer `backend/internal/content/parser.go`
  - Parser frontmatter YAML avec `gopkg.in/yaml.v3`
  - Parser body markdown (descriptions par section)
  - Mapper vers les structs existantes (`models.Experience`, `models.Skill`, `models.Project`)
- [ ] Creer `backend/internal/content/loader.go`
  - Charger tous les `.md` d'un repertoire
  - Tri par date, gestion i18n (fichier `.en.md` pour la version anglaise)
  - Cache en memoire, rechargement a chaud optionnel (watch file changes)

### Phase 2 : Creer les fichiers Markdown
- [ ] Migrer `seed_data.sql` -> fichiers `.md` dans `content/`
  - Extraire chaque experience en fichier separe
  - Consolider les skills dans `skills.md`
  - Extraire les projets en fichiers separes
- [ ] Enrichir avec le contenu de `profile.md` d'indeed-outreach et autres sources

### Phase 3 : Brancher le CV Service sur le content loader
- [ ] Modifier `cv_service.go` : remplacer les queries DB par des appels au content loader
- [ ] Le scoring (`cv_scoring.go`) reste inchange — il recoit les memes structs avec tags/technologies
- [ ] Le PDF (`pdf_service.go`) reste inchange — il recoit les memes structs scorees
- [ ] L'API reste inchangee — memes endpoints, memes responses JSON

### Phase 4 : Frontend (optionnel, si necessaire)
- [ ] Verifier que le frontend n'a pas de dependance directe au format DB
- [ ] Le frontend consomme l'API, donc normalement zero changement

### Phase 5 : Cleanup
- [ ] Retirer les seeds SQL (ou les garder comme fallback)
- [ ] Documenter le format markdown dans un README
- [ ] Ajouter un CLI `go run cmd/validate-content` pour valider les .md

## Points cles

1. **Zero changement sur le scoring** — le systeme de tags/poids par theme fonctionne identiquement, il recoit juste les donnees d'une source differente
2. **Zero changement sur le PDF** — le template HTML + chromedp recoit les memes structs
3. **Zero changement sur le frontend** — l'API retourne les memes JSON
4. **i18n** — convention `experience.md` (fr) + `experience.en.md` (en), le loader charge les deux
5. **Hot reload en dev** — le loader peut watcher le dossier `content/` et recharger sans restart
6. **Le format markdown est lisible par les humains** — pas besoin d'ouvrir une DB pour editer son CV
7. **Git-friendly** — les diffs sur les `.md` sont clairs, on voit exactement ce qui change

## Use case indeed-outreach

Une fois ce plan implemente, indeed-outreach peut :
1. Appeler `GET /api/v1/cv/export?theme=backend&format=pdf&lang=fr`
2. Recuperer le PDF genere depuis les memes `.md` que le site web
3. L'utiliser comme `files/cv.pdf` pour les candidatures browser-use

Ou plus simple : un script qui regen le PDF periodiquement.
