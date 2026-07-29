# Analytics API Documentation

## Overview

Public real-time analytics and statistics endpoints.

**All endpoints are public** - no authentication required.

---

## Endpoints

### Get Realtime Stats

```http
GET /api/v1/analytics/realtime
```

#### Response

```json
{
  "success": true,
  "data": {
    "active_visitors": 5,
    "last_updated": "2025-12-08T10:30:00Z"
  }
}
```

---

### Get Aggregated Stats

```http
GET /api/v1/analytics/stats?period={period}
```

#### Parameters

| Name | Type | Default | Values |
|------|------|---------|--------|
| period | string | day | day, week, month |

#### Response

```json
{
  "success": true,
  "data": {
    "period": "day",
    "total_visitors": 256,
    "unique_visitors": 128,
    "page_views": 512,
    "average_session_length": 245.5,
    "letters_generated": 42
  }
}
```

---

### Get Top Themes

```http
GET /api/v1/analytics/themes?limit={limit}
```

#### Parameters

| Name | Type | Default | Range |
|------|------|---------|-------|
| limit | int | 5 | 1-20 |

#### Response

```json
{
  "success": true,
  "data": [
    {
      "theme": "backend",
      "count": 87,
      "percentage": 25.5
    }
  ]
}
```

---

### Get Letters Stats

```http
GET /api/v1/analytics/letters?period={period}
```

#### Response

```json
{
  "success": true,
  "data": {
    "total": 84,
    "motivation": 42,
    "anti_motivation": 42,
    "period": "day"
  }
}
```

---

### Get Timeline

```http
GET /api/v1/analytics/timeline?limit={limit}&offset={offset}
```

#### Parameters

| Name | Type | Default | Range |
|------|------|---------|-------|
| limit | int | 50 | 1-100 |
| offset | int | 0 | ≥0 |

#### Response

```json
{
  "success": true,
  "data": [
    {
      "id": "evt_123",
      "event_type": "page_view",
      "created_at": "2025-12-08T10:30:00Z",
      "page_url": "/cv"
    }
  ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "count": 50
  }
}
```

---

### Get Heatmap Data

```http
GET /api/v1/analytics/heatmap?page_url={url}&hours={hours}
```

#### Parameters

| Name | Type | Default | Range |
|------|------|---------|-------|
| page_url | string | "" | Any URL |
| hours | int | 24 | 1-168 |

#### Response

```json
{
  "success": true,
  "data": [
    {
      "x": 450,
      "y": 200,
      "count": 15,
      "page_url": "/cv"
    }
  ],
  "meta": {
    "page_url": "/cv",
    "hours": 24,
    "count": 342
  }
}
```

---

### Track Event

```http
POST /api/v1/analytics/event
```

#### Request Body

```json
{
  "event_type": "button_click",
  "event_data": {
    "button": "cta",
    "x": 450,
    "y": 200
  },
  "page_url": "/cv"
}
```

#### Response

```json
{
  "success": true,
  "message": "Event tracked successfully"
}
```

---

## Event Types

| Type | Description |
|------|-------------|
| page_view | Page visited |
| button_click | Button clicked |
| cv_theme_selected | CV theme changed |
| letter_generated | Letter generated |
| pdf_downloaded | PDF downloaded |
| scroll | Page scrolled |
| hover | Element hovered |

---

## WebSocket

### Endpoint

```
ws://localhost:5000/ws/analytics
```

### Protocol

See [WEBSOCKET_PROTOCOL.md](WEBSOCKET_PROTOCOL.md)

---

## Caching

Analytics data is cached in Redis:

- Realtime stats: 5 seconds
- Aggregated stats: 5 minutes
- Top themes: 10 minutes

---

## Error Codes

| Code | Status | Description |
|------|--------|-------------|
| INVALID_PERIOD | 400 | Invalid period parameter |
| INVALID_EVENT | 400 | Invalid event type |
| MISSING_VISITOR | 400 | No visitor session |
