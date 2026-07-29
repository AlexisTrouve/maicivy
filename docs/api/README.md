# maicivy API Documentation

Welcome to the complete API documentation for **maicivy** - CV interactif intelligent avec IA.

---

## 🚀 Quick Start

### Interactive Documentation (Recommended)

```
http://localhost:5000/api/docs/
```

Try endpoints directly in your browser with Swagger UI!

### Import Postman Collection

```
File → Import → maicivy.postman_collection.json
```

---

## 📚 Documentation Files

### Getting Started

| File | Description | Audience |
|------|-------------|----------|
| [API_OVERVIEW.md](API_OVERVIEW.md) | General API information | Everyone |
| [API_DOCUMENTATION_GUIDE.md](API_DOCUMENTATION_GUIDE.md) | How to use this documentation | Everyone |

### API Endpoints

| File | Description | Endpoints |
|------|-------------|-----------|
| [CV_API.md](CV_API.md) | CV and profile endpoints | 6 endpoints |
| [LETTERS_API.md](LETTERS_API.md) | AI letter generation | 8 endpoints |
| [ANALYTICS_API.md](ANALYTICS_API.md) | Analytics and stats | 7 endpoints |

### Technical References

| File | Description |
|------|-------------|
| [openapi.yaml](openapi.yaml) | OpenAPI 3.0 specification |
| [ERROR_CODES.md](ERROR_CODES.md) | Complete error codes (E001-E900) |
| [RATE_LIMITING.md](RATE_LIMITING.md) | Rate limiting details |
| [WEBSOCKET_PROTOCOL.md](WEBSOCKET_PROTOCOL.md) | WebSocket protocol spec |
| [CHANGELOG.md](CHANGELOG.md) | API version history |

### Tools

| File | Description |
|------|-------------|
| [maicivy.postman_collection.json](maicivy.postman_collection.json) | Postman collection |
| [generate-api-clients.sh](../../scripts/generate-api-clients.sh) | Client generator |

---

## 🎯 By Use Case

### I want to...

**...test the API quickly**
→ [Swagger UI](http://localhost:5000/api/docs/)

**...integrate the API in my frontend**
→ [API_OVERVIEW.md](API_OVERVIEW.md) + [CV_API.md](CV_API.md) / [LETTERS_API.md](LETTERS_API.md)

**...debug an error**
→ [ERROR_CODES.md](ERROR_CODES.md)

**...understand rate limits**
→ [RATE_LIMITING.md](RATE_LIMITING.md)

**...use WebSockets**
→ [WEBSOCKET_PROTOCOL.md](WEBSOCKET_PROTOCOL.md)

**...generate TypeScript types**
→ `bash scripts/generate-api-clients.sh`

**...test with Postman**
→ Import [maicivy.postman_collection.json](maicivy.postman_collection.json)

**...check recent changes**
→ [CHANGELOG.md](CHANGELOG.md)

---

## 📖 By Role

### Frontend Developer

1. Read: [API_OVERVIEW.md](API_OVERVIEW.md)
2. Endpoints: [CV_API.md](CV_API.md), [LETTERS_API.md](LETTERS_API.md), [ANALYTICS_API.md](ANALYTICS_API.md)
3. WebSocket: [WEBSOCKET_PROTOCOL.md](WEBSOCKET_PROTOCOL.md)
4. Generate types: `bash scripts/generate-api-clients.sh`

### Backend Developer

1. Spec: [openapi.yaml](openapi.yaml)
2. Errors: [ERROR_CODES.md](ERROR_CODES.md)
3. Rate Limits: [RATE_LIMITING.md](RATE_LIMITING.md)
4. Changelog: [CHANGELOG.md](CHANGELOG.md)

### QA / Tester

1. Test: [Swagger UI](http://localhost:5000/api/docs/)
2. Import: [maicivy.postman_collection.json](maicivy.postman_collection.json)
3. Errors: [ERROR_CODES.md](ERROR_CODES.md)

### DevOps

1. Health: `/health`, `/health/deep`
2. Rate Limits: [RATE_LIMITING.md](RATE_LIMITING.md)
3. WebSocket: [WEBSOCKET_PROTOCOL.md](WEBSOCKET_PROTOCOL.md)

---

## 🔍 Quick Reference

### Base URL

```
Development: http://localhost:5000
Production:  https://api.maicivy.dev
```

### API Version

```
Current: v1.0.0
Prefix: /api/v1
```

### Authentication

Session cookies (automatic)

### Rate Limits

- Letter generation: 5/day
- Cooldown: 2 minutes

### Endpoints (30+)

| Category | Count | Doc |
|----------|-------|-----|
| Health | 2 | Built-in |
| CV | 6 | [CV_API.md](CV_API.md) |
| Letters | 8 | [LETTERS_API.md](LETTERS_API.md) |
| Analytics | 7 | [ANALYTICS_API.md](ANALYTICS_API.md) |
| GitHub | 2 | [ANALYTICS_API.md](ANALYTICS_API.md) |
| WebSocket | 1 | [WEBSOCKET_PROTOCOL.md](WEBSOCKET_PROTOCOL.md) |

---

## 🛠️ Tools & Resources

### Interactive Testing

- **Swagger UI**: http://localhost:5000/api/docs/
- **Postman Collection**: [maicivy.postman_collection.json](maicivy.postman_collection.json)

### Code Generation

```bash
# Generate TypeScript, Go, Python clients
bash scripts/generate-api-clients.sh
```

### Validation

```bash
# Validate OpenAPI spec
swagger-cli validate openapi.yaml
```

---

## 📦 Installation

### Prerequisites

**For Using API:**
- None (REST API, use any HTTP client)

**For Client Generation:**
```bash
npm install -g openapi-typescript
go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest
npm install -g @openapitools/openapi-generator-cli
```

### Setup

```bash
# 1. Clone repository
git clone https://github.com/alexi/maicivy

# 2. Start backend
cd backend
go run cmd/main.go

# 3. Access Swagger UI
open http://localhost:5000/api/docs/
```

---

## 🐛 Troubleshooting

### Swagger UI not loading

**Check**: Backend is running on port 5000
```bash
curl http://localhost:5000/health
```

### "Access denied" error (E004)

**Solution**: Visit site 3 times to unlock AI features

**Check access**:
```bash
curl http://localhost:5000/api/v1/letters/access-status
```

### Rate limit exceeded (E005)

**Check status**:
```bash
curl http://localhost:5000/api/v1/letters/rate-limit-status
```

**Wait for**: Reset time in response

---

## 📞 Support

- **Documentation Issues**: GitHub Issues
- **API Questions**: contact@maicivy.dev
- **Live Testing**: [Swagger UI](http://localhost:5000/api/docs/)

---

## 🔄 Keeping Up to Date

### Check for Changes

```bash
# View recent changes
cat CHANGELOG.md

# Check current version
curl http://localhost:5000/health
```

### Update Clients

```bash
# Regenerate after spec changes
bash scripts/generate-api-clients.sh
```

---

## 📄 License

MIT License - See repository for details

---

**Version**: 1.0.0
**Last Updated**: 2025-12-08
**Maintainer**: Alexi

---

**Quick Links**:
- [Swagger UI](http://localhost:5000/api/docs/) - Try it now!
- [API Overview](API_OVERVIEW.md) - Start here
- [Documentation Guide](API_DOCUMENTATION_GUIDE.md) - How to use
- [OpenAPI Spec](openapi.yaml) - Machine-readable spec
