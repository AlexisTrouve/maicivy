# API Changelog

All notable changes to the maicivy API will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planned

- Multi-language support (EN/FR)
- Webhook notifications
- Public API authentication (API keys)
- GraphQL endpoint
- Batch operations

---

## [1.0.0] - 2025-12-08

### Added

#### CV API
- `GET /api/v1/cv` - Adaptive CV by theme
- `GET /api/v1/cv/themes` - List available themes
- `GET /api/v1/experiences` - All experiences
- `GET /api/v1/skills` - All skills
- `GET /api/v1/projects` - All projects
- `GET /api/v1/cv/export` - PDF export

#### Letters API
- `POST /api/v1/letters/generate` - Generate letter pair (async)
- `GET /api/v1/letters/jobs/:jobId` - Job status polling
- `GET /api/v1/letters/:id` - Get letter by ID
- `GET /api/v1/letters/:id/pdf` - Download letter PDF
- `GET /api/v1/letters/pair` - Get letter pair by company
- `GET /api/v1/letters/history` - Generation history
- `GET /api/v1/letters/access-status` - Check AI access
- `GET /api/v1/letters/rate-limit-status` - Check rate limits

#### Analytics API
- `GET /api/v1/analytics/realtime` - Realtime visitor stats
- `GET /api/v1/analytics/stats` - Aggregated statistics
- `GET /api/v1/analytics/themes` - Top CV themes
- `GET /api/v1/analytics/letters` - Letters generation stats
- `GET /api/v1/analytics/timeline` - Recent events
- `GET /api/v1/analytics/heatmap` - Click heatmap data
- `POST /api/v1/analytics/event` - Track custom events
- `WS /ws/analytics` - Real-time WebSocket

#### GitHub API
- `POST /api/v1/github/sync` - Trigger project sync
- `GET /api/v1/github/status` - Sync status

#### Profile API
- Profile detection system
- Target profile bypass (recruiters, tech leads)

#### Timeline API
- Interactive career timeline endpoints

#### Health Checks
- `GET /health` - Shallow health check
- `GET /health/deep` - Deep health check (DB + Redis)

### Features

- **Access Gate**: 3-visit minimum for AI features
- **Rate Limiting**: 5 generations/day with 2-min cooldown
- **Profile Detection**: Automatic bypass for target profiles
- **Async Processing**: Job-based queue for letter generation
- **Redis Caching**: 1-hour cache for CV data
- **WebSocket**: Real-time analytics updates
- **PDF Generation**: CV and letter PDF exports
- **Session Tracking**: Cookie-based visitor tracking

### Security

- Session cookie authentication
- Rate limiting on AI endpoints
- Input validation and sanitization
- CORS configuration
- SQL injection protection (GORM)
- XSS prevention

### Documentation

- OpenAPI 3.0 specification
- Complete API documentation
- Error codes reference
- Rate limiting guide
- WebSocket protocol documentation
- Client examples (JS, Python, Go)

---

## [0.9.0] - 2025-12-06 (Beta)

### Added

- Initial beta release
- Basic CV endpoints
- Letter generation (synchronous)
- Simple analytics

### Changed

- Migrated letter generation to async (job queue)

### Fixed

- Rate limiting edge cases
- Session cookie expiration

---

## [0.5.0] - 2025-12-01 (Alpha)

### Added

- Alpha release for testing
- Core CV API
- Database schema
- Redis integration

---

## Migration Guides

### Migrating from v0.9 to v1.0

#### Breaking Changes

**Letter Generation (Async)**

Old (0.9):
```http
POST /api/v1/letters/generate
→ Returns letters immediately (sync)
```

New (1.0):
```http
POST /api/v1/letters/generate
→ Returns job_id (async)

GET /api/v1/letters/jobs/{jobId}
→ Poll for completion
```

**Migration Steps**:

1. Update client code to handle async flow
2. Implement polling mechanism
3. Handle job statuses (queued, processing, completed, failed)

Example:

```javascript
// Old (v0.9)
const response = await fetch('/api/v1/letters/generate', {
  method: 'POST',
  body: JSON.stringify({ company_name: 'Google' })
});
const { motivation_letter, anti_motivation_letter } = await response.json();

// New (v1.0)
const response = await fetch('/api/v1/letters/generate', {
  method: 'POST',
  body: JSON.stringify({ company_name: 'Google' })
});
const { job_id } = await response.json();

// Poll for completion
const checkStatus = async () => {
  const statusRes = await fetch(`/api/v1/letters/jobs/${job_id}`);
  const status = await statusRes.json();

  if (status.status === 'completed') {
    const letter1 = await fetch(`/api/v1/letters/${status.letter_motivation_id}`);
    const letter2 = await fetch(`/api/v1/letters/${status.letter_anti_motivation_id}`);
    // Process letters
  } else if (status.status === 'failed') {
    // Handle error
  } else {
    // Still processing, poll again
    setTimeout(checkStatus, 2000);
  }
};
checkStatus();
```

---

## Deprecation Notices

### v1.0

No deprecations in this version.

### Future Deprecations

- **v1.1**: Synchronous letter generation endpoint will be deprecated
- **v2.0**: Old response format for CV endpoints will be removed

---

## Support Policy

- **Current version (1.0.x)**: Full support
- **Previous version (0.9.x)**: Security fixes only (until 2026-06-08)
- **Older versions (<0.9)**: No support

---

## Versioning

API versions follow semantic versioning:

- **Major version (v1, v2)**: Breaking changes
- **Minor version (v1.1, v1.2)**: New features (backward compatible)
- **Patch version (v1.0.1, v1.0.2)**: Bug fixes

---

## Release Schedule

- **Major releases**: Annually
- **Minor releases**: Quarterly
- **Patch releases**: As needed

---

## How to Stay Updated

- **GitHub Releases**: Watch repository
- **RSS Feed**: `/api/changelog.rss`
- **Newsletter**: Subscribe at https://maicivy.dev
- **Twitter**: @maicivy_dev

---

## Contact

Questions about the changelog?
- Email: contact@maicivy.dev
- GitHub: https://github.com/alexi/maicivy/issues
