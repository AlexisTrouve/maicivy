# API Requirements - Letters Frontend

**Date:** 2025-12-08
**Frontend Phase:** 3 - IA Lettres

Ce document liste les endpoints API que le frontend Letters attend du backend.

---

## 🔌 Endpoints Requis

### 1. POST /api/v1/letters/generate

**Description:** Génère une lettre de motivation et une anti-motivation pour une entreprise donnée.

**Request:**
```json
{
  "companyName": "string" // min: 2, max: 100, regex: ^[a-zA-Z0-9\s\-&.,'À-ÿ]+$
}
```

**Response (200 OK):**
```json
{
  "id": "uuid",
  "companyName": "string",
  "motivationLetter": "string", // contenu texte formaté
  "antiMotivationLetter": "string", // contenu texte formaté
  "companyInfo": {
    "industry": "string (optionnel)",
    "description": "string (optionnel)",
    "website": "string (optionnel)",
    "size": "string (optionnel)",
    "location": "string (optionnel)"
  },
  "createdAt": "ISO 8601 timestamp"
}
```

**Errors:**
- `403 Forbidden`: Visiteur n'a pas accès (< 3 visites)
  ```json
  {
    "success": false,
    "message": "Access denied. Visit the site 3 times to unlock.",
    "code": "ACCESS_DENIED"
  }
  ```

- `429 Too Many Requests`: Rate limit atteint
  ```json
  {
    "success": false,
    "message": "Rate limit exceeded. Try again in X minutes.",
    "code": "RATE_LIMIT_EXCEEDED"
  }
  ```
  Headers: `Retry-After: 120` (secondes)

- `500 Internal Server Error`: Erreur IA ou serveur
  ```json
  {
    "success": false,
    "message": "Failed to generate letters",
    "code": "GENERATION_FAILED"
  }
  ```

**Comportement attendu:**
- Durée: ~30-60 secondes (génération IA)
- Timeout frontend: 60 secondes
- Cookies de session requis (credentials: include)

---

### 2. GET /api/v1/visitors/check

**Description:** Vérifie le statut du visiteur (compteur visites, accès fonctionnalités).

**Request:**
- Headers: Cookie avec session ID

**Response (200 OK):**
```json
{
  "visitCount": 3, // nombre de visites
  "hasAccess": true, // true si visitCount >= 3 OU profil détecté
  "profileDetected": "recruiter" | "tech_lead" | null, // détection profil
  "remainingVisits": 0, // 3 - visitCount (si < 3)
  "sessionId": "uuid"
}
```

**Errors:**
- `500 Internal Server Error`: Erreur Redis/DB
  ```json
  {
    "success": false,
    "message": "Failed to check visitor status",
    "code": "CHECK_FAILED"
  }
  ```

**Comportement attendu:**
- Durée: < 100ms (lecture Redis)
- Pas de rate limiting sur ce endpoint
- Crée session si première visite

---

### 3. GET /api/v1/letters/:id/pdf

**Description:** Télécharge le PDF d'une lettre générée.

**Request:**
- Params: `id` (UUID de la lettre)
- Query: `type=motivation|anti|both`

**Response (200 OK):**
- Content-Type: `application/pdf`
- Content-Disposition: `attachment; filename="lettre-{companyName}-{type}.pdf"`
- Body: Binary PDF data

**Errors:**
- `404 Not Found`: Lettre introuvable
  ```json
  {
    "success": false,
    "message": "Letter not found",
    "code": "NOT_FOUND"
  }
  ```

- `500 Internal Server Error`: Erreur génération PDF
  ```json
  {
    "success": false,
    "message": "Failed to generate PDF",
    "code": "PDF_GENERATION_FAILED"
  }
  ```

**Comportement attendu:**
- Durée: ~2-5 secondes (génération PDF)
- Taille fichier: ~50-200 KB
- Format: A4, marges standard

---

### 4. GET /api/v1/letters/:id (Optionnel)

**Description:** Récupère une lettre générée précédemment.

**Request:**
- Params: `id` (UUID de la lettre)

**Response (200 OK):**
```json
{
  "id": "uuid",
  "companyName": "string",
  "motivationLetter": "string",
  "antiMotivationLetter": "string",
  "companyInfo": { ... },
  "createdAt": "ISO 8601"
}
```

**Errors:**
- `404 Not Found`: Lettre introuvable

---

## 🔐 Authentification / Session

**Mécanisme:**
- Cookies HTTP-only pour session ID
- Pas d'authentification utilisateur (anonymous)
- Tracking par `visitor_session` cookie

**Cookies attendus:**
```
visitor_session: uuid
  - Path: /
  - HttpOnly: true
  - SameSite: Lax
  - Secure: true (production)
  - Max-Age: 30 jours
```

**Headers requis par frontend:**
```
credentials: 'include' // sur tous les appels fetch/axios
```

---

## 📊 Rate Limiting

**Règles attendues:**

1. **Génération lettres:**
   - Max: 5 générations / jour / session
   - Cooldown: 2 minutes entre générations
   - Key Redis: `ratelimit:letters:{sessionId}`

2. **Check visiteur:**
   - Pas de rate limit (lecture seule)

3. **Download PDF:**
   - Max: 10 downloads / heure / session
   - Key Redis: `ratelimit:pdf:{sessionId}`

**Headers de réponse (429):**
```
Retry-After: 120 // secondes avant retry
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1702123456 // timestamp
```

---

## 🎯 Tracking Visiteurs

**Règles métier:**

