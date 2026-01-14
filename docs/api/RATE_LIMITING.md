# Rate Limiting Documentation

## Overview

Rate limiting protects the API from abuse and controls costs for AI-powered features.

---

## Rate Limits by Endpoint

### AI Features (Letters Generation)

| Endpoint | Limit | Window | Cooldown |
|----------|-------|--------|----------|
| POST /api/v1/letters/generate | 5 requests | 24 hours | 2 minutes |

### Public Endpoints (No Limits)

All other endpoints have no rate limits:
- CV endpoints
- Analytics endpoints
- GitHub endpoints

---

## Headers

### Request Headers

No special headers required.

### Response Headers

All responses include:

```http
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 3
X-RateLimit-Reset: 1670505600
X-RateLimit-Window: 86400
```

### Rate Limit Exceeded Response

Status: **429 Too Many Requests**

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1670505600
Retry-After: 86400

{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Max 5 generations per day. Try again tomorrow."
}
```

---

## Implementation

### Redis Keys

#### Daily Limit

```
Key: ratelimit:ai:{session_id}:daily
Value: {count}
TTL: 24 hours (resets at midnight UTC)
```

#### Cooldown

```
Key: ratelimit:ai:{session_id}:cooldown
Value: "locked"
TTL: 120 seconds
```

### Algorithm

```go
func CheckRateLimit(sessionID string) (allowed bool, remaining int) {
    // Check daily limit
    dailyKey := fmt.Sprintf("ratelimit:ai:%s:daily", sessionID)
    count, _ := redis.Get(dailyKey).Int()

    if count >= 5 {
        return false, 0
    }

    // Check cooldown
    cooldownKey := fmt.Sprintf("ratelimit:ai:%s:cooldown", sessionID)
    exists := redis.Exists(cooldownKey).Val()

    if exists > 0 {
        return false, 5 - count
    }

    return true, 5 - count
}
```

---

## Bypass Rules

### Target Profiles

These profiles bypass ALL rate limits:

- **Recruiters**
- **Tech Leads**
- **CTOs**
- **CEOs**
- **HR Managers**

### Detection Method

Profiles detected via:
- LinkedIn referrer
- Company IP lookup (Clearbit API)
- User-Agent analysis
- Navigation patterns

### Checking Bypass Status

```bash
curl -X GET "http://localhost:5000/api/v1/letters/access-status" \
  -b "session_id=abc123"
```

Response:

```json
{
  "has_access": true,
  "profile_detected": "recruiter",
  "access_granted_by": "profile"
}
```

---

## Rate Limit Windows

### Daily Reset

Resets at **00:00 UTC** every day.

Example:
- First request: 2025-12-08 10:00 UTC
- Reset time: 2025-12-09 00:00 UTC

### Cooldown Timer

**2 minutes** from last successful request.

Example:
- Request 1: 10:00:00
- Request 2 (blocked): 10:01:30
- Request 2 (allowed): 10:02:01

---

## Testing Rate Limits

### Check Status

```bash
curl -X GET "http://localhost:5000/api/v1/letters/rate-limit-status" \
  -b "session_id=abc123"
```

Response:

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

### Trigger Rate Limit

```bash
# Make 5 requests quickly
for i in {1..5}; do
  curl -X POST "http://localhost:5000/api/v1/letters/generate" \
    -H "Content-Type: application/json" \
    -b "session_id=test123" \
    -d '{"company_name": "Test'$i'"}'
  sleep 1
done

# 6th request will fail with 429
curl -X POST "http://localhost:5000/api/v1/letters/generate" \
  -H "Content-Type: application/json" \
  -b "session_id=test123" \
  -d '{"company_name": "Test6"}'
```

---

## Client Implementation

### JavaScript Example

```javascript
async function generateLetter(companyName) {
  const response = await fetch('/api/v1/letters/generate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ company_name: companyName })
  });

  // Handle rate limit
  if (response.status === 429) {
    const retryAfter = response.headers.get('Retry-After');
    const resetTime = response.headers.get('X-RateLimit-Reset');

    throw new Error(`Rate limit exceeded. Retry after ${retryAfter}s`);
  }

  // Handle cooldown
  const remaining = response.headers.get('X-RateLimit-Remaining');
  console.log(`${remaining} generations remaining today`);

  return await response.json();
}
```

### Python Example

```python
import requests
import time

def generate_letter(company_name, session_id):
    response = requests.post(
        'http://localhost:5000/api/v1/letters/generate',
        json={'company_name': company_name},
        cookies={'session_id': session_id}
    )

    if response.status_code == 429:
        retry_after = int(response.headers.get('Retry-After', 0))
        print(f"Rate limited. Waiting {retry_after}s...")
        time.sleep(retry_after)
        return generate_letter(company_name, session_id)

    return response.json()
```

---

## Monitoring

### Prometheus Metrics

```promql
# Rate limit hits
rate_limit_exceeded_total{endpoint="/api/v1/letters/generate"}

# Remaining capacity
rate_limit_remaining{session_id="*"}

# Average wait time
rate_limit_retry_after_seconds
```

### Grafana Dashboard

Panels:
- Rate limit hit rate (requests/minute)
- Top rate-limited sessions
- Cooldown activations over time

---

## Cost Control

### Motivation

AI API costs:
- Claude: ~$0.015/1000 tokens
- GPT-4: ~$0.01/1000 tokens

Average generation: ~900 tokens = **$0.012 per pair**

### Budget Calculation

With 5 requests/day limit:
- Per user: 5 × $0.012 = $0.06/day
- 100 users: $6/day = $180/month
- 1000 users: $60/day = $1800/month

### Dynamic Adjustment

If costs exceed budget, adjust limits in Redis:

```bash
# Set global daily limit to 3
redis-cli SET ratelimit:global:daily 3
```

---

## FAQ

**Q: Can I increase my limit?**
A: No, limits are fixed for fairness.

**Q: Does the limit apply per device?**
A: No, per session cookie (shared across devices with same cookie).

**Q: What if I clear cookies?**
A: You get a new session with fresh limits (but still 3-visit access gate).

**Q: Can I buy additional requests?**
A: Not currently. This is a free demo project.

**Q: Do failed requests count?**
A: No, only successful generations (status 202) count.
