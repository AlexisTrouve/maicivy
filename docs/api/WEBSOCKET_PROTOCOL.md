# WebSocket Protocol Documentation

## Analytics WebSocket

Real-time analytics updates via WebSocket.

---

## Connection

### Endpoint

```
ws://localhost:5000/ws/analytics
wss://api.maicivy.dev/ws/analytics (production)
```

### JavaScript Example

```javascript
const ws = new WebSocket('ws://localhost:5000/ws/analytics');

ws.onopen = () => {
  console.log('Connected to analytics WebSocket');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket connection closed');
};
```

---

## Message Types

### Server → Client

#### 1. Realtime Visitors

Sent every 5 seconds with current visitor count.

```json
{
  "type": "realtime_update",
  "data": {
    "active_visitors": 5,
    "timestamp": "2025-12-08T10:30:00Z"
  }
}
```

#### 2. New Event

Sent when a new analytics event occurs.

```json
{
  "type": "new_event",
  "data": {
    "event_type": "page_view",
    "page_url": "/cv",
    "timestamp": "2025-12-08T10:30:05Z"
  }
}
```

#### 3. Theme Stats Update

Sent when CV theme stats change.

```json
{
  "type": "theme_stats",
  "data": {
    "theme": "backend",
    "count": 88,
    "percentage": 25.8
  }
}
```

#### 4. Letter Generated

Sent when a letter is generated (anonymized).

```json
{
  "type": "letter_generated",
  "data": {
    "letter_type": "motivation",
    "timestamp": "2025-12-08T10:31:00Z"
  }
}
```

#### 5. Heartbeat (Ping)

Sent every 30 seconds to keep connection alive.

```json
{
  "type": "ping",
  "timestamp": "2025-12-08T10:30:30Z"
}
```

### Client → Server

#### Pong (Heartbeat Response)

```json
{
  "type": "pong",
  "timestamp": "2025-12-08T10:30:30Z"
}
```

---

## Connection Lifecycle

### 1. Connect

```javascript
const ws = new WebSocket('ws://localhost:5000/ws/analytics');
```

### 2. Initial Data

Server sends current state immediately:

```json
{
  "type": "init",
  "data": {
    "active_visitors": 5,
    "timestamp": "2025-12-08T10:30:00Z"
  }
}
```

### 3. Periodic Updates

- Realtime stats: every 5 seconds
- Event notifications: as they occur
- Heartbeat: every 30 seconds

### 4. Disconnect

```javascript
ws.close();
```

Server cleans up connection.

---

## React Hook Example

```typescript
import { useEffect, useState } from 'react';

interface RealtimeStats {
  active_visitors: number;
  timestamp: string;
}

export function useAnalyticsWebSocket() {
  const [stats, setStats] = useState<RealtimeStats | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const ws = new WebSocket('ws://localhost:5000/ws/analytics');

    ws.onopen = () => {
      setConnected(true);
    };

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);

      if (message.type === 'realtime_update') {
        setStats(message.data);
      } else if (message.type === 'ping') {
        ws.send(JSON.stringify({ type: 'pong' }));
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      setConnected(false);
    };

    ws.onclose = () => {
      setConnected(false);
      // Reconnect after 5 seconds
      setTimeout(() => {
        // Create new connection
      }, 5000);
    };

    return () => {
      ws.close();
    };
  }, []);

  return { stats, connected };
}
```

---

## Reconnection Strategy

### Exponential Backoff

```javascript
let reconnectDelay = 1000; // Start with 1 second
const maxDelay = 30000; // Max 30 seconds

function connect() {
  const ws = new WebSocket('ws://localhost:5000/ws/analytics');

  ws.onclose = () => {
    console.log(`Reconnecting in ${reconnectDelay}ms...`);
    setTimeout(connect, reconnectDelay);

    // Exponential backoff
    reconnectDelay = Math.min(reconnectDelay * 2, maxDelay);
  };

  ws.onopen = () => {
    // Reset delay on successful connection
    reconnectDelay = 1000;
  };

  return ws;
}
```

---

## Heartbeat Mechanism

### Purpose

