# maicivy

**An AI-native CV and portfolio platform, running in production: an interactive map of real projects, a tool-using chat agent, and a live multi-agent run that builds a tailored project dossier.**

[![CI](https://github.com/AlexisTrouve/maicivy/actions/workflows/ci.yml/badge.svg)](https://github.com/AlexisTrouve/maicivy/actions/workflows/ci.yml)
[![Security](https://github.com/AlexisTrouve/maicivy/actions/workflows/security.yml/badge.svg)](https://github.com/AlexisTrouve/maicivy/actions/workflows/security.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Live site: [maicivy.etheryale.com](https://maicivy.etheryale.com)**

Built and operated by **Alexis Trouvé**, AI Orchestration Engineer ("Technomancer"), specialised in AI agents and multi-agent orchestration in production.

---

## What it is

maicivy replaces the static CV with a working system. Visitors explore projects, ask an agent questions answered from live data, or describe a need and watch a team of agents assemble a dossier from real project evidence. The same codebase serves the owner's private tooling.

It is a full-stack product: a Next.js front end, a Go API, PostgreSQL and Redis, and AI calls routed through an Anthropic-compatible gateway. Profile, project and blog content is served by a separate content service, maiProFiles.

## Key features

### Multi-agent and AI

- **Live dossier** (`/dossier-vivant`): a visitor describes a need; an orchestrator dispatches four sub-agents that run concurrently (need reader, evidence researcher, architect, writer). The architect delegates the architecture diagram to a dedicated diagrammer agent through a tool call. Progress streams to the browser over SSE, and the finished dossier can be exported as PDF.
- **Chat agent** (`/chat`): streamed over SSE, with tool use against the live profile (profile lookup, project search, blog articles, stats, follow-up suggestions). Answers are paired with structured cards in a side panel.
- **Model fallback**: a failed completion is replayed on a configured fallback model through the same gateway; token usage and cost are tracked per model.

### Public site

- **The Mountain** (home): projects laid out as a relief, from flagship work at the top to foundations at the bottom, linked by real "used to build" relations. Server-rendered HTML and SVG, usable without JavaScript; an optional 3D network view (three.js) loads on demand.
- **CV** (`/cv`): themed views (AI, backend, C++, artistic, full-stack, DevOps) and PDF export rendered by headless Chromium with an ATS-friendly text layer.
- **CV tailored to a job offer**: paste an offer (text or URL) and receive a PDF adapted to it, in the requested language (`POST /api/v1/cv/generate`, rate-limited).
- **Portfolio** (`/portfolio`, `/portfolio/[id]`) and **blog** with an RSS feed and an email newsletter sent on publication.
- **Git stats** (`/gitstats`): commit activity aggregated from Gitea and GitLab, served from a Redis cache with stale-while-revalidate and single-flight background refresh.
- **Analytics** (`/analytics`): public dashboard with real-time updates over WebSocket and a click heatmap.
- **Architecture** (`/architecture`): how the site itself is built.
- **Five languages** (fr, en, de, it, zh) via next-intl, dark theme.

### Owner tools

An owner-only admin panel (`/admin`, not indexed) groups the day-to-day tooling: CV from a job offer, a message composer agent, a mailbox with relevance filtering and translation, an agent chat, and private stats (AI cost per model, suspicious traffic).

### Platform

- **Frontdoor**: a small Go reverse proxy in front of all traffic. It scores client reputation and rate-limits before proxying `/api` to the backend and everything else to the front end.
- **Rate limits** on AI endpoints backed by Redis.
- **Security headers** (CSP and related) and signed session cookies.
- **Observability**: structured JSON logs (zerolog), Prometheus metrics, Grafana dashboards (`monitoring/`).
- **OpenAPI** spec served by the backend at `/api/docs` (source: [`backend/docs/api/openapi.yaml`](backend/docs/api/openapi.yaml)).

## Architecture

```
Browser
   |
Cloudflare (CDN, WAF)
   |
nginx (TLS termination)
   |
Frontdoor (Go: reputation scoring, rate limiting)
   |
   |-- /api/* ------------> Go API (Fiber)
   |                          |-- PostgreSQL 16 (GORM)
   |                          |-- Redis 7 (cache, sessions, rate limits)
   |                          |-- AI gateway (Anthropic-compatible, primary + fallback model)
   |                          |-- maiProFiles (profile, projects, blog content)
   |                          |-- Gitea / GitLab APIs (git stats)
   |                          '-- Headless Chromium (PDF rendering)
   |
   '-- everything else ---> Next.js 15 front end (server-rendered, calls the Go API)
```

The application services run as containers under Docker Compose and bind to localhost only; nginx on the host is the single public entry point.

## Tech stack

| Layer | Technologies |
|-------|--------------|
| Front end | Next.js 15, React 19, TypeScript 5, Tailwind CSS, shadcn/ui, Framer Motion, three.js, Recharts, next-intl |
| Back end | Go 1.26, Fiber, GORM, go-redis, Anthropic Go SDK, chromedp, zerolog |
| Data | PostgreSQL 16, Redis 7 |
| AI | Anthropic-compatible gateway with primary and fallback models, tool use, SSE streaming |
| Infrastructure | Docker Compose, nginx, Cloudflare, Prometheus, Grafana |
| Testing | Go `testing` + testify + miniredis, Jest + Testing Library, Playwright, k6 / Artillery |

## Quick start

Requirements: Docker with Docker Compose. For local development without containers: Go 1.26 and Node.js 20.

```bash
git clone https://github.com/AlexisTrouve/maicivy.git
cd maicivy
cp .env.example .env        # then fill in the values (see below)
docker compose up -d --build
```

Services, bound to localhost:

| Service | URL |
|---------|-----|
| Front end | http://localhost:3000 |
| API | http://localhost:8080 (Swagger UI at `/api/docs`) |
| Frontdoor | http://localhost:8090 |

Configuration lives in `.env` (template: [`.env.example`](.env.example)). The variables that matter most:

- `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `REDIS_PASSWORD`: databases.
- `SESSION_SECRET`, `JWT_SECRET`, `RATE_LIMIT_SECRET`: signing secrets.
- `CLAUDE_API_KEY`, `ANTHROPIC_BASE_URL`, `AI_FALLBACK_MODEL`: AI gateway and fallback model.
- `MAIPROFILES_API_KEY`: access to the content service. Without it, profile and project content is unavailable.
- `GITEA_STATS_*`, `GITLAB_STATS_*`: optional sources for the Git stats page.

Per-machine overrides go in `docker-compose.override.yml` (template: [`docker-compose.override.yml.example`](docker-compose.override.yml.example)).

### Running without Docker

PostgreSQL and Redis must be reachable. The API reads its configuration from the environment or from `backend/.env` (template: [`backend/.env.example`](backend/.env.example)).

```bash
# API
cd backend
make run                     # go run ./cmd/main.go

# Front end (separate terminal)
cd frontend
npm ci
npm run dev                  # http://localhost:3000
```

The front end calls the API at `NEXT_PUBLIC_API_URL` (default `http://localhost:8080`).

## Testing

```bash
# Back end: unit tests (miniredis, no database or Redis required)
cd backend && go test -short ./...

# Front end: unit tests, type checking, lint
cd frontend && npx jest
cd frontend && npm run type-check && npm run lint

# End-to-end (Playwright, specs in frontend/tests/e2e)
cd frontend && npx playwright install && npm run test:e2e
```

Playwright starts the dev server itself, or targets an existing deployment when `PLAYWRIGHT_TEST_BASE_URL` is set. Load-test scenarios (k6, Artillery) live in `load-tests/`.

## Project structure

```
maicivy/
├── backend/            Go API
│   ├── cmd/            entry points: API server, frontdoor proxy
│   ├── internal/       handlers (api), services, multi-agent run (dossiervivant),
│   │                   middleware, models, cache, workers
│   ├── migrations/     SQL migrations
│   └── templates/      HTML templates for PDF rendering
├── frontend/           Next.js app
│   ├── app/            routes ([locale]/... public pages, admin/ owner panel)
│   ├── components/     UI by domain (montagne, chat, cv, dossier-vivant, analytics...)
│   ├── messages/       translations (fr, en, de, it, zh)
│   └── tests/e2e/      Playwright specs
├── docs/               API reference and design notes
├── monitoring/         Prometheus and Grafana configuration
├── nginx/              reverse proxy configuration
├── load-tests/         k6 and Artillery scenarios
├── scripts/            backup, restore, health checks
└── docker-compose.yml  full stack: postgres, redis, backend, frontend, frontdoor
```

## License

Released under the [MIT License](LICENSE). Copyright (c) 2025 Alexis Trouvé.
