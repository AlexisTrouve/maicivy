# WebSocket Handlers

Ce dossier contient les handlers WebSocket pour les fonctionnalités temps réel de maicivy.

## Structure

```
websocket/
├── analytics.go        # Handler WebSocket pour analytics temps réel
└── README.md          # Ce fichier
```

## Analytics WebSocket

### Endpoint
```
ws://localhost:8080/ws/analytics
```

### Fonctionnalités
- Broadcast des statistiques analytics en temps réel
- Heartbeat automatique toutes les 5 secondes
- Support multi-clients via goroutines thread-safe
- Intégration Redis Pub/Sub pour scalabilité horizontale

### Utilisation

#### Côté Serveur (Go)
```go
import (
    "maicivy/internal/services"
    "maicivy/internal/websocket"
)

// Créer le handler
analyticsService := services.NewAnalyticsService(db, redisClient)
wsHandler := websocket.NewAnalyticsWSHandler(analyticsService, redisClient)

// Enregistrer les routes
wsHandler.RegisterRoutes(app)

// Fermer proprement au shutdown
defer wsHandler.Close()
```

#### Côté Client (JavaScript)
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/analytics');

ws.onopen = () => {
    console.log('Connected to analytics WebSocket');
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    switch(data.type) {
        case 'initial_stats':
            console.log('Initial stats:', data.data);
            break;
        case 'heartbeat':
            updateDashboard(data.data);
            break;
        default:
            console.log('Event:', data);
    }
};

ws.onerror = (error) => {
    console.error('WebSocket error:', error);
};

ws.onclose = () => {
    console.log('WebSocket closed');
    // Reconnection logic here
};

// Envoyer message au serveur
ws.send(JSON.stringify({ type: 'refresh_stats' }));
```

### Protocol

#### Messages envoyés par le serveur

**Initial Stats** (à la connexion)
```json
{
  "type": "initial_stats",
  "data": {
    "current_visitors": 12,
    "unique_today": 145,
    "total_events": 2340,
    "letters_today": 23,
    "timestamp": 1733683200
  }
}
```

**Heartbeat** (toutes les 5s)
```json
{
  "type": "heartbeat",
  "data": { ... }
}
```

**Event** (temps réel via Pub/Sub)
```json
{
  "type": "page_view",
  "visitor_id": "uuid",
  "timestamp": "2025-12-08T12:34:56Z",
  "data": { ... }
}
```

#### Messages envoyés par le client

**Refresh Stats**
```json
{ "type": "refresh_stats" }
```

**Ping**
```json
{ "type": "ping" }
```

### Architecture

```
Client 1 ──┐
           │
Client 2 ──┼──► AnalyticsWSHandler ◄──► Redis Pub/Sub ◄──► Analytics Service
           │         │                                              │
Client N ──┘         │                                              │
                     └──────────────────────────────────────────────┘
                            Broadcast Channel (goroutine)
```

### Scalabilité

Le handler utilise Redis Pub/Sub pour permettre la scalabilité horizontale :
- Chaque instance backend a son propre handler WebSocket
- Les événements sont publiés via Redis Pub/Sub
- Tous les handlers reçoivent les événements et les broadcast à leurs clients
- Les clients restent connectés à la même instance (sticky session recommandée)

### Performance

- **Clients simultanés:** 50+ supportés par instance
- **Latence broadcast:** < 50ms
- **Memory per client:** ~4KB
- **Heartbeat interval:** 5 secondes (configurable)

### Testing

```bash
# Installer wscat
npm install -g wscat

# Se connecter
wscat -c ws://localhost:8080/ws/analytics

# Envoyer message
> {"type":"refresh_stats"}

# Recevoir messages
< {"type":"initial_stats","data":{...}}
< {"type":"heartbeat","data":{...}}
```

### Monitoring

Le handler expose une méthode pour monitoring :
```go
connectedClients := wsHandler.GetConnectedClients()
fmt.Printf("Connected clients: %d\n", connectedClients)
```

Métrique Prometheus disponible :
```promql
maicivy_websocket_connections
```

## Bonnes Pratiques

### Reconnection Logic (Client)
```javascript
class AnalyticsWebSocket {
    constructor(url) {
        this.url = url;
        this.reconnectInterval = 5000;
        this.connect();
    }

    connect() {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            console.log('Connected');
            this.reconnectInterval = 5000;
        };

        this.ws.onclose = () => {
            console.log('Disconnected, reconnecting in', this.reconnectInterval);
            setTimeout(() => this.connect(), this.reconnectInterval);
            this.reconnectInterval = Math.min(this.reconnectInterval * 1.5, 30000);
        };

        this.ws.onerror = (error) => {
            console.error('Error:', error);
            this.ws.close();
        };
    }

    send(data) {
        if (this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        }
    }
}

const analytics = new AnalyticsWebSocket('ws://localhost:8080/ws/analytics');
```

### Heartbeat Detection (Client)
```javascript
let lastHeartbeat = Date.now();

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    if (data.type === 'heartbeat') {
        lastHeartbeat = Date.now();
    }
};

// Vérifier si le serveur est toujours vivant
setInterval(() => {
    const timeSinceLastHeartbeat = Date.now() - lastHeartbeat;
    if (timeSinceLastHeartbeat > 15000) { // 3x heartbeat interval
        console.warn('No heartbeat received, connection may be dead');
        ws.close(); // Force reconnection
    }
}, 10000);
```

## Troubleshooting

### Connection refused
- Vérifier que le serveur est démarré
- Vérifier que le port est correct (8080 par défaut)
- Vérifier les logs serveur pour erreurs

### Messages non reçus
- Vérifier Redis Pub/Sub : `redis-cli PUBSUB CHANNELS`
- Vérifier logs serveur : "Broadcasted message to all clients"
- Tester avec wscat pour isoler problème client/serveur

### High memory usage
- Vérifier nombre de clients : `wsHandler.GetConnectedClients()`
- Limiter connexions simultanées si nécessaire
- Vérifier fuites mémoire (clients non fermés)

## Futures Améliorations

- [ ] Authentification WebSocket (JWT)
- [ ] Filtres côté client (subscribe à événements spécifiques)
- [ ] Compression messages (permessage-deflate)
- [ ] Binary protocol (Protocol Buffers)
- [ ] Rate limiting par client
