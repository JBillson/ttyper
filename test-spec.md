# Real-Time Dashboard Websocket Architecture

## Overview

A websocket-based real-time dashboard system that pushes live updates to connected clients without polling. Designed for operational dashboards showing order status, warehouse activity, and delivery tracking across multiple tenants.

## Problem

Current dashboards poll the API every 30 seconds, creating unnecessary load and stale data. During peak hours (200+ concurrent users), the polling traffic accounts for 40% of total API requests while delivering a poor user experience - users see data that's up to 30 seconds old.

## Goals

- Sub-second update delivery for critical state changes (order status, stock movements)
- Reduce API polling load by 90%
- Support 500 concurrent websocket connections per pod
- Tenant-isolated channels - no data leakage between organizations
- Graceful degradation to polling if websocket connection fails

## Non-Goals

- Replacing REST APIs for CRUD operations
- Chat or messaging features
- Offline support for websocket data

---

## Architecture

### Connection Flow

```
Client                    API Gateway              WS Server              Redis Pub/Sub
  |                          |                        |                        |
  |-- WS upgrade request --> |                        |                        |
  |                          |-- Auth + tenant ID --> |                        |
  |                          |                        |-- SUBSCRIBE channel -->|
  |                          |<-- Connection ACK -----|                        |
  |<-- Connected ----------- |                        |                        |
  |                          |                        |                        |
  |                          |   (state change)       |                        |
  |                          |                        |<-- PUBLISH event ------|
  |<-- Push update --------- |<-- Forward ----------- |                        |
```

### Channel Structure

Each tenant gets isolated channels:

| Channel | Pattern | Example |
|---------|---------|---------|
| Orders | `{tenantId}:orders` | `acme-corp:orders` |
| Warehouse | `{tenantId}:warehouse:{warehouseId}` | `acme-corp:warehouse:wh-001` |
| Deliveries | `{tenantId}:deliveries` | `acme-corp:deliveries` |
| System | `{tenantId}:system` | `acme-corp:system` |

### Event Schema

```json
{
  "channel": "acme-corp:orders",
  "event": "order.status_changed",
  "payload": {
    "orderId": "ORD-12345",
    "previousStatus": "processing",
    "newStatus": "shipped",
    "updatedBy": "warehouse-worker-7",
    "timestamp": "2026-03-21T14:30:00Z"
  },
  "sequence": 847291
}
```

### Scaling Strategy

- **Horizontal scaling**: Multiple WS server pods behind a load balancer with sticky sessions
- **Redis Pub/Sub**: All pods subscribe to the same Redis channels - a publish on any pod reaches all connected clients
- **Connection limits**: Max 500 connections per pod, new connections routed to least-loaded pod
- **Heartbeat**: Ping/pong every 30s, disconnect after 3 missed pongs

## Security

- Websocket upgrade requires valid JWT in the initial HTTP request
- Tenant ID extracted from JWT claims - clients cannot subscribe to other tenants' channels
- Rate limiting: max 10 subscribe/unsubscribe operations per second per connection
- Message size limit: 64KB per frame

## Fallback

If the websocket connection drops:
1. Client attempts reconnect with exponential backoff (1s, 2s, 4s, 8s, max 30s)
2. On reconnect, client sends last received `sequence` number
3. Server replays missed events from a 5-minute Redis stream buffer
4. If gap > 5 minutes, client falls back to REST polling until caught up
5. UI shows a "Live" / "Reconnecting..." / "Polling" indicator

---

## Implementation Phases

| Phase | Scope | Timeline |
|-------|-------|----------|
| 1 | Order status events + basic dashboard | 2 weeks |
| 2 | Warehouse activity events | 1 week |
| 3 | Delivery tracking events | 1 week |
| 4 | Historical replay + gap recovery | 1 week |
