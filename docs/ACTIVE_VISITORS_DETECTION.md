# Système de Détection de Visiteurs Actifs - maicivy

**Version:** 1.0
**Date:** 2026-01-16
**Auteur:** Claude Sonnet 4.5

---

## 📋 Table des Matières

1. [Vue d'ensemble](#vue-densemble)
2. [Architecture](#architecture)
3. [Technologies utilisées](#technologies-utilisées)
4. [Fonctionnement](#fonctionnement)
5. [Implémentation Backend](#implémentation-backend)
6. [Implémentation Frontend](#implémentation-frontend)
7. [Configuration](#configuration)
8. [Monitoring](#monitoring)
9. [Best Practices](#best-practices)
10. [Troubleshooting](#troubleshooting)

---

## Vue d'ensemble

Le système de détection de visiteurs actifs permet de tracker en **temps réel** le nombre de visiteurs actuellement présents sur le site maicivy. Il utilise une combinaison de techniques modernes basées sur les **best practices 2026** :

### Caractéristiques principales

- ✅ **Heartbeat client-serveur** : Le client envoie un signal toutes les 30 secondes
- ✅ **WebSocket bidirectionnel** : Communication temps réel serveur ↔ client
- ✅ **Redis Sorted Set** : Stockage optimisé avec timestamps pour détection rapide
- ✅ **Détection automatique** : Nettoyage des visiteurs inactifs > 5 minutes
- ✅ **Événements temps réel** : Notifications de connexion/déconnexion via Pub/Sub
- ✅ **Gestion de la visibilité** : Arrêt automatique si page cachée (économie ressources)

### Pourquoi ce système ?

**Problème** : Comment savoir en temps réel combien de personnes consultent actuellement votre CV ?

**Solution** : Un système de heartbeat qui :
1. Marque un visiteur comme actif dès qu'il arrive
2. Maintient son statut actif via des heartbeats réguliers
3. Le marque automatiquement comme inactif après 5 minutes sans heartbeat
4. Notifie tous les clients connectés des changements en temps réel

---

## Architecture

```
┌─────────────────┐
│   Frontend      │
│   (Next.js)     │
└────────┬────────┘
         │
         │ 1. Heartbeat HTTP (30s)
         │    POST /api/v1/visitors/heartbeat
         │
         ├──────────────────────────┐
         │                          │
         ▼                          ▼
┌────────────────┐         ┌────────────────┐
│  Visitor API   │         │  WebSocket     │
│  Handler       │         │  Analytics     │
└────────┬───────┘         └────────┬───────┘
         │                          │
         │ 2. MarkVisitorActive()   │ ping/pong + heartbeat
         ▼                          ▼
┌─────────────────────────────────────┐
│    Analytics Service                │
│  - MarkVisitorActive()              │
│  - CleanupInactiveVisitors()        │
│  - GetRealtimeStats()               │
└────────┬────────────────────────────┘
         │
         │ 3. ZADD + ZREM
         ▼
┌─────────────────────────────────────┐
│    Redis                            │
│  - Sorted Set: realtime:visitors    │
│    Score = timestamp Unix           │
│    Member = visitor_id (UUID)       │
│                                     │
│  - Pub/Sub: analytics:realtime      │
│    Events: visitor_connected,       │
│             visitor_disconnected    │
└─────────────────────────────────────┘
         │
         │ 4. Redis Pub/Sub
         ▼
┌─────────────────────────────────────┐
│    WebSocket Clients                │
│  - Reçoivent les événements         │
│  - Mettent à jour le compteur       │
└─────────────────────────────────────┘

Background Job (1 minute):
┌─────────────────────────────────────┐
│  VisitorCleanupJob                  │
│  - CleanupInactiveVisitors()        │
│  - Publie events déconnexion        │
└─────────────────────────────────────┘
```

---

## Technologies utilisées

### Backend (Go)

| Technologie | Rôle | Fichier |
|-------------|------|---------|
| **Fiber** | Framework HTTP | `backend/internal/api/visitor.go` |
| **Redis** | Cache + Pub/Sub | `backend/internal/services/analytics.go` |
| **WebSocket** | Communication temps réel | `backend/internal/websocket/analytics.go` |
| **GORM** | ORM PostgreSQL | `backend/internal/models/visitor.go` |
| **Zerolog** | Logging | Tous les fichiers |

### Frontend (Next.js + TypeScript)

| Technologie | Rôle | Fichier |
|-------------|------|---------|
| **React Hooks** | Gestion d'état | `frontend/hooks/useVisitorHeartbeat.ts` |
| **Fetch API** | Requêtes HTTP | `frontend/lib/api.ts` |
| **WebSocket** | Communication temps réel | `frontend/hooks/useAnalyticsWebSocket.ts` |
| **Next.js App Router** | Routing | N/A |

---

## Fonctionnement

### 1. Marquage initial (première visite)

```mermaid
sequenceDiagram
    participant C as Client (Browser)
    participant M as Tracking Middleware
    participant DB as PostgreSQL
    participant R as Redis
    participant A as Analytics Service

    C->>M: HTTP Request (any page)
    M->>M: Créer/récupérer session_id (cookie)
    M->>DB: Créer/mettre à jour Visitor
    M->>R: INCR visitor:session_id:count
    M->>A: MarkVisitorActive(visitor_id)
    A->>R: ZADD realtime:visitors (score=now, member=visitor_id)
    A->>R: PUBLISH analytics:realtime (visitor_connected)
    M-->>C: Response + Set-Cookie
```

### 2. Heartbeat périodique (toutes les 30s)

```javascript
// Frontend automatique
useVisitorHeartbeat({
  interval: 30000, // 30 secondes
  onHeartbeat: (activeVisitors) => {
    console.log('Active visitors:', activeVisitors);
  }
});
```

```
Client                    API Endpoint                 Redis
  │                            │                          │
  │ POST /visitors/heartbeat   │                          │
  ├───────────────────────────>│                          │
  │                            │ MarkVisitorActive()      │
  │                            ├─────────────────────────>│
  │                            │ ZADD + update timestamp  │
  │                            │<─────────────────────────┤
  │ { success: true,           │                          │
  │   active_visitors: 42 }    │                          │
  │<───────────────────────────┤                          │
  │                            │                          │

  ... 30 secondes plus tard ...

  │ POST /visitors/heartbeat   │                          │
  ├───────────────────────────>│                          │
  │                            │ MarkVisitorActive()      │
  │                            ├─────────────────────────>│
```

### 3. Nettoyage automatique (toutes les 1 minute)

```
VisitorCleanupJob (background)      Redis
        │                             │
        │ Every 1 minute              │
        ├────────────────────────────>│
        │ CleanupInactiveVisitors()   │
        │ ZRANGEBYSCORE (< now-5min)  │
        │<────────────────────────────┤
        │ [visitor_1, visitor_2, ...]  │
        │                             │
        │ ZREMRANGEBYSCORE            │
        ├────────────────────────────>│
        │                             │
        │ PUBLISH visitor_disconnected│
        ├────────────────────────────>│
        │ (pour chaque visiteur)      │
```

### 4. WebSocket temps réel

```
Client WebSocket          Analytics WebSocket Handler         Redis Pub/Sub
      │                            │                               │
      │ Connect /ws/analytics      │                               │
      ├───────────────────────────>│                               │
      │                            │ Subscribe analytics:realtime  │
      │                            ├──────────────────────────────>│
      │ { type: "initial_stats" }  │                               │
      │<───────────────────────────┤                               │
      │                            │                               │
      │                            │ Heartbeat (5s)                │
      │ { type: "heartbeat",       │                               │
      │   data: { current_visitors │                               │
      │           : 42 } }          │                               │
      │<───────────────────────────┤                               │
      │                            │                               │
      │ { type: "ping",            │                               │
      │   visitor_id: "uuid" }     │                               │
      ├───────────────────────────>│                               │
      │                            │ MarkVisitorActive()           │
      │ { type: "pong" }           │                               │
      │<───────────────────────────┤                               │
      │                            │                               │
      │                            │ PUBLISH visitor_connected     │
      │                            │<──────────────────────────────┤
      │ { type: "visitor_connected"│                               │
      │   visitor_id: "..." }      │                               │
      │<───────────────────────────┤                               │
```

---

## Implémentation Backend

### 1. Endpoint Heartbeat

**Fichier:** `backend/internal/api/visitor.go`

```go
// POST /api/v1/visitors/heartbeat
func (vh *VisitorHandler) Heartbeat(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Récupérer visitor_id du context (set par tracking middleware)
    visitorID := c.Locals("visitor_id").(uuid.UUID)

    // Marquer comme actif
    if err := vh.analyticsService.MarkVisitorActive(ctx, visitorID); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to update visitor status"})
    }

    // Récupérer stats temps réel
    stats, _ := vh.analyticsService.GetRealtimeStats(ctx)
    activeVisitors := stats["current_visitors"].(int64)

    return c.JSON(HeartbeatResponse{
        Success:        true,
        Timestamp:      time.Now().Unix(),
        ActiveVisitors: int(activeVisitors),
    })
}
```

### 2. Service Analytics

**Fichier:** `backend/internal/services/analytics.go`

```go
// MarkVisitorActive marque un visiteur comme actif
func (s *AnalyticsService) MarkVisitorActive(ctx context.Context, visitorID uuid.UUID) error {
    key := "analytics:realtime:visitors"
    now := time.Now().Unix()

    // Vérifier si nouvelle connexion
    wasActive, _ := s.redis.ZScore(ctx, key, visitorID.String()).Result()
    isNewConnection := wasActive == 0

    // Ajouter/mettre à jour dans Redis Sorted Set
    s.redis.ZAdd(ctx, key, redis.Z{
        Score:  float64(now),
        Member: visitorID.String(),
    })

    // Nettoyer visiteurs inactifs > 5 minutes
    fiveMinutesAgo := now - 5*60
    s.redis.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", fiveMinutesAgo))

    // Publier événement si nouveau
    if isNewConnection {
        s.publishVisitorEvent(ctx, "visitor_connected", visitorID)
    }

    return nil
}

// CleanupInactiveVisitors nettoie et publie événements
func (s *AnalyticsService) CleanupInactiveVisitors(ctx context.Context) (int64, error) {
    key := "analytics:realtime:visitors"
    now := time.Now().Unix()
    fiveMinutesAgo := now - 5*60

    // Récupérer visiteurs inactifs
    inactiveVisitors, err := s.redis.ZRangeByScore(ctx, key, &redis.ZRangeBy{
        Min: "-inf",
        Max: fmt.Sprintf("%d", fiveMinutesAgo),
    }).Result()

    if len(inactiveVisitors) == 0 {
        return 0, nil
    }

    // Supprimer
    removed, err := s.redis.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", fiveMinutesAgo)).Result()

    // Publier événements de déconnexion
    for _, visitorIDStr := range inactiveVisitors {
        visitorID, _ := uuid.Parse(visitorIDStr)
        s.publishVisitorEvent(ctx, "visitor_disconnected", visitorID)
    }

    return removed, nil
}
```

### 3. Background Job

**Fichier:** `backend/internal/jobs/visitor_cleanup.go`

```go
type VisitorCleanupJob struct {
    service  *services.AnalyticsService
    interval time.Duration
}

func (j *VisitorCleanupJob) Start(ctx context.Context) {
    ticker := time.NewTicker(j.interval) // 1 minute
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            removed, err := j.service.CleanupInactiveVisitors(ctx)
            if err != nil {
                log.Error().Err(err).Msg("Failed to cleanup inactive visitors")
            } else if removed > 0 {
                log.Info().Int64("removed", removed).Msg("Cleaned up inactive visitors")
            }

        case <-ctx.Done():
            return
        }
    }
}
```

### 4. WebSocket Handler

**Fichier:** `backend/internal/websocket/analytics.go`

```go
func (h *AnalyticsWSHandler) handleClientMessage(c *websocket.Conn, message []byte) {
    var req struct {
        Type      string `json:"type"`
        VisitorID string `json:"visitor_id,omitempty"`
    }
    json.Unmarshal(message, &req)

    switch req.Type {
    case "ping", "heartbeat":
        // Marquer visiteur actif via WebSocket
        if req.VisitorID != "" {
            visitorUUID, _ := uuid.Parse(req.VisitorID)
            h.service.MarkVisitorActive(ctx, visitorUUID)
        }

        // Répondre pong
        response, _ := json.Marshal(map[string]interface{}{
            "type": "pong",
            "time": time.Now().Unix(),
        })
        c.WriteMessage(websocket.TextMessage, response)
    }
}
```

---

## Implémentation Frontend

### 1. Hook React `useVisitorHeartbeat`

**Fichier:** `frontend/hooks/useVisitorHeartbeat.ts`

```typescript
export function useVisitorHeartbeat(options: UseVisitorHeartbeatOptions = {}) {
  const {
    interval = 30000, // 30 secondes
    enabled = true,
    trackPageUrl = true,
    onHeartbeat,
    onError,
  } = options;

  const pathname = usePathname();
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const isPageVisibleRef = useRef(true);

  // Fonction d'envoi heartbeat
  const sendHeartbeat = useCallback(async () => {
    if (!enabled || !isPageVisibleRef.current) return;

    try {
      const pageUrl = trackPageUrl ? pathname : undefined;
      const response = await visitorsApi.sendHeartbeat(pageUrl);

      if (onHeartbeat) {
        onHeartbeat(response.active_visitors);
      }
    } catch (error) {
      if (onError) onError(error as Error);
    }
  }, [enabled, pathname, trackPageUrl, onHeartbeat, onError]);

  // Gestion visibilité page
  useEffect(() => {
    const handleVisibilityChange = () => {
      isPageVisibleRef.current = !document.hidden;
      if (!document.hidden) {
        sendHeartbeat(); // Heartbeat immédiat si page redevient visible
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [sendHeartbeat]);

  // Configuration intervalle
  useEffect(() => {
    if (!enabled) return;

    // Heartbeat immédiat au montage
    sendHeartbeat();

    // Intervalle
    intervalRef.current = setInterval(sendHeartbeat, interval);

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [enabled, interval, sendHeartbeat]);

  return { sendHeartbeat };
}
```

### 2. Utilisation dans un composant

**Exemple:** `app/layout.tsx` (layout racine)

```typescript
'use client';

import { useVisitorHeartbeat } from '@/hooks/useVisitorHeartbeat';

export default function RootLayout({ children }) {
  // Heartbeat automatique sur toutes les pages
  useVisitorHeartbeat({
    interval: 30000, // 30 secondes
    onHeartbeat: (activeVisitors) => {
      console.log('Visiteurs actifs:', activeVisitors);
    },
  });

  return (
    <html>
      <body>{children}</body>
    </html>
  );
}
```

**Exemple:** Page avec WebSocket

```typescript
'use client';

import { useVisitorHeartbeat } from '@/hooks/useVisitorHeartbeat';
import { useAnalyticsWebSocket } from '@/hooks/useAnalyticsWebSocket';
import { useState } from 'react';

export default function AnalyticsDashboard() {
  const [activeVisitors, setActiveVisitors] = useState(0);

  // Heartbeat HTTP (fallback)
  useVisitorHeartbeat({
    interval: 30000,
    onHeartbeat: (count) => setActiveVisitors(count || 0),
  });

  // WebSocket temps réel
  const { data, isConnected } = useAnalyticsWebSocket({
    onMessage: (msg) => {
      if (msg.type === 'heartbeat' || msg.type === 'initial_stats') {
        setActiveVisitors(msg.data.current_visitors || 0);
      }
    },
  });

  return (
    <div>
      <h1>Dashboard Analytics</h1>
      <p>Visiteurs actifs: {activeVisitors}</p>
      <p>WebSocket: {isConnected ? '🟢 Connecté' : '🔴 Déconnecté'}</p>
    </div>
  );
}
```

### 3. API Client

**Fichier:** `frontend/lib/api.ts`

```typescript
export const visitorsApi = {
  checkStatus: () =>
    api.get<VisitorStatus>('/api/v1/visitors/check'),

  sendHeartbeat: (pageUrl?: string, eventData?: Record<string, any>) =>
    api.post<HeartbeatResponse>('/api/v1/visitors/heartbeat', {
      page_url: pageUrl,
      event_data: eventData,
    }),
};
```

---

## Configuration

### Variables d'environnement

**Backend** (`.env`):

```bash
# Redis (requis pour le tracking)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# PostgreSQL (requis pour la persistance)
DATABASE_URL=postgres://user:pass@localhost:5432/maicivy
```

**Frontend** (`.env.local`):

```bash
# URL de l'API backend
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Paramètres ajustables

| Paramètre | Valeur par défaut | Description | Fichier |
|-----------|-------------------|-------------|---------|
| Intervalle heartbeat client | 30 secondes | Fréquence d'envoi heartbeat HTTP | `useVisitorHeartbeat.ts` |
| Timeout inactivité | 5 minutes | Durée avant marquage inactif | `analytics.go` |
| Intervalle cleanup job | 1 minute | Fréquence du job de nettoyage | `main.go` |
| Heartbeat WebSocket serveur | 5 secondes | Fréquence des stats temps réel | `analytics.go` |
| TTL Redis | 10 minutes | Durée de vie de la clé Redis | `analytics.go` |

---

## Monitoring

### Métriques Prometheus

Le système expose les métriques Prometheus suivantes :

```go
// Nombre de visiteurs actuellement actifs
maicivy_current_visitors{} 42

// Nombre total d'événements heartbeat reçus
maicivy_events_total{type="heartbeat"} 15234

// Nombre de nettoyages effectués
maicivy_visitor_cleanups_total{} 234

// Visiteurs inactifs supprimés
maicivy_inactive_visitors_removed_total{} 45
```

### Logs

**Logs importants** (Zerolog JSON):

```json
{
  "level": "info",
  "msg": "Visitor marked active via WebSocket heartbeat",
  "visitor_id": "550e8400-e29b-41d4-a716-446655440000"
}

{
  "level": "info",
  "msg": "Cleaned up inactive visitors",
  "removed": 3,
  "events_published": 3
}

{
  "level": "debug",
  "msg": "Heartbeat received",
  "visitor_id": "550e8400-e29b-41d4-a716-446655440000",
  "active_visitors": 42
}
```

### Dashboard Grafana

Requêtes PromQL utiles :

```promql
# Visiteurs actifs actuels
maicivy_current_visitors

# Taux de heartbeats par seconde
rate(maicivy_events_total{type="heartbeat"}[1m])

# Visiteurs supprimés (nettoyage)
rate(maicivy_inactive_visitors_removed_total[5m])

# Uptime du job de nettoyage
up{job="maicivy_visitor_cleanup"}
```

---

## Best Practices

### ✅ DO

1. **Utiliser le hook dans le layout racine** : `app/layout.tsx` pour couvrir toutes les pages
2. **Gérer la visibilité de la page** : Arrêter les heartbeats si l'utilisateur change d'onglet
3. **Configurer un intervalle raisonnable** : 30 secondes = bon équilibre performance/précision
4. **Monitorer les métriques** : Surveiller le nombre de visiteurs actifs via Prometheus
5. **Tester la déconnexion** : Vérifier que les visiteurs sont bien supprimés après 5 minutes
6. **Logger les erreurs** : Utiliser `onError` pour tracker les problèmes de heartbeat

### ❌ DON'T

1. **Ne pas envoyer des heartbeats trop fréquents** : < 10 secondes = surcharge serveur
2. **Ne pas ignorer les erreurs** : Un taux d'erreur élevé indique un problème
3. **Ne pas bloquer l'UI** : Les heartbeats doivent être asynchrones et non bloquants
4. **Ne pas oublier le cleanup** : Toujours nettoyer les timers au démontage
5. **Ne pas tracker sans consentement** : Respecter RGPD (cookies = session anonyme)

---

## Troubleshooting

### Problème : Les visiteurs ne sont pas comptés

**Symptômes** : `current_visitors` reste à 0

**Causes possibles** :

1. ✅ **Redis non connecté** : Vérifier `REDIS_HOST` et `REDIS_PORT`
   ```bash
   redis-cli ping
   # Doit retourner: PONG
   ```

2. ✅ **Middleware tracking désactivé** : Vérifier que le middleware est bien enregistré
   ```go
   // main.go
   app.Use(trackingMW.Handler())
   ```

3. ✅ **visitor_id non défini** : Vérifier les logs backend
   ```
   WARN Heartbeat called without visitor_id in context
   ```

**Solution** :
```bash
# Vérifier Redis
docker logs maicivy-redis-1

# Vérifier le backend
docker logs maicivy-backend-1 | grep visitor
```

---

### Problème : Les visiteurs ne sont jamais supprimés

**Symptômes** : `current_visitors` augmente indéfiniment

**Causes possibles** :

1. ✅ **Job de nettoyage non démarré** : Vérifier les logs au démarrage
   ```
   INFO Visitor cleanup job started
   ```

2. ✅ **Timeout trop long** : Vérifier le paramètre dans `MarkVisitorActive`
   ```go
   fiveMinutesAgo := now - 5*60 // 5 minutes
   ```

**Solution** :
```bash
# Vérifier les logs du job
docker logs maicivy-backend-1 | grep "Cleaned up inactive"

# Test manuel via Redis CLI
redis-cli
> ZRANGE analytics:realtime:visitors 0 -1 WITHSCORES
```

---

### Problème : Heartbeats échouent côté client

**Symptômes** : Erreurs console `Failed to send heartbeat`

**Causes possibles** :

1. ✅ **CORS non configuré** : Vérifier `ALLOWED_ORIGINS` backend
2. ✅ **API_URL incorrecte** : Vérifier `NEXT_PUBLIC_API_URL`
3. ✅ **Session expirée** : Cookie `maicivy_session` supprimé

**Solution** :
```javascript
// Debug dans la console
await fetch('http://localhost:8080/api/v1/visitors/heartbeat', {
  method: 'POST',
  credentials: 'include',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ page_url: '/cv' })
});
```

---

### Problème : WebSocket se déconnecte fréquemment

**Symptômes** : Reconnexions constantes

**Causes possibles** :

1. ✅ **Reverse proxy (Nginx) timeout** : Augmenter `proxy_read_timeout`
   ```nginx
   location /ws/ {
       proxy_read_timeout 3600s;
       proxy_send_timeout 3600s;
   }
   ```

2. ✅ **Heartbeat WebSocket manquant** : Le client doit envoyer des pings

**Solution** :
```typescript
// Envoyer un ping toutes les 30 secondes
setInterval(() => {
  ws.send(JSON.stringify({ type: 'ping', visitor_id: '...' }));
}, 30000);
```

---

## Références

### Documentation externe

- [WebSocket Best Practices (Medium)](https://kxming.medium.com/why-does-websocket-require-frontend-heartbeat-detection-and-is-there-a-native-detection-mechanism-42f30a6b466b)
- [Active User Detection (Medium)](https://medium.com/@onakoyak/real-time-reliability-using-client-server-heartbeats-to-ensure-consistent-online-status-in-a-chat-429ae3c2d94a)
- [Redis Sorted Sets](https://redis.io/docs/data-types/sorted-sets/)
- [React useEffect Hook](https://react.dev/reference/react/useEffect)

### Fichiers du projet

- Backend API: `backend/internal/api/visitor.go`
- Service Analytics: `backend/internal/services/analytics.go`
- WebSocket: `backend/internal/websocket/analytics.go`
- Hook React: `frontend/hooks/useVisitorHeartbeat.ts`
- Job Cleanup: `backend/internal/jobs/visitor_cleanup.go`

---

**Dernière mise à jour :** 2026-01-16
**Contributeurs :** Claude Sonnet 4.5