1. **Compteur visites:**
   - Incrémenté à chaque visite de la homepage
   - Stocké dans Redis: `visitor:{sessionId}:count`
   - Expiré après 30 jours

2. **Accès IA:**
   ```
   IF visitCount >= 3 OR profileDetected IN ['recruiter', 'tech_lead', 'cto']
     THEN hasAccess = true
   ELSE
     THEN hasAccess = false
   ```

3. **Détection profil (optionnel Phase 4):**
   - User-Agent analysis
   - IP lookup (Clearbit API)
   - LinkedIn referrer detection

---

## 🧪 Tests Backend Requis

### Endpoint /api/v1/letters/generate

```bash
# Test 1: Génération réussie
curl -X POST http://localhost:8080/api/v1/letters/generate \
  -H "Content-Type: application/json" \
  -H "Cookie: visitor_session=test-uuid" \
  -d '{"companyName":"Google"}'

# Attendu: 200 OK + JSON avec id, motivationLetter, antiMotivationLetter

# Test 2: Accès refusé (< 3 visites)
curl -X POST http://localhost:8080/api/v1/letters/generate \
  -H "Content-Type: application/json" \
  -H "Cookie: visitor_session=new-uuid" \
  -d '{"companyName":"Google"}'

# Attendu: 403 Forbidden + JSON avec code ACCESS_DENIED

# Test 3: Rate limit atteint
curl -X POST http://localhost:8080/api/v1/letters/generate \
  -H "Content-Type: application/json" \
  -H "Cookie: visitor_session=test-uuid" \
  -d '{"companyName":"Google"}'
# (répéter 6 fois rapidement)

# Attendu: 429 Too Many Requests + Header Retry-After

# Test 4: Validation input (nom trop court)
curl -X POST http://localhost:8080/api/v1/letters/generate \
  -H "Content-Type: application/json" \
  -d '{"companyName":"G"}'

# Attendu: 400 Bad Request + JSON avec erreur validation
```

### Endpoint /api/v1/visitors/check

```bash
# Test 1: Première visite
curl http://localhost:8080/api/v1/visitors/check

# Attendu: 200 OK + visitCount=0, hasAccess=false, remainingVisits=3

# Test 2: Troisième visite
curl http://localhost:8080/api/v1/visitors/check \
  -H "Cookie: visitor_session=test-uuid-with-3-visits"

# Attendu: 200 OK + visitCount=3, hasAccess=true, remainingVisits=0
```

### Endpoint /api/v1/letters/:id/pdf

```bash
# Test 1: Download PDF motivation
curl http://localhost:8080/api/v1/letters/test-uuid/pdf?type=motivation \
  -H "Cookie: visitor_session=test-uuid" \
  --output lettre-motivation.pdf

# Attendu: 200 OK + fichier PDF valide

# Test 2: Download PDF dual
curl http://localhost:8080/api/v1/letters/test-uuid/pdf?type=both \
  --output lettre-dual.pdf

# Attendu: 200 OK + fichier PDF avec 2 pages

# Test 3: Lettre introuvable
curl http://localhost:8080/api/v1/letters/invalid-uuid/pdf?type=motivation

# Attendu: 404 Not Found
```

---

## 🚨 Points Critiques

### 1. Timeout
**Problème:** Génération IA prend 30-60s
**Solution backend:**
- Utiliser contexte avec timeout 90s
- Retourner erreur 504 si timeout
- Implémenter queue asynchrone (optionnel)

### 2. Coûts API IA
**Problème:** Chaque génération coûte $0.01-0.10
**Solution backend:**
- Rate limiting strict (5/jour)
- Cooldown 2 minutes
- Cache Redis des résultats (1 heure)

### 3. PDF Size
**Problème:** PDFs lourds peuvent ralentir download
**Solution backend:**
- Compression PDF
- Limiter taille lettres (max 2000 mots)
- Streaming response (chunked)

### 4. Session Hijacking
**Problème:** Cookie session peut être volé
**Solution backend:**
- HttpOnly cookies
- SameSite=Lax minimum
- HTTPS en production
- Expiration 30 jours max

---

## 📝 Dépendances Backend Attendues

### Document 08: BACKEND_AI_SERVICES.md
**Fourni:**
- Service de génération lettres (Claude/GPT-4)
- Scraper infos entreprises
- Génération PDF

### Document 09: BACKEND_LETTERS_API.md
**Fourni:**
- POST /api/v1/letters/generate
- GET /api/v1/letters/:id/pdf
- Queue asynchrone (jobs)
- Rate limiting middleware

### Document 04: BACKEND_MIDDLEWARES.md
**Fourni:**
- Tracking visiteurs (middleware)
- Rate limiting (middleware)
- CORS

---

## ✅ Validation Checklist Backend

Avant d'intégrer avec le frontend, vérifier:

- [ ] Endpoint `/api/v1/letters/generate` fonctionne
- [ ] Endpoint `/api/v1/visitors/check` fonctionne
- [ ] Endpoint `/api/v1/letters/:id/pdf` fonctionne
- [ ] Cookies de session créés automatiquement
- [ ] Rate limiting actif (5 générations/jour)
- [ ] Cooldown 2 minutes entre générations
- [ ] Validation input (Zod ou équivalent)
- [ ] Error responses conformes au format JSON
- [ ] Timeout génération IA configuré (90s)
- [ ] PDFs générés valides (ouvrables)
- [ ] CORS configuré pour frontend (credentials: true)
- [ ] Tests E2E passent

---

**Auteur:** Claude (Agent IA)
**Date:** 2025-12-08
**Version:** 1.0
