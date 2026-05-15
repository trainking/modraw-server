# modraw-server

Backend server for [Modraw](https://github.com/trainking/modraw) — a hand-drawn style collaborative whiteboard application. Provides REST API for resource management and WebSocket-based real-time collaborative editing with CRDT conflict resolution.

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.17 |
| HTTP Framework | Gin |
| Database | PostgreSQL |
| Auth | JWT (HS256) + bcrypt |
| Real-time | WebSocket (gorilla/websocket) |
| Conflict Resolution | State-based CRDT (LWW) |

## Quick Start

### Prerequisites

- Go 1.17+
- PostgreSQL 12+ (with `uuid-ossp` extension)

### Setup

```bash
# Clone
git clone https://github.com/trainking/modraw-server.git
cd modraw-server

# Configure environment
cp .env.example .env
# Edit .env — set DATABASE_URL and JWT_SECRET

# Install dependencies
go mod download

# Build & run
go run ./cmd/server
```

The server starts on `:8080` by default. Migrations run automatically on startup.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `:8080` | Server listen address |
| `DATABASE_URL` | `postgres://modraw:modraw@localhost:5432/modraw?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | `change-me-in-production` | HMAC-SHA256 signing key |
| `ACCESS_TOKEN_TTL` | `15m` | Access token lifetime |
| `REFRESH_TOKEN_TTL` | `168h` | Refresh token lifetime |
| `CORS_ORIGINS` | `*` | Comma-separated allowed origins |
| `WS_MAX_MSG_SIZE` | `4096` | WebSocket max message size (bytes) |
| `GIN_MODE` | `debug` | Gin run mode (`debug`/`release`/`test`) |

## Project Structure

```
cmd/server/main.go           Entrypoint & dependency wiring
internal/
  config/                    Environment-based configuration
  database/                  PostgreSQL connection pool & migration runner
  model/                     Data structs (User, Canvas, Folder, etc.)
  repository/                Raw SQL database queries
  service/                   Business logic & authorization
  handler/                   HTTP request handlers (Gin)
  middleware/                JWT auth, CORS, panic recovery
  ws/                        WebSocket hub/room/client for real-time collaboration
  crdt/                      CRDT conflict resolution for canvas operations
pkg/
  jwt/                       JWT token generation & validation
  httputil/                  JSON response helpers
migrations/                  SQL migration files (auto-run at startup)
```

### Architecture Pattern

Explicit dependency injection — `main.go` creates repos → services → handlers → routes. No DI framework.

```
Request → Middleware (CORS, Auth) → Handler → Service → Repository → PostgreSQL
                                              ↘ CRDT (in-memory, per canvas room)
```

## API Overview

See [API.md](API.md) for full endpoint documentation.

### REST Endpoints

| Group | Method | Path | Auth |
|---|---|---|---|
| Auth | POST | `/api/v1/auth/register` | No |
| Auth | POST | `/api/v1/auth/login` | No |
| Auth | POST | `/api/v1/auth/refresh` | No |
| Auth | DELETE | `/api/v1/auth/logout` | Yes |
| Users | GET | `/api/v1/users/me` | Yes |
| Users | PUT | `/api/v1/users/me` | Yes |
| Users | PUT | `/api/v1/users/me/password` | Yes |
| Folders | GET/POST | `/api/v1/folders` | Yes |
| Folders | GET/PUT/DELETE | `/api/v1/folders/:id` | Yes |
| Folders | PUT | `/api/v1/folders/:id/move` | Yes |
| Canvases | GET/POST | `/api/v1/canvases` | Yes |
| Canvases | GET/PUT/DELETE | `/api/v1/canvases/:id` | Yes |
| Canvases | PUT | `/api/v1/canvases/:id/data` | Yes |
| Canvases | PUT | `/api/v1/canvases/:id/move` | Yes |
| Collaborators | GET/POST | `/api/v1/canvases/:id/collaborators` | Yes |
| Collaborators | PUT/DELETE | `/api/v1/canvases/:id/collaborators/:user_id` | Yes |
| Share Links | GET/POST | `/api/v1/canvases/:id/shares` | Yes |
| Share Links | DELETE | `/api/v1/canvases/:id/shares/:share_id` | Yes |
| Libraries | GET/POST | `/api/v1/libraries` | Yes |
| Libraries | GET/PUT/DELETE | `/api/v1/libraries/:id` | Yes |
| Shares | GET | `/api/v1/shares/:code` | No |
| Shares | POST | `/api/v1/shares/:code/validate` | No |
| WS | GET | `/ws` | Query token |

### WebSocket Protocol

Connect to `/ws?token=<access_token>`. The server uses a message-based protocol — all frames are JSON with `{ "type": "...", "payload": {...} }`.

**Client → Server messages:**

| Type | Payload | Description |
|---|---|---|
| `join` | `{ canvas_id, share_token? }` | Join a canvas room |
| `leave` | `{ canvas_id }` | Leave current room |
| `op` | `{ canvas_id, seq, operation }` | Send a CRDT operation |
| `cursor` | `{ canvas_id, position }` | Broadcast cursor position |
| `awareness` | `{ canvas_id, state }` | Broadcast presence state |
| `save` | `{ canvas_id, data }` | Persist current canvas state |
| `ping` | — | Heartbeat |

**Server → Client messages:**

| Type | Payload | Description |
|---|---|---|
| `joined` | `{ canvas_id, clients[] }` | Successfully joined room |
| `left` | `{ canvas_id, user_id }` | A user left the room |
| `op` | `{ canvas_id, user_id, seq, operation }` | CRDT operation relayed |
| `cursor` | `{ canvas_id, user_id, position }` | Cursor position relayed |
| `awareness` | `{ canvas_id, user_id, state }` | Presence state relayed |
| `ack` | `{ canvas_id, seq }` | Operation acknowledged |
| `saved` | `{ canvas_id, updated_at }` | State persisted |
| `pong` | — | Heartbeat response |
| `error` | `{ code, message }` | Error notification |

### CRDT Operations

Operations sent via `op` messages use a state-based CRDT with LWW (Last-Writer-Wins) semantics:

```json
{ "op": "element_add",    "elem_id": "uuid", "version": 1, "props": {...} }
{ "op": "element_update", "elem_id": "uuid", "version": 2, "props": {...} }
{ "op": "element_delete", "elem_id": "uuid", "version": 3 }
{ "op": "elements_reorder", "elem_ids": ["uuid1", "uuid2", ...], "version": 1 }
{ "op": "canvas_update", "version": 1, "props": { "background": "grid" } }
```

Each element carries a monotonic `version`. Higher versions overwrite; stale versions (≤ current) are acked but not applied. Property merging is shallow per-key.

### Response Format

All API responses follow a consistent envelope:

```json
// Success
{ "ok": true, "data": {...} }

// Created (201)
{ "ok": true, "data": {...} }

// Paginated list
{ "ok": true, "data": [...], "page": 1, "limit": 20, "total": 42 }

// Error
{ "ok": false, "error": "ERROR_CODE", "message": "Human-readable description" }
```

### Error Codes

| Code | HTTP Status | Description |
|---|---|---|
| `VALIDATION` | 400 | Invalid request body |
| `WEAK_PASSWORD` | 400 | Password must be 8-72 characters |
| `MISSING_TOKEN` | 401 | Authorization header missing |
| `INVALID_TOKEN` | 401 | Token invalid or expired |
| `INVALID_CREDENTIALS` | 401 | Wrong email or password |
| `TOKEN_REVOKED` | 401 | Refresh token has been revoked |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `EMAIL_TAKEN` | 409 | Email already registered |
| `EXPIRED` | 410 | Share link expired |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

## Database

PostgreSQL with UUID primary keys via `uuid-ossp` extension. Tables:

- **users** — accounts with bcrypt-hashed passwords
- **folders** — self-referencing tree structure (parent_id)
- **canvases** — JSONB data column stores the full scene state
- **libraries** — reusable asset libraries (JSONB)
- **share_links** — shareable canvas access links with optional password & expiration
- **canvas_collaborators** — per-canvas user permissions (readonly/collaborate)
- **refresh_tokens** — token rotation with revocation support

Migrations in `migrations/` run sequentially by filename on server startup.

## License

MIT
