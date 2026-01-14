# API Error Codes Reference

## Format

All errors follow this format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "message": "Detailed explanation",
  "details": "Additional context (optional)"
}
```

---

## Common Errors

### E001 - INVALID_REQUEST

**Status**: 400 Bad Request

**Description**: Request body is malformed or unparseable

**Resolution**: Check JSON syntax and structure

---

### E002 - VALIDATION_ERROR

**Status**: 400 Bad Request

**Description**: Request data failed validation

**Resolution**: Check required fields and data types

---

### E003 - SESSION_REQUIRED

**Status**: 401 Unauthorized

**Description**: Session cookie missing or invalid

**Resolution**: Ensure cookies are enabled and visit the site first

---

### E004 - ACCESS_GATE_LOCKED

**Status**: 403 Forbidden

**Description**: AI features require 3 visits minimum

**Resolution**: Visit the site 2 more times or wait for profile detection

---

### E005 - RATE_LIMIT_EXCEEDED

**Status**: 429 Too Many Requests

**Description**: Daily rate limit (5/day) exceeded

**Resolution**: Wait until reset time (see `Retry-After` header)

---

### E006 - COOLDOWN_ACTIVE

**Status**: 429 Too Many Requests

**Description**: 2-minute cooldown between generations active

**Resolution**: Wait for cooldown to expire

---

## CV API Errors

### E100 - INVALID_THEME

**Status**: 400 Bad Request

**Description**: Specified theme does not exist

**Resolution**: Use GET /api/v1/cv/themes to see available themes

---

### E101 - INVALID_FORMAT

**Status**: 400 Bad Request

**Description**: Export format not supported (only PDF)

**Resolution**: Use `format=pdf` parameter

---

### E102 - PDF_GENERATION_ERROR

**Status**: 500 Internal Server Error

**Description**: PDF generation failed

**Resolution**: Retry request or contact support

---

## Letters API Errors

### E200 - COMPANY_NAME_REQUIRED

**Status**: 400 Bad Request

**Description**: company_name field is missing or empty

**Resolution**: Provide valid company_name in request body

---

### E201 - JOB_NOT_FOUND

**Status**: 404 Not Found

**Description**: Generation job ID does not exist

**Resolution**: Check job_id value returned from POST /letters/generate

---

### E202 - LETTER_NOT_FOUND

**Status**: 404 Not Found

**Description**: Letter ID does not exist or access denied

**Resolution**: Verify letter ID and session ownership

---

### E203 - QUEUE_ERROR

**Status**: 500 Internal Server Error

**Description**: Failed to enqueue generation job

**Resolution**: Retry or check queue service status

---

### E204 - AI_ERROR

**Status**: 500 Internal Server Error

**Description**: AI service (Claude/GPT) error

**Resolution**: Check AI service availability, retry later

---

### E205 - VISITOR_NOT_FOUND

**Status**: 401 Unauthorized

**Description**: Visitor record not found in database

**Resolution**: Clear cookies and revisit site

---

## Analytics API Errors

### E300 - INVALID_PERIOD

**Status**: 400 Bad Request

**Description**: Period parameter must be day/week/month

**Resolution**: Use valid period value

---

### E301 - INVALID_EVENT

**Status**: 400 Bad Request

**Description**: Event type is invalid or missing

**Resolution**: Provide valid event_type

---

### E302 - MISSING_VISITOR

**Status**: 400 Bad Request

**Description**: Visitor session not found in context

**Resolution**: Ensure tracking middleware is active

---

## GitHub API Errors

### E400 - GITHUB_SYNC_ERROR

**Status**: 500 Internal Server Error

**Description**: GitHub synchronization failed

**Resolution**: Check GitHub API key and rate limits

---

### E401 - GITHUB_API_UNAVAILABLE

**Status**: 503 Service Unavailable

**Description**: GitHub API is down or unreachable

**Resolution**: Wait and retry later

---

## Database Errors

### E500 - DB_ERROR

**Status**: 500 Internal Server Error

**Description**: Database query failed

**Resolution**: Check database connectivity

---

### E501 - DB_CONNECTION_LOST

**Status**: 503 Service Unavailable

**Description**: Database connection lost

**Resolution**: Service will reconnect automatically

---

## Redis Errors

### E600 - REDIS_ERROR

**Status**: 500 Internal Server Error

**Description**: Redis operation failed

**Resolution**: Check Redis connectivity

---

### E601 - CACHE_ERROR

**Status**: 500 Internal Server Error

**Description**: Cache read/write error

**Resolution**: Data will be fetched from database

---

## General Errors

### E900 - INTERNAL_ERROR

**Status**: 500 Internal Server Error

**Description**: Unexpected internal error

**Resolution**: Contact support with error details

---

### E901 - SERVICE_UNAVAILABLE

**Status**: 503 Service Unavailable

**Description**: Service temporarily unavailable

**Resolution**: Retry with exponential backoff

---

### E902 - TIMEOUT

**Status**: 504 Gateway Timeout

**Description**: Request timed out (>30s)

**Resolution**: Operation may have succeeded, check status

---

## Response Headers

Error responses include these headers:

```http
X-Error-Code: E001
X-Request-ID: req_abc123
Retry-After: 120 (for rate limits)
```

---

## Retry Logic

### Retryable Errors (5xx)

- E500 - DB_ERROR
- E600 - REDIS_ERROR
- E900 - INTERNAL_ERROR
- E901 - SERVICE_UNAVAILABLE
- E902 - TIMEOUT

**Strategy**: Exponential backoff
- 1st retry: 1s
- 2nd retry: 2s
- 3rd retry: 4s
- Max retries: 3

### Non-Retryable Errors (4xx)

- E001-E006 (validation errors)
- E100-E301 (client errors)

**Strategy**: Fix request and retry

---

## Contact Support

For persistent errors:
- Email: contact@maicivy.dev
- Include: X-Request-ID header value
