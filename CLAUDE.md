# maicivy - Instructions Claude

## Workflows
- [Sync GitHub](.claude/SYNC_GITHUB.md) - Synchroniser vers le repo public GitHub

## Deploy

Le CI/CD GitHub Actions **ne sert pas au deploy**. On deploy manuellement via SSH :

```bash
ssh debian@57.131.33.10 "cd ~/maicivy && git fetch origin main && git reset --hard origin/main && docker compose up -d --build"
```

Le repo git sur le VPS est `~/maicivy` (pas `/opt/maicivy`).
La commande docker est `docker compose` (v2, pas `docker-compose`).

## Architecture stealth ATS

Le PDF injecte des termes ATS invisibles via deux mécanismes :

1. **`.stealth` div** (sous chaque exp-card) : `position:absolute; z-index:0` couvert par la card (`z-index:1; background:white`). Couleur `#0f172a` — texte sombre sur fond sombre, pas flaggé "white text trick".

2. **Header stealth** (`<span>` dans `.header`) : `universalATSTerms` + skills tailorés injectés dans le header sombre (`background:#0f172a`), même couleur `#0f172a`. Invisible, 0 contrast, ATS-safe.

Règle Chromium PDF : `color:#ffffff` exact = texte supprimé du stream. `opacity<1` = ToUnicode corrompu. Couleur foncée sur fond foncé = encodé correctement.

## API — Endpoints principaux

Base URL prod : `https://maicivy.etheryale.com/api/v1`

### Auth / Session
- Cookie `maicivy_session` requis sur tous les endpoints IA
- Header `X-Owner-Key: <MAICIVY_OWNER_API_KEY>` → bypass rate limit + modèle Opus
- Obtenir une session : `curl -si https://maicivy.etheryale.com/api/v1/` → `Set-Cookie`

### Lettres (queue async)
```
POST /letters/generate       → enqueue job, retourne { job_id }
GET  /letters/job/:jobId     → status + progress (polling)
GET  /letters/:id            → lettre par ID
GET  /letters/:id/pdf        → télécharger PDF
GET  /letters/pair?company=X → paire motivation + anti-motivation
GET  /letters/history        → historique paginé
GET  /letters/access/status  → accès IA du visiteur
GET  /letters/ratelimit/status → compteurs rate limit
```

Body `POST /letters/generate` :
```json
{ "company_name": "Stripe", "lang": "fr", "job_offer": "..." }
```

### CV — Export PDF
```
GET /cv/export?theme=fullstack&lang=fr  → CV standard en PDF (thème fixe)
POST /cv/generate                       → CV tailored depuis une offre d'emploi, retourne PDF
```

**`GET /cv/export`** — paramètres :
- `theme` : `fullstack` (défaut) | `backend` | `cpp` | `artistique` | `devops`
- `lang` : `fr` (défaut) | `en`

**`POST /cv/generate`** — body JSON :
```json
{ "offer": "texte brut de l'offre ou URL", "lang": "fr", "format": "pdf" }
```
Retourne directement le PDF binaire. Sans `"format": "pdf"` → retourne du JSON (AdaptiveCVResponse).

```bash
curl -X POST https://maicivy.etheryale.com/api/v1/cv/generate \
  -H "Content-Type: application/json" \
  -H "X-Owner-Key: <MAICIVY_OWNER_API_KEY>" \
  -d '{"offer": "texte offre", "lang": "fr", "format": "pdf"}' \
  --output CV.pdf
```

**Note infra** : PDF généré via headless-shell 123 (`/opt/headless-shell/headless-shell`). Alpine/Debian auto-update avait tiré Chromium 146 qui crashe sur le kernel VPS — résolu en copiant le binaire depuis `chromedp/headless-shell:123.0.6312.58`.

### Messages plateforme (sync)
```
POST /messages/generate → retourne le message directement (< 5s)
```

Body :
```json
{ "mission": "...", "platform": "malt|linkedin|upwork", "tjm": 250, "lang": "fr" }
```

### Rate limiting
- Visiteurs : 5 générations/jour par session, 3/jour par IP, cooldown 2min
- Owner (X-Owner-Key) : illimité, modèle Opus
- Flush IP en dev : `ssh debian@57.131.33.10 "docker exec maicivy-redis redis-cli DEL 'ratelimit:ai:ip:172.18.0.1:daily'"`

### Modèles
- Visiteurs → `claude-haiku-4-5-20251001` (prompt simplifié)
- Owner → `claude-opus-4-6` (prompt v2 complet avec few-shot)
- Switch prompt sans rebuild : env var `PROMPT_VERSION=v1|v2` + restart backend
## APIs externes

**maiProFiles** — profil + projets : `https://maiprofiles.etheryale.com` — voir `docs/maiprofiles_api.md`
