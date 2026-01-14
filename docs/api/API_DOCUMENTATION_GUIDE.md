# API Documentation Usage Guide

Complete guide to using the maicivy API documentation.

---

## 📚 Documentation Files

### Core Documentation

| File | Purpose | Audience |
|------|---------|----------|
| [API_OVERVIEW.md](API_OVERVIEW.md) | General API information, authentication, rate limiting | Everyone |
| [openapi.yaml](openapi.yaml) | OpenAPI 3.0 specification | Developers, Tools |
| [CV_API.md](CV_API.md) | CV endpoints documentation | Frontend developers |
| [LETTERS_API.md](LETTERS_API.md) | AI letter generation endpoints | Frontend developers |
| [ANALYTICS_API.md](ANALYTICS_API.md) | Analytics and stats endpoints | Frontend/Analytics |
| [ERROR_CODES.md](ERROR_CODES.md) | Complete error codes reference | Debugging |
| [RATE_LIMITING.md](RATE_LIMITING.md) | Rate limiting details | Backend/Frontend |
| [WEBSOCKET_PROTOCOL.md](WEBSOCKET_PROTOCOL.md) | WebSocket protocol spec | Real-time features |
| [CHANGELOG.md](CHANGELOG.md) | API version history | Everyone |

### Tools

| File | Purpose |
|------|---------|
| [maicivy.postman_collection.json](maicivy.postman_collection.json) | Postman collection |
| [generate-api-clients.sh](../../scripts/generate-api-clients.sh) | Client generator script |

---

## 🚀 Quick Start

### 1. Browse Interactive Documentation

**Swagger UI** (Recommended for beginners):

```
http://localhost:5000/api/docs/
```

- ✅ Try endpoints directly in browser
- ✅ See request/response examples
- ✅ Auto-complete and validation
- ✅ No installation required

### 2. Import Postman Collection

```bash
# Import in Postman
File → Import → docs/api/maicivy.postman_collection.json

# Set environment variable
base_url = http://localhost:5000
```

### 3. Read Markdown Docs

Start with [API_OVERVIEW.md](API_OVERVIEW.md), then dive into specific APIs.

---

## 🔧 For Developers

### Frontend Developers

**Workflow**:

1. Read [API_OVERVIEW.md](API_OVERVIEW.md) for basics
2. Read endpoint-specific docs:
   - [CV_API.md](CV_API.md) - For CV pages
   - [LETTERS_API.md](LETTERS_API.md) - For letter generator
   - [ANALYTICS_API.md](ANALYTICS_API.md) - For dashboards
3. Use Swagger UI to test endpoints
4. Generate TypeScript types:
   ```bash
   bash scripts/generate-api-clients.sh
   ```
5. Import generated types in your code:
   ```typescript
   import { paths } from './generated-clients/typescript/api';
   ```

**Key Files**:
- API_OVERVIEW.md
- CV_API.md
- LETTERS_API.md
- ANALYTICS_API.md
- WEBSOCKET_PROTOCOL.md

---

### Backend Developers

**Workflow**:

1. Read [openapi.yaml](openapi.yaml) for spec
2. Review error codes in [ERROR_CODES.md](ERROR_CODES.md)
3. Understand rate limiting in [RATE_LIMITING.md](RATE_LIMITING.md)
4. Add new endpoints to OpenAPI spec
5. Update version in [CHANGELOG.md](CHANGELOG.md)

**Key Files**:
- openapi.yaml
- ERROR_CODES.md
- RATE_LIMITING.md
- CHANGELOG.md

---

### QA / Testers

**Workflow**:

1. Import Postman collection
2. Use Swagger UI for manual testing
3. Reference ERROR_CODES.md for expected errors
4. Check CHANGELOG.md for recent changes

