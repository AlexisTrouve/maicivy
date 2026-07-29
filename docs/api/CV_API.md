# CV API Documentation

## Endpoints

### Get Adaptive CV

Retrieve CV adapted to a specific theme.

```http
GET /api/v1/cv?theme={theme}
```

#### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| theme | string | No | Theme ID (default: fullstack) |

#### Themes

- `backend` - Backend Engineer
- `frontend` - Frontend Developer
- `fullstack` - Full-Stack Developer
- `cpp` - C++ Developer
- `artistique` - Creative/Artistic
- `devops` - DevOps Engineer

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/cv?theme=backend" \
  -H "Content-Type: application/json"
```

#### Example Response

```json
{
  "theme": "backend",
  "title": "Backend Engineer",
  "summary": "Experienced backend developer...",
  "experiences": [
    {
      "id": 1,
      "title": "Senior Backend Engineer",
      "company": "TechCorp",
      "description": "Led Go development...",
      "start_date": "2023-01-01",
      "end_date": "2025-12-08",
      "technologies": ["Go", "PostgreSQL", "Redis"],
      "tags": ["backend", "databases"],
      "category": "backend",
      "relevance_score": 0.95
    }
  ],
  "skills": [
    {
      "id": 1,
      "name": "Go",
      "level": "expert",
      "category": "backend",
      "tags": ["languages", "backend"],
      "years_experience": 5,
      "relevance_score": 0.98
    }
  ],
  "projects": [
    {
      "id": 1,
      "title": "maicivy",
      "description": "CV AI-powered",
      "github_url": "https://github.com/alexi/maicivy",
      "technologies": ["Go", "Next.js"],
      "stars": 42,
      "relevance_score": 0.88
    }
  ]
}
```

---

### Get Available Themes

Retrieve list of all CV themes.

```http
GET /api/v1/cv/themes
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/cv/themes"
```

#### Example Response

```json
{
  "themes": [
    {
      "id": "backend",
      "name": "Backend Engineer",
      "description": "Focus on server-side development",
      "icon": "🖥️",
      "color": "#4CAF50"
    }
  ],
  "count": 6
}
```

---

### Get All Experiences

Retrieve all professional experiences.

```http
GET /api/v1/experiences
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/experiences"
```

#### Example Response

```json
{
  "experiences": [...],
  "count": 8
}
```

---

### Get All Skills

Retrieve all skills.

```http
GET /api/v1/skills
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/skills"
```

#### Example Response

```json
{
  "skills": [...],
  "count": 25
}
```

---

### Get All Projects

Retrieve all projects.

```http
GET /api/v1/projects
```

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/projects"
```

#### Example Response

```json
{
  "projects": [...],
  "count": 12
}
```

---

### Export CV as PDF

Generate and download CV as PDF.

```http
GET /api/v1/cv/export?theme={theme}&format=pdf
```

#### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| theme | string | No | Theme ID (default: fullstack) |
| format | string | No | Export format (only pdf, default: pdf) |

#### Example Request

```bash
curl -X GET "http://localhost:5000/api/v1/cv/export?theme=backend&format=pdf" \
  -H "Accept: application/pdf" \
  -o cv_backend.pdf
```

#### Response

- Content-Type: `application/pdf`
- Content-Disposition: `attachment; filename=cv_backend.pdf`
- Binary PDF data

---

## Relevance Scoring

The adaptive CV uses a relevance scoring algorithm to rank items:

### Algorithm

```
relevance_score = (tag_matches * 0.4) +
                  (keyword_matches * 0.3) +
                  (category_match * 0.2) +
                  (recency * 0.1)
```

### Factors

- **Tag matches**: Tags overlapping with theme tags
- **Keyword matches**: Keywords in description matching theme
- **Category match**: Exact category match
- **Recency**: More recent items score higher

### Thresholds

- Items with score ≥ 0.5 are included
- Top 5 experiences, top 10 skills, top 6 projects

---

## Caching

CV data is cached in Redis:

- **Cache key**: `cv:theme:{theme_id}`
- **TTL**: 1 hour
- **Invalidation**: Manual or on data update

---

## Error Codes

| Code | Status | Description |
|------|--------|-------------|
| INVALID_THEME | 400 | Theme does not exist |
| INVALID_FORMAT | 400 | Export format not supported |
| PDF_GENERATION_ERROR | 500 | Failed to generate PDF |
| DB_ERROR | 500 | Database query failed |

---

## Rate Limiting

CV endpoints have **no rate limits** (public read-only data).

---

## Notes

- All endpoints are **GET** (read-only)
- No authentication required
- Data updated via admin panel (not via API)
- PDF generation uses `gofpdf` library