- Keep connection alive
- Detect broken connections
- Prevent idle timeouts

### Server Behavior

- Sends `ping` every 30 seconds
- Expects `pong` within 10 seconds
- Closes connection if no pong

### Client Behavior

```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);

  if (message.type === 'ping') {
    ws.send(JSON.stringify({
      type: 'pong',
      timestamp: new Date().toISOString()
    }));
  }
};
```

---

## Error Handling

### Connection Errors

```javascript
ws.onerror = (error) => {
  console.error('WebSocket error:', error);

  // Show user notification
  showNotification('Connection lost. Reconnecting...');
};
```

### Message Parsing Errors

```javascript
ws.onmessage = (event) => {
  try {
    const message = JSON.parse(event.data);
    // Process message
  } catch (error) {
    console.error('Failed to parse message:', error);
    // Ignore malformed messages
  }
};
```

---

## Security

### CORS

WebSocket connections respect CORS policies:
- Allowed origins: configured in server
- Development: `localhost:3000`
- Production: `maicivy.dev`

### Authentication

No authentication required (public analytics).

### Rate Limiting

WebSocket connections are rate-limited:
- Max 10 connections per IP
- Max 1 message/second from client

---

## Testing

### wscat (CLI Tool)

```bash
# Install
npm install -g wscat

# Connect
wscat -c ws://localhost:5000/ws/analytics

# You'll receive messages:
> {"type":"init","data":{...}}
> {"type":"realtime_update","data":{...}}
```

### Browser DevTools

```javascript
// Open console
const ws = new WebSocket('ws://localhost:5000/ws/analytics');
ws.onmessage = (e) => console.log(JSON.parse(e.data));
```

---

## Monitoring

### Prometheus Metrics

```promql
# Active WebSocket connections
websocket_connections_active

# Messages sent/received
websocket_messages_total{direction="sent|received"}

# Connection errors
websocket_errors_total
```

---

## Performance

### Scalability

- Supports 10,000+ concurrent connections
- Uses goroutines (lightweight)
- Redis pub/sub for multi-server setups

### Bandwidth

Average message size: ~100 bytes
- Realtime updates (every 5s): 100 bytes/5s = 20 bytes/s
- Heartbeat (every 30s): 50 bytes/30s = 1.67 bytes/s
- **Total**: ~22 bytes/s per connection

1000 connections: ~22 KB/s = ~190 GB/month

---

## Complete Example (React)

```typescript
import React, { useEffect, useState } from 'react';

interface RealtimeStats {
  active_visitors: number;
  timestamp: string;
}

export const RealtimeVisitors: React.FC = () => {
  const [stats, setStats] = useState<RealtimeStats | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    let ws: WebSocket;
    let reconnectTimeout: NodeJS.Timeout;
    let reconnectDelay = 1000;

    function connect() {
      ws = new WebSocket('ws://localhost:5000/ws/analytics');

      ws.onopen = () => {
        console.log('WebSocket connected');
        setConnected(true);
        reconnectDelay = 1000;
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);

          switch (message.type) {
            case 'init':
            case 'realtime_update':
              setStats(message.data);
              break;
            case 'ping':
              ws.send(JSON.stringify({ type: 'pong' }));
              break;
          }
        } catch (error) {
          console.error('Failed to parse message:', error);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        setConnected(false);

        // Reconnect with exponential backoff
        reconnectTimeout = setTimeout(() => {
          connect();
        }, reconnectDelay);

        reconnectDelay = Math.min(reconnectDelay * 2, 30000);
      };
    }

    connect();

    return () => {
      if (ws) ws.close();
      if (reconnectTimeout) clearTimeout(reconnectTimeout);
    };
  }, []);

  if (!connected) {
    return <div>Connecting to realtime analytics...</div>;
  }

  if (!stats) {
    return <div>Loading...</div>;
  }

  return (
    <div>
      <h2>Realtime Visitors</h2>
      <p className="text-4xl font-bold">{stats.active_visitors}</p>
      <p className="text-sm text-gray-500">
        Last updated: {new Date(stats.timestamp).toLocaleTimeString()}
      </p>
    </div>
  );
};
```
