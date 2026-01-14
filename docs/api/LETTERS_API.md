# Letters API Documentation

## Overview

AI-powered generation of motivation and anti-motivation letters.

### Features

- **Dual Generation**: Creates both motivation AND anti-motivation letters
- **Company Research**: Automatic scraping of company information
- **Access Gate**: Requires 3 visits OR detected profile
- **Rate Limiting**: 5 generations/day, 2-minute cooldown
- **Async Processing**: Job-based queue system

---

## Endpoints

### Generate Letters

Create motivation and anti-motivation letters for a company.

```http
POST /api/v1/letters/generate
```

#### Request Body

```json
{
  "company_name": "Google",
  "job_title": "Backend Engineer",
  "theme": "backend"
}
```

#### Example Request

```bash
curl -X POST "http://localhost:5000/api/v1/letters/generate" \
  -H "Content-Type: application/json" \
  -b "session_id=abc123" \
  -d '{
    "company_name": "Google",
    "job_title": "Backend Engineer"
  }'
```

#### Success Response (202 Accepted)

```json
{
  "job_id": "job_abc123def456",
  "status": "queued",
  "message": "Génération en cours. Encore 4 génération(s) disponible(s) aujourd'hui.",
  "rate_limit_remaining": 4
}
```

#### Error Responses

**403 Forbidden (Access Gate)**
```json
{
  "error": "Access denied",
  "code": "ACCESS_GATE_LOCKED",
  "message": "AI features available after 3 visits"
}
```

**429 Too Many Requests (Rate Limit)**
```json
{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Max 5 generations per day. Try again tomorrow."
}
```

---

### Get Job Status

Poll generation job status.

```http
GET /api/v1/letters/jobs/{jobId}
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/letters/jobs/job_abc123def456"
```

#### Response (Queued)

```json
{
  "job_id": "job_abc123def456",
  "status": "queued",
  "progress": 0,
  "estimated_time": 60
}
```

#### Response (Processing)

```json
{
  "job_id": "job_abc123def456",
  "status": "processing",
  "progress": 50,
  "estimated_time": 30
}
```

#### Response (Completed)

```json
{
  "job_id": "job_abc123def456",
  "status": "completed",
  "progress": 100,
  "letter_motivation_id": 42,
  "letter_anti_motivation_id": 43,
  "estimated_time": 0
}
```

#### Response (Failed)

```json
{
  "job_id": "job_abc123def456",
  "status": "failed",
  "progress": 0,
  "error": "AI service unavailable"
}
```

---

### Get Letter by ID

Retrieve a specific letter.

```http
GET /api/v1/letters/{id}
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/letters/42" \
  -b "session_id=abc123"
```

#### Response

```json
{
  "id": 42,
  "company_name": "Google",
  "letter_type": "motivation",
  "content": "Cher Monsieur...",
  "created_at": "2025-12-08 10:30:00",
  "ai_model": "claude-3-sonnet-20240229",
  "tokens_used": 512,
  "generation_ms": 2340,
  "cost": 0.0076,
  "pdf_url": "http://localhost:5000/api/v1/letters/42/pdf"
}
```

---

### Download Letter PDF

Download letter as PDF.

```http
GET /api/v1/letters/{id}/pdf
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/letters/42/pdf" \
  -b "session_id=abc123" \
  -o letter.pdf
```

#### Response

- Content-Type: `application/pdf`
- Binary PDF data

---

### Get Letter Pair

Retrieve both letters for a company.

```http
GET /api/v1/letters/pair?company={company_name}
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/letters/pair?company=Google" \
  -b "session_id=abc123"
```

#### Response

```json
{
  "motivation_letter": {
    "id": 42,
    "company_name": "Google",
    "letter_type": "motivation",
    "content": "..."
  },
  "anti_motivation_letter": {
    "id": 43,
    "company_name": "Google",
    "letter_type": "anti_motivation",
    "content": "..."
  },
  "company_name": "Google"
}
```

---

