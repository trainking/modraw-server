# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build ./cmd/server        # Build the server binary
go run ./cmd/server          # Run the server
go vet ./...                 # Vet all packages
```

No tests exist yet — add tests with `go test ./...` when ready.

## Architecture

**modraw-server** is a Go 1.17 REST + WebSocket backend for [modraw](https://github.com/trainking/modraw), a collaborative hand-drawn style whiteboard app. Uses **Gin** for HTTP routing, **gorilla/websocket** for real-time collaboration, **PostgreSQL** for persistence, and **JWT** for auth (HS256, `golang-jwt/jwt`).

### Layered structure (top → bottom)

```
cmd/server/main.go          — Entrypoint: wires everything, starts HTTP server
    internal/handler/       — HTTP handlers (Gin), thin: parse input → call service → respond
    internal/service/       — Business logic, authorization checks
    internal/repository/    — Database queries (raw SQL, no ORM)
    internal/model/         — Data structs (User, Canvas, Folder, Library, ShareLink, CanvasCollaborator)
    internal/crdt/          — CRDT conflict resolution for collaborative canvas editing
    internal/database/      — Postgres connection + migration runner
    internal/middleware/     — AuthRequired (JWT), CORS, Recovery
    internal/ws/            — WebSocket hub/room/client for real-time canvas collaboration
    internal/config/        — Env-based configuration
    pkg/jwt/                — JWT token generation & validation (access, refresh, share tokens)
    pkg/httputil/           — JSON response helpers (Success, Error, Paginated, etc.)
```

### Wire-up pattern (explicit DI, no framework)

`setupRouter()` in `cmd/server/main.go` manually creates repos → services → handlers and registers routes. There is no DI container — follow this same pattern when adding new resources.

### API structure

All routes are under `/api/v1`. Standard Gin groups with `AuthRequired` middleware:

**Auth** (public):
- `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`
- `DELETE /auth/logout` (auth required)

**Users** (auth required):
- `GET/PUT /users/me`, `PUT /users/me/password`

**Folders** (auth required):
- `GET/POST /folders`, `GET/PUT/DELETE /folders/:id`, `PUT /folders/:id/move`
- Supports `?tree=true` for tree-structured listing

**Canvases** (auth required):
- `GET/POST /canvases`, `GET/PUT/DELETE /canvases/:id`
- `PUT /canvases/:id/data` (full data save), `PUT /canvases/:id/move`
- `GET /canvases/:id/collaborators`, `POST /canvases/:id/collaborators`, `PUT/DELETE /canvases/:id/collaborators/:user_id`
- `GET /canvases/:id/shares`, `POST /canvases/:id/shares`, `DELETE /canvases/:id/shares/:share_id`

**Libraries** (auth required):
- `GET/POST /libraries`, `GET/PUT/DELETE /libraries/:id`

**Share links** (public):
- `GET /shares/:code` (get share info), `POST /shares/:code/validate` (validate password, returns share_token)

**WebSocket**:
- `GET /ws` (token passed as `?token=` query param)

### WebSocket collaboration model

The hub/room/client pattern in `internal/ws/`:

- **Hub** owns rooms (one per canvas ID). Rooms are created lazily when the first client joins and destroyed when the last client leaves.
- **Room** holds connected clients + an in-memory CRDT `CanvasState`. It broadcasts join/leave/op/cursor/awareness events.
- **Client** runs two goroutines: `ReadPump` (reads WS messages, routes by message type) and `WritePump` (writes from send channel).
- Clients join a canvas room by sending a `join` message with `canvas_id` and optional `share_token`. Read-only clients cannot send ops or saves.

### CRDT conflict resolution

`internal/crdt/` implements a simple state-based CRDT for collaborative canvas editing:

- Each canvas element has a unique client-generated ID and a monotonic `_v` version number.
- Operations: `element_add`, `element_update`, `element_delete`, `elements_reorder`, `canvas_update`.
- **LWW (Last-Writer-Wins)**: updates with higher version numbers overwrite. Stale operations (version <= current) are acked but not applied.
- JSON properties are shallow-merged per key — top-level keys in props overwrite, not the entire element.
- CRDT state is loaded from `canvases.data` JSONB when a room is created, kept in memory during collaboration, and persisted to DB on `save` WS message.

### WS access control (3-tier)

The `wsAccessChecker` in `main.go` checks permissions in order:
1. **Share token** — JWT-encoded share token from share link validation (grants `readonly` or `collaborate`)
2. **Canvas owner** — always gets `collaborate` permission
3. **Collaborators table** — `canvas_collaborators` row determines permission

### Database

- PostgreSQL with `uuid-ossp` extension for UUID primary keys
- Migrations in `migrations/` — `*.up.sql` files are run sequentially at startup by `database.RunMigrations()`
- Key tables: `users`, `folders` (self-referencing tree), `canvases` (JSONB data column), `libraries`, `share_links`, `canvas_collaborators`, `refresh_tokens`
- Connection pool: 25 open / 5 idle, 5min lifetime, 1min idle timeout

### Auth

- Access tokens (HS256, default 15min TTL) + refresh tokens (default 7 day TTL) with refresh token rotation
- Share tokens (HS256, 24h TTL) for share link access — encode `canvas_id` + `permission`
- Password hashing via bcrypt; share link passwords also bcrypt-hashed
- `AuthRequired` middleware extracts `user_id`, `email`, `nickname` into Gin context; handlers access via `middleware.GetUserID(c)`

### Config

All config via environment variables (see `.env.example`). `internal/config/config.go` maps env vars to a `Config` struct. Copy `.env.example` to `.env` for local development.
