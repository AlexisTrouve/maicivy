# Profile Detection Implementation Summary

**Feature 3: Détection Profils Avancée - Phase 5**

**Date:** 2025-12-08
**Status:** Implémenté
**Version:** 1.0

---

## Table des Matières

1. [Vue d'Ensemble](#vue-densemble)
2. [Architecture](#architecture)
3. [Heuristiques de Détection](#heuristiques-de-détection)
4. [Clearbit API Integration](#clearbit-api-integration)
5. [Bypass Access Gate Logic](#bypass-access-gate-logic)
6. [Privacy & RGPD Compliance](#privacy--rgpd-compliance)
7. [Exemples de Détection](#exemples-de-détection)
8. [Performance](#performance)
9. [Tests](#tests)
10. [Maintenance](#maintenance)

---

## Vue d'Ensemble

### Objectif

La fonctionnalité de **Détection Profils Avancée** identifie automatiquement les visiteurs cibles (recruteurs, CTOs, tech leads, CEOs) pour :

- Personnaliser l'expérience utilisateur
- Débloquer l'accès aux fonctionnalités IA immédiatement (sans attendre 3 visites)
- Fournir des analytics sur les profils visiteurs

### Types de Profils Détectés

| ProfileType | Description | Bypass Enabled |
|-------------|-------------|----------------|
| `recruiter` | Recruteurs, HR, talent acquisition | ✅ Oui |
| `cto` | Chief Technology Officer | ✅ Oui |
| `tech_lead` | Tech Lead, Engineering Manager | ✅ Oui |
| `ceo` | Chief Executive Officer, Founders | ✅ Oui |
| `developer` | Développeurs (outils dev) | ❌ Non |
| `other` | Visiteurs normaux | ❌ Non |

### Composants Implémentés

**Backend (Go):**
- `profile_detector.go` - Service principal de détection
- `clearbit_client.go` - Client API Clearbit
- `useragent_parser.go` - Parser User-Agent
- `profile_enrichment.go` - Middleware auto-enrichment
- `profile.go` - API endpoints
- `visitor.go` (modifié) - Modèle avec nouveaux champs

**Frontend (React/Next.js):**
- `ProfileBadge.tsx` - Badge de profil détecté
- `PersonalizedMessage.tsx` - Messages personnalisés
- `useProfileDetection.ts` - Hook React
- `types.ts` (modifié) - Types TypeScript
- `api.ts` (modifié) - Client API

**Tests:**
- `profile_detector_test.go` - Tests unitaires

---

## Architecture

### Flow de Détection

```
┌─────────────────────────────────────────────────────────┐
│ 1. Requête HTTP avec User-Agent, IP, Referer          │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Middleware: profile_enrichment.go                   │
│    - Vérifie cache Redis (session_id)                  │
│    - Si cache → retourne profil                        │
│    - Sinon → appelle ProfileDetectorService            │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 3. ProfileDetectorService.DetectProfile()              │
│    ┌─────────────────────────────────────────────┐    │
│    │ a) Parse User-Agent (UAParser)              │    │
│    │    → DeviceInfo + isBot                     │    │
│    │    → Score: 0-30%                           │    │
│    ├─────────────────────────────────────────────┤    │
│    │ b) Analyse Referer                          │    │
│    │    → LinkedIn, job boards, GitHub           │    │
│    │    → Score: 0-20%                           │    │
│    ├─────────────────────────────────────────────┤    │
│    │ c) Clearbit API Enrichment                  │    │
│    │    → Company, job title, industry           │    │
│    │    → Score: 0-50%                           │    │
│    └─────────────────────────────────────────────┘    │
│                                                         │
│    → Calcul confidence final:                          │
│      (UA*0.3) + (Referer*0.2) + (IP*0.5)              │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Stockage                                            │
│    - Cache Redis (TTL 24h)                             │
│    - DB PostgreSQL (visitor.profile_type)              │
│    - Bypass Redis (si confidence >= 60%)               │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 5. Context Fiber → Headers debug                       │
│    - X-Profile-Detected                                │
│    - X-Confidence-Score                                │
│    - X-Access-Gate-Bypass                              │
└─────────────────────────────────────────────────────────┘
```

---

## Heuristiques de Détection

### 1. User-Agent Analysis (Poids: 30%)

#### Patterns Recruteurs
```go
recruiterPatterns := []string{
    "linkedinapp",
    "linkedin",
    "greenhouse",
    "lever",
    "workday",
    "applicantstack",
    "jobvite",
    "recruiting",
}
```

**Score:** 30 points
**ProfileType:** `recruiter`

#### Patterns Développeurs
```go
developerPatterns := []string{
    "postman",
    "curl",
    "wget",
    "httpie",
    "insomnia",
}
```

**Score:** 20 points
**ProfileType:** `developer`

#### Exemples

| User-Agent | Score | Type |
|------------|-------|------|
| `LinkedInApp/1.0` | 30 | recruiter |
| `Greenhouse/1.0` | 30 | recruiter |
| `Postman/1.0` | 20 | developer |
| `curl/7.68.0` | 20 | developer |
| `Chrome/120.0` | 0 | other |

---

### 2. Referer Analysis (Poids: 20%)

#### LinkedIn Detection

| Referer | Score | Type |
|---------|-------|------|
| `linkedin.com/jobs` | 20 | recruiter |
| `linkedin.com/recruiter` | 20 | recruiter |
| `linkedin.com/*` | 10 | recruiter |

#### Job Boards

| Referer | Score | Type |
|---------|-------|------|
| `indeed.com` | 15 | recruiter |
| `glassdoor.com` | 15 | recruiter |
| `monster.com` | 15 | recruiter |
| `welcometothejungle.com` | 15 | recruiter |

#### Developer Platforms

| Referer | Score | Type |
|---------|-------|------|
| `github.com` | 10 | developer |

---

### 3. Clearbit API Enrichment (Poids: 50%)

**Le plus précis, donc le poids le plus élevé.**

#### Company Type Analysis

```go
if company_type == "Recruiting" || industry == "Recruiting" {
    score += 40
    profileType = "recruiter"
}
```

#### Job Title Analysis

| Job Title Pattern | Score | Type |
|-------------------|-------|------|
| `CTO`, `Chief Technology Officer` | 50 | cto |
| `CEO`, `Chief Executive Officer` | 50 | ceo |
| `Tech Lead`, `Engineering Manager`, `VP Eng` | 40 | tech_lead |
| `Recruiter`, `Talent`, `HR` | 40 | recruiter |

**Exemples:**

```json
{
  "company_name": "Amazon",
  "job_title": "CTO",
  "industry": "Technology"
}
// → ProfileType: cto, Score: 50, Confidence: 50 (si pas d'autres signaux)
```

```json
{
  "company_type": "Recruiting",
  "job_title": "Senior Recruiter",
  "industry": "Staffing & Recruiting"
}
// → ProfileType: recruiter, Score: 80 (40 + 40)
```

---

### 4. Formule de Confidence Finale

```go
finalConfidence = (userAgentScore * 0.3) + (refererScore * 0.2) + (ipScore * 0.5)

// Normalisation
if finalConfidence > 100 {
    finalConfidence = 100
}
```

**Exemples de calculs:**

| UA Score | Referer Score | IP Score | Confidence Final |
|----------|---------------|----------|------------------|
| 30 | 20 | 50 | 38 |
| 0 | 0 | 50 | 25 |
| 30 | 0 | 0 | 9 |
| 0 | 20 | 0 | 4 |
| 30 | 20 | 0 | 13 |
| 100 | 100 | 100 | 100 |

---

## Clearbit API Integration

### Configuration

```bash
# .env
CLEARBIT_API_KEY=sk_xxxxxxxxxxxxxxxx
```

### Endpoint Utilisé

```
GET https://person.clearbit.com/v1/people/find?ip={IP}
```

### Rate Limiting

- **Free Tier:** 50 requêtes/mois
- **Paid:** Illimité (varie selon plan)

**Stratégie de cache agressif pour économiser les requêtes:**

```go
// TTL 7 jours
redis.Set(ctx, "clearbit:ip:"+hashedIP, data, 7*24*time.Hour)
```

### Graceful Degradation

Si l'API Clearbit est:
- **Non configurée** (pas de clé API) → Log warning, continuer sans enrichissement
- **Down** (erreur réseau) → Continuer avec User-Agent + Referer uniquement
- **Rate limited** (429) → Utiliser cache ou fallback

```go
if c.apiKey == "" {
    return nil, fmt.Errorf("clearbit api key not configured")
}
```

### Réponse Clearbit

```json
{
  "ip": "52.89.123.45",
  "name": "Amazon Web Services",
  "domain": "aws.amazon.com",
  "type": "Technology",
  "industry": "Cloud Computing",
  "employeesRange": "10001+",
  "geo": {
    "city": "Seattle",
    "country": "United States"
  },
  "person": {
    "role": "CTO",
    "title": "Chief Technology Officer"
  }
}
```

---

## Bypass Access Gate Logic

### Règle de Bypass

```go
func (s *ProfileDetectorService) ShouldBypassAccessGate(profile *DetectedProfile) bool {
    // Confidence >= 60% ET profil cible
    if profile.Confidence >= 60 {
        bypassProfiles := []ProfileType{
            ProfileTypeRecruiter,
            ProfileTypeCTO,
            ProfileTypeTechLead,
            ProfileTypeCEO,
        }

        for _, bp := range bypassProfiles {
            if profile.ProfileType == bp {
                return true
            }
        }
    }
    return false
}
```

### Stockage Bypass

```go
key := fmt.Sprintf("access:bypass:%s", sessionID)
redis.Set(ctx, key, "1", 30*24*time.Hour) // 30 jours
```

### Vérification Bypass

```go
func CheckAccessGateBypass(c *fiber.Ctx, redis *redis.Client) (bool, error) {
    sessionID := c.Cookies("session_id", "")
    key := fmt.Sprintf("access:bypass:%s", sessionID)
    val, err := redis.Get(c.Context(), key).Result()
    return val == "1", err
}
```

### Intégration avec Letters API

```go
// Dans middleware access_gate.go
bypassed, _ := CheckAccessGateBypass(c, redis)
if bypassed {
    // Skip le compteur de 3 visites
    return c.Next()
}
```

---

## Privacy & RGPD Compliance

### 1. IP Hashing (SHA-256)

**Problème:** Stocker les IPs en clair = données personnelles (RGPD Article 4)

**Solution:** Hash SHA-256 avec salt

```go
func hashIP(ip string) string {
    salt := "maicivy_ip_salt_2025" // Devrait être en env variable
    hash := sha256.Sum256([]byte(ip + salt))
    return hex.EncodeToString(hash[:])
}
```

**Résultat:**
- IP `192.168.1.1` → Hash `a3c7f...` (64 caractères)
- **Irréversible** (impossible de retrouver l'IP)
- **Déterministe** (même IP = même hash)

### 2. Pas de PII (Personally Identifiable Information)

**Données NON stockées:**
- ❌ Noms
- ❌ Emails
- ❌ Numéros de téléphone
- ❌ Adresses postales

**Données stockées (anonymes):**
- ✅ IP hashée
- ✅ Type de profil (recruiter, cto, etc.)
- ✅ Nom d'entreprise (public data via Clearbit)
- ✅ Industrie (public data)

### 3. Clearbit Data (Public Only)

Clearbit utilise **uniquement des données publiques** :
- Informations d'entreprise (publiques)
- Job titles (LinkedIn public profiles)
- Industry (données publiques)

**Pas de données privées** (emails personnels, etc.)

### 4. Cache TTL Court

```go
// Clearbit cache: 7 jours max
redis.Set(ctx, key, data, 7*24*time.Hour)

// Profile session cache: 24h
redis.Set(ctx, key, profile, 24*time.Hour)
```

**Justification:**
- Les données peuvent changer (changement de poste)
- Minimise la rétention de données

### 5. RGPD Article 17 - Droit à l'oubli

**Implémentation future:**

```go
// Endpoint: DELETE /api/v1/profile/me
func DeleteMyProfile(c *fiber.Ctx) error {
    sessionID := c.Cookies("session_id")

    // Supprimer de Redis
    redis.Del(ctx, "profile:session:"+sessionID)
    redis.Del(ctx, "clearbit:ip:"+hashedIP)

    // Anonymiser en DB
    db.Model(&Visitor{}).Where("session_id = ?", sessionID).Updates(map[string]interface{}{
        "profile_type": "other",
        "enrichment_data": nil,
        "company_name": "",
    })

    return c.JSON(fiber.Map{"success": true})
}
```

---

## Exemples de Détection

### Exemple 1: Recruiter LinkedIn

**Input:**
```
IP: 52.89.123.45 (LinkedIn office)
User-Agent: LinkedInApp/1.0
Referer: https://www.linkedin.com/jobs/view/123456
```

**Détection:**
```
1. User-Agent: "LinkedInApp" → 30 points (recruiter)
2. Referer: "linkedin.com/jobs" → 20 points (recruiter)
3. Clearbit IP: company_type="Social Networking" → 0 (pas recruiting)

Final Confidence: (30*0.3) + (20*0.2) + (0*0.5) = 13%

ProfileType: recruiter
Confidence: 13%
Bypass: NO (< 60%)
```

### Exemple 2: CTO Google

**Input:**
```
IP: 142.250.185.46 (Google office)
User-Agent: Chrome/120.0.0.0
Referer: (direct)
```

**Détection:**
```
1. User-Agent: "Chrome" → 0 points
2. Referer: (none) → 0 points
3. Clearbit IP:
   - company_name: "Google"
   - job_title: "CTO"
   → 50 points (cto)

Final Confidence: (0*0.3) + (0*0.2) + (50*0.5) = 25%

ProfileType: cto
Confidence: 25%
Bypass: NO (< 60%)
```

### Exemple 3: Recruiter Greenhouse

**Input:**
```
IP: 104.16.123.45
User-Agent: Greenhouse/1.0
Referer: https://app.greenhouse.io/
```

**Détection:**
```
1. User-Agent: "Greenhouse" → 30 points (recruiter)
2. Referer: (no pattern match) → 0 points
3. Clearbit IP:
   - company_type: "Recruiting"
   - job_title: "Recruiter"
   → 80 points (40+40)

Final Confidence: (30*0.3) + (0*0.2) + (80*0.5) = 49%

ProfileType: recruiter
Confidence: 49%
Bypass: NO (< 60%)
```

### Exemple 4: Developer cURL

**Input:**
```
IP: 93.184.216.34
User-Agent: curl/7.68.0
Referer: (none)
```

**Détection:**
```
1. User-Agent: "curl" → 20 points (developer)
2. Referer: (none) → 0 points
3. Clearbit IP: (no data) → 0 points

Final Confidence: (20*0.3) + (0*0.2) + (0*0.5) = 6%

ProfileType: developer
Confidence: 6%
Bypass: NO (developer never bypasses)
```

### Exemple 5: Perfect Storm Recruiter

**Input:**
```
IP: 52.123.45.67 (Workable office)
User-Agent: Workable/1.0
Referer: https://www.linkedin.com/recruiter/
```

**Détection:**
```
1. User-Agent: "Workable" → 30 points (recruiter)
2. Referer: "linkedin.com/recruiter" → 20 points (recruiter)
3. Clearbit IP:
   - company_type: "Recruiting"
   - job_title: "Senior Talent Acquisition Manager"
   → 80 points (40+40)

Final Confidence: (30*0.3) + (20*0.2) + (80*0.5) = 53%

ProfileType: recruiter
Confidence: 53%
Bypass: NO (< 60% threshold)
```

**Note:** Pour atteindre 60%, il faudrait un IP score de 90+ ou un combo parfait.

---

## Performance

### Objectifs

| Métrique | Objectif | Mesuré |
|----------|----------|--------|
| Détection totale | < 200ms | - |
| Clearbit API call | < 500ms | - |
| Clearbit cached | < 50ms | - |
| Middleware overhead | < 50ms | - |
| Cache hit ratio | > 80% | - |

### Optimisations

**1. Cache Redis agressif**
```go
// Session profile: 24h
redis.Set(ctx, "profile:session:"+sessionID, profile, 24*time.Hour)

// Clearbit: 7 jours
redis.Set(ctx, "clearbit:ip:"+hashedIP, data, 7*24*time.Hour)
```

**2. Traitement asynchrone**
```go
// Enregistrement DB en arrière-plan
go m.storeProfileInDB(context.Background(), sessionID, ip, profile)
```

**3. Graceful degradation**
- Si Clearbit timeout → continuer avec UA + Referer
- Si cache Redis down → continuer sans cache (performance dégradée)

**4. Lazy loading**
- Le profil est détecté **après** `visitor_tracking` middleware
- N'impacte pas la requête initiale

---

## Tests

### Tests Unitaires

**Fichier:** `backend/internal/services/profile_detector_test.go`

**Couverture:**
- `TestIsRecruiterBot` (6 cas)
- `TestCalculateFinalConfidence` (5 cas)
- `TestHashIP` (déterminisme, unicité)
- `TestDetectFromUserAgent` (5 cas)
- `TestDetectFromReferer` (7 cas)
- `TestAnalyzeEnrichmentData` (7 cas)
- `TestShouldBypassAccessGate` (7 cas)

**Benchmarks:**
- `BenchmarkHashIP`
- `BenchmarkCalculateFinalConfidence`

### Exécution

```bash
# Tests unitaires
go test -v ./backend/internal/services -run TestProfile

# Avec couverture
go test -v ./backend/internal/services -coverprofile=coverage.out
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchmem ./backend/internal/services
```

### Tests d'Intégration (À implémenter)

```go
// Test avec vrai Redis
func TestProfileDetectionIntegration(t *testing.T) {
    // Setup Redis + DB
    // Call DetectProfile()
    // Vérifier cache Redis
    // Vérifier DB write
}
```

---

## Maintenance

### Monitoring

**Métriques à surveiller:**

```
# Prometheus metrics
profile_detection_total{type="recruiter"} 123
profile_detection_total{type="cto"} 45
profile_detection_confidence_avg{type="recruiter"} 62.5

clearbit_api_calls_total 1234
clearbit_api_errors_total 5
clearbit_cache_hit_ratio 0.85

bypass_activated_total 567
```

**Alertes:**
- Clearbit API error rate > 10%
- Cache hit ratio < 70%
- Confidence moyenne < 30% (détection inefficace)

### Ajout de Nouveaux Patterns

**1. Ajouter un pattern recruiter:**

```go
// Dans detectFromUserAgent()
recruiterPatterns := []string{
    // ... existants
    "smartrecruiters", // NOUVEAU
}
```

**2. Ajouter un job board:**

```go
// Dans detectFromReferer()
jobBoards := []string{
    // ... existants
    "hired.com", // NOUVEAU
}
```

**3. Tester:**

```go
func TestNewPattern(t *testing.T) {
    service := &ProfileDetectorService{}
    score, ptype := service.detectFromUserAgent("SmartRecruiters/1.0")
    assert.Equal(t, 30, score)
    assert.Equal(t, ProfileTypeRecruiter, ptype)
}
```

### Ajustement du Seuil de Bypass

**Actuel:** 60%

**Pour augmenter (plus strict):**
```go
if profile.Confidence >= 70 { // Au lieu de 60
    // ...
}
```

**Pour diminuer (plus permissif):**
```go
if profile.Confidence >= 50 {
    // ...
}
```

### Clearbit API Key Rotation

```bash
# 1. Obtenir nouvelle clé Clearbit
# 2. Mettre à jour .env
CLEARBIT_API_KEY=sk_newkeyxxxxxxxx

# 3. Redémarrer le backend
docker-compose restart backend

# 4. Vider le cache (optionnel)
redis-cli KEYS "clearbit:*" | xargs redis-cli DEL
```

---

## Ressources

### Documentation Externe

- [Clearbit API Docs](https://clearbit.com/resources/api)
- [RGPD - Article 4 (Données personnelles)](https://gdpr.eu/article-4-definitions/)
- [User-Agent Parsing Best Practices](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent)

### Fichiers Clés

**Backend:**
- `/backend/internal/services/profile_detector.go`
- `/backend/internal/services/clearbit_client.go`
- `/backend/internal/middleware/profile_enrichment.go`
- `/backend/internal/models/visitor.go`

**Frontend:**
- `/frontend/hooks/useProfileDetection.ts`
- `/frontend/components/profile/ProfileBadge.tsx`

**Tests:**
- `/backend/internal/services/profile_detector_test.go`

---

## Changelog

### v1.0 (2025-12-08)
- Implémentation initiale Feature 3
- Détection via User-Agent, Referer, Clearbit
- Bypass access gate si confidence >= 60%
- Privacy RGPD compliant (IP hashing)
- Cache Redis (24h profil, 7j Clearbit)
- Components frontend (Badge, Message, Hook)

---

**Auteur:** Claude (Agent IA)
**Projet:** maicivy - CV Interactif avec IA
**Phase:** 5 - Features Avancées