### Get History

Retrieve generation history.

```http
GET /api/v1/letters/history?page={page}&per_page={per_page}
```

#### Parameters

| Name | Type | Default | Description |
|------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Results per page (max: 50) |

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/letters/history?page=1&per_page=10" \
  -b "session_id=abc123"
```

#### Response

```json
{
  "letters": [
    {
      "id": 42,
      "company_name": "Google",
      "letter_type": "motivation",
      "created_at": "2025-12-08 10:30:00",
      "downloaded": true
    }
  ],
  "total": 8,
  "page": 1,
  "per_page": 10
}
```

---

### Get Access Status

Check AI access status.

```http
GET /api/v1/letters/access-status
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/letters/access-status" \
  -b "session_id=abc123"
```

#### Response (No Access)

```json
{
  "has_access": false,
  "current_visits": 1,
  "required_visits": 3,
  "visits_remaining": 2,
  "profile_detected": "",
  "access_granted_by": "",
  "message": "Encore 2 visite(s) nécessaire(s) pour débloquer l'IA"
}
```

#### Response (Access Granted)

```json
{
  "has_access": true,
  "current_visits": 3,
  "required_visits": 3,
  "visits_remaining": 0,
  "profile_detected": "",
  "access_granted_by": "visits",
  "message": "Accès aux fonctionnalités IA accordé"
}
```

---

### Get Rate Limit Status

Check rate limit status.

```http
GET /api/v1/letters/rate-limit-status
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/letters/rate-limit-status" \
  -b "session_id=abc123"
```

#### Response

```json
{
  "daily_limit": 5,
  "daily_used": 2,
  "daily_remaining": 3,
  "reset_at": "2025-12-09 00:00:00",
  "cooldown_active": false,
  "cooldown_remaining": 0
}
```

---

## Generation Flow

```
1. POST /api/v1/letters/generate
   → Returns job_id

2. Poll GET /api/v1/letters/jobs/{jobId}
   → Status: queued → processing → completed

3. GET /api/v1/letters/{letter_id}
   → Retrieve letter content

4. GET /api/v1/letters/{letter_id}/pdf
   → Download PDF
```

---

## AI Models

### Claude (Anthropic)

- Model: `claude-3-sonnet-20240229`
- Use: Motivation letters
- Cost: ~$0.015 per 1000 tokens

### GPT-4 (OpenAI)

- Model: `gpt-4-turbo-preview`
- Use: Anti-motivation letters
- Cost: ~$0.01 per 1000 tokens

---

## Rate Limiting

### Daily Limit

- **5 generations per day** per session
- Resets at midnight UTC
- Tracked in Redis: `ratelimit:ai:{session_id}:daily`

### Cooldown

- **2 minutes** between generations
- Prevents rapid successive requests
- Tracked in Redis: `ratelimit:ai:{session_id}:cooldown`

### Bypass

Target profiles bypass rate limits:
- Recruiters
- Tech leads
- CTOs/CEOs

---

## Error Codes

| Code | Status | Description |
|------|--------|-------------|
| VALIDATION_ERROR | 400 | Invalid request data |
| SESSION_REQUIRED | 401 | Missing session cookie |
| ACCESS_GATE_LOCKED | 403 | Needs 3 visits |
| RATE_LIMIT_EXCEEDED | 429 | 5/day limit reached |
| COOLDOWN_ACTIVE | 429 | 2-minute cooldown |
| JOB_NOT_FOUND | 404 | Invalid job ID |
| LETTER_NOT_FOUND | 404 | Letter doesn't exist |
| QUEUE_ERROR | 500 | Job queue failed |
| AI_ERROR | 500 | AI service error |

---

## Caching

Letters are NOT cached (each generation is unique).

---

## Cost Estimation

Average generation:
- Motivation letter: ~500 tokens → $0.0075
- Anti-motivation letter: ~400 tokens → $0.004
- **Total per pair**: ~$0.012

Monthly budget (1000 generations): ~$12