**Key Files**:
- maicivy.postman_collection.json
- Swagger UI (http://localhost:5000/api/docs/)
- ERROR_CODES.md

---

### DevOps / Infrastructure

**Workflow**:

1. Monitor rate limiting (RATE_LIMITING.md)
2. Set up health check endpoints (`/health`, `/health/deep`)
3. Configure WebSocket support (WEBSOCKET_PROTOCOL.md)
4. Review API_OVERVIEW.md for infrastructure requirements

**Key Files**:
- RATE_LIMITING.md
- WEBSOCKET_PROTOCOL.md
- API_OVERVIEW.md

---

## 📖 Using Swagger UI

### Access

```
http://localhost:5000/api/docs/
```

### Features

1. **Try It Out**: Execute requests directly
2. **Schemas**: View request/response models
3. **Examples**: See sample data
4. **Authorization**: Test with cookies/auth

### Workflow

```
1. Open Swagger UI
2. Find endpoint (e.g., GET /api/v1/cv)
3. Click "Try it out"
4. Fill parameters
5. Click "Execute"
6. See response
```

### Tips

- **Cookies**: Swagger automatically includes cookies
- **CORS**: Make sure frontend is running on `localhost:3000`
- **Rate Limits**: Watch for 429 responses

---

## 📦 Using Postman

### Import Collection

```
1. Open Postman
2. File → Import
3. Select: docs/api/maicivy.postman_collection.json
4. Collection imported!
```

### Environment Setup

Create environment with:

```
base_url = http://localhost:5000
```

### Running Requests

```
1. Select collection
2. Choose request (e.g., "Get Adaptive CV")
3. Click "Send"
4. View response
```

### Pre-request Scripts

Collection includes scripts for:
- Auto-saving job_id from letter generation
- Setting session cookies
- Logging requests

---

## 🤖 Generating API Clients

### Prerequisites

Install tools:

```bash
# TypeScript
npm install -g openapi-typescript

# Go
go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest

# Python
npm install -g @openapitools/openapi-generator-cli
```

### Generate

```bash
bash scripts/generate-api-clients.sh
```

### Output

```
generated-clients/
├── typescript/
│   └── api.ts
├── go/
│   ├── types.go
│   └── client.go
├── python/
│   └── maicivy_api/
└── README.md
```

### Usage Examples

**TypeScript**:

```typescript
import { paths } from './generated-clients/typescript/api';

type CVResponse = paths['/api/v1/cv']['get']['responses']['200']['content']['application/json'];

const response = await fetch('http://localhost:5000/api/v1/cv?theme=backend');
const cv: CVResponse = await response.json();
```

**Go**:

```go
import apiclient "maicivy/generated-clients/go"

client, _ := apiclient.NewClient("http://localhost:5000")
cv, _ := client.GetCV(ctx, &apiclient.GetCVParams{Theme: "backend"})
```

---

## 🐛 Debugging with Documentation

### Problem: "Access denied" error

1. Check [ERROR_CODES.md](ERROR_CODES.md) → E004 (ACCESS_GATE_LOCKED)
2. Read [LETTERS_API.md](LETTERS_API.md) → Access Gate section
3. Solution: Visit site 3 times OR get profile detected

### Problem: Rate limit exceeded

1. Check [ERROR_CODES.md](ERROR_CODES.md) → E005 (RATE_LIMIT_EXCEEDED)
2. Read [RATE_LIMITING.md](RATE_LIMITING.md)
3. Check status: `GET /api/v1/letters/rate-limit-status`
4. Solution: Wait for reset time

### Problem: Invalid theme

1. Check [CV_API.md](CV_API.md) → Get Available Themes
2. Fetch: `GET /api/v1/cv/themes`
3. Use valid theme ID

---

## 📝 Contributing to Documentation

### Update OpenAPI Spec

```yaml
# docs/api/openapi.yaml

paths:
  /api/v1/new-endpoint:
    get:
      summary: New endpoint
      description: Description
      responses:
        '200':
          description: Success
```

### Add to Postman

```json
{
  "name": "New Endpoint",
  "request": {
    "method": "GET",
    "url": "{{base_url}}/api/v1/new-endpoint"
  }
}
```

### Update Markdown Docs

Add section to relevant API doc (e.g., CV_API.md):

```markdown
### New Endpoint

```http
GET /api/v1/new-endpoint
```

#### Response

```json
{
  "data": "..."
}
```
```

### Update CHANGELOG

```markdown
## [1.1.0] - 2025-12-15

### Added

- New endpoint GET /api/v1/new-endpoint
```

---

## 🔗 External Resources

- **OpenAPI Spec**: https://spec.openapis.org/oas/v3.0.0
- **Swagger UI**: https://swagger.io/tools/swagger-ui/
- **Postman**: https://www.postman.com/
- **oapi-codegen**: https://github.com/deepmap/oapi-codegen

---

## 📞 Support

Questions about documentation?

- **Issues**: GitHub Issues
- **Email**: contact@maicivy.dev
- **Swagger UI**: Test directly at /api/docs/

---

## ✅ Checklist for Using Documentation

### Before Starting Development

- [ ] Read API_OVERVIEW.md
- [ ] Understand authentication (session cookies)
- [ ] Check rate limiting rules
- [ ] Set up Swagger UI or Postman
- [ ] Generate API clients (optional)

### During Development

- [ ] Reference endpoint-specific docs (CV_API.md, etc.)
- [ ] Use Swagger UI to test endpoints
- [ ] Check ERROR_CODES.md for errors
- [ ] Monitor rate limits

### Before Deployment

- [ ] Review CHANGELOG.md for breaking changes
- [ ] Update API client versions
- [ ] Test with Postman collection
- [ ] Verify WebSocket connections (if used)

---

**Version**: 1.0.0
**Last Updated**: 2025-12-08
**Maintainer**: Alexi
