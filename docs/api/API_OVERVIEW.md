# maicivy API - Overview

## Base URL

```
Development: http://localhost:5000
Production:  https://api.maicivy.dev
```

## API Version

Current version: **v1.0.0**

All API endpoints are prefixed with `/api/v1` except health checks.

## Authentication

The API uses **session cookies** for visitor tracking and access control.

- **Cookie name**: `session_id`
- **Set automatically** on first visit
- **No manual authentication required** for public endpoints
- **Required for AI features** (letter generation)

### Session Cookie Flow

```
1. First Visit → Cookie automatically set
2. Track visits → Stored in Redis + PostgreSQL
3. Visit count ≥ 3 → AI features unlocked
4. OR profile detected → Immediate access
```

## Rate Limiting

### AI Features Rate Limits

| Feature | Limit | Window | Cooldown |
|---------|-------|--------|----------|
| Letter Generation | 5 requests | 24 hours | 2 minutes |

### Headers

Rate limit information is returned in response headers:

```http
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 3
X-RateLimit-Reset: 1670505600
Retry-After: 120
```

### Rate Limit Response

When rate limit is exceeded:

```json
{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Max 5 generations per day. Try again tomorrow."
}
```

## Access Gate

AI features (letter generation) are protected by an **access gate**:

### Requirements

One of the following must be true:
- **3 visits minimum** to the site
- **Target profile detected** (recruiter, tech lead, CTO, CEO)

### Profile Detection

Automatic detection based on:
- User-Agent analysis
- IP lookup (company identification)
- LinkedIn referrer
- Navigation patterns

### Detected Profiles

- `recruiter`
- `tech_lead`
- `cto`
- `ceo`
- `hr_manager`

## Request Format

### Content-Type

All POST/PUT requests must use:

```http
Content-Type: application/json
```

### Request Body Example

```json
{
  "company_name": "Google",
  "job_title": "Backend Engineer",
  "theme": "backend"
}
```

## Response Format

### Success Response

```json
{
  "success": true,
  "data": {
    // Response data
  },
  "meta": {
    // Optional metadata
  }
}
```

### Error Response

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "message": "Detailed explanation",
  "details": "Additional context"
}
```

## HTTP Status Codes

| Code | Meaning | Usage |
|------|---------|-------|
| 200 | OK | Successful GET request |
| 201 | Created | Resource created successfully |
| 202 | Accepted | Request accepted (async processing) |
| 400 | Bad Request | Invalid request data |
| 401 | Unauthorized | Session required |
| 403 | Forbidden | Access denied (access gate) |
| 404 | Not Found | Resource not found |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |
| 503 | Service Unavailable | Service degraded |

## Pagination

Paginated endpoints support query parameters:

```
GET /api/v1/letters/history?page=1&per_page=10
```

### Parameters

- `page`: Page number (default: 1)
- `per_page`: Results per page (default: 10, max: 50)

### Response

```json
{
  "letters": [...],
  "total": 42,
  "page": 1,
  "per_page": 10
}
```

## CORS

CORS is enabled for:
- Origins: `http://localhost:3000`, `https://maicivy.dev`
- Methods: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`
- Headers: `Content-Type`, `Authorization`, `X-Requested-With`
- Credentials: Allowed (for cookies)

## Caching

### Redis Caching

The following endpoints use Redis caching:

| Endpoint | Cache TTL |
|----------|-----------|
| GET /api/v1/cv | 1 hour |
| GET /api/v1/cv/themes | 24 hours |
| GET /api/v1/experiences | 1 hour |
| GET /api/v1/skills | 1 hour |
| GET /api/v1/projects | 30 minutes |

### Cache Headers

```http
X-Cache-Hit: true
X-Cache-TTL: 3600
```

## Versioning

API versioning follows **URL path versioning**:

```
/api/v1/...  (current)
/api/v2/...  (future)
```

### Breaking Changes

Breaking changes will result in a new API version. Non-breaking changes (additions) will be made to the current version.

### Deprecation Policy

- **Notice**: 6 months before deprecation
- **Support**: Old versions supported for 12 months
- **Removal**: After 12 months

## Error Codes Reference

See [ERROR_CODES.md](ERROR_CODES.md) for complete error codes documentation.

## WebSocket Endpoints

### Analytics WebSocket

```
ws://localhost:5000/ws/analytics
```

Real-time analytics updates. See [WEBSOCKET_PROTOCOL.md](WEBSOCKET_PROTOCOL.md) for protocol details.

## SDKs and Client Libraries

### Official Clients

- **TypeScript/JavaScript**: Auto-generated from OpenAPI spec
- **Go**: Auto-generated with oapi-codegen
- **Python**: Available via openapi-generator

See [API Client Generation](#api-client-generation) for instructions.

## Testing

### Swagger UI

Interactive API documentation and testing:

```
http://localhost:5000/api/docs/
```

### Postman Collection

Import the Postman collection:

```
docs/api/maicivy.postman_collection.json
```

### cURL Examples

See individual API documentation files:
- [CV_API.md](CV_API.md)
- [LETTERS_API.md](LETTERS_API.md)
- [ANALYTICS_API.md](ANALYTICS_API.md)

## API Limits

| Resource | Limit |
|----------|-------|
| Request body size | 1 MB |
| Request timeout | 30 seconds |
| Long-running operations | Async (job-based) |

## Health Checks

```bash
# Shallow health check
curl http://localhost:5000/health

# Deep health check (DB + Redis)
curl http://localhost:5000/health/deep
```

## Security

- **HTTPS only** in production
- **Rate limiting** on all endpoints
- **Input validation** with sanitization
- **SQL injection protection** via GORM
- **XSS protection** via output escaping
- **CSRF protection** for state-changing operations

See [SECURITY.md](../../implementation/17_SECURITY.md) for complete security documentation.

## Support

- **Issues**: GitHub Issues
- **Email**: contact@maicivy.dev
- **Documentation**: https://maicivy.dev/docs

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for API version history and changes.
