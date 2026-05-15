# API Documentation

Base URL: `http://localhost:8080/api/v1`

All responses use a JSON envelope:

```json
// Success
{ "ok": true, "data": {...} }

// Created
{ "ok": true, "data": {...} }

// Paginated
{ "ok": true, "data": [...], "page": 1, "limit": 20, "total": 42 }

// Error
{ "ok": false, "error": "CODE", "message": "..." }
```

---

## Authentication

### POST /auth/register

Register a new user account.

**Request**
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "nickname": "Alice"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `email` | string | Yes | Must be valid email |
| `password` | string | Yes | 8-72 characters |
| `nickname` | string | No | Display name |

**Response** `201 Created`
```json
{
  "ok": true,
  "data": {
    "access_token": "eyJhbG...",
    "refresh_token": "eyJhbG...",
    "user": {
      "id": "550e8400-...",
      "email": "user@example.com",
      "nickname": "Alice",
      "avatar_url": "",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  }
}
```

Errors: `VALIDATION` (400), `WEAK_PASSWORD` (400), `EMAIL_TAKEN` (409)

---

### POST /auth/login

**Request**
```json
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response** `200 OK` — Same token + user shape as register.

Errors: `VALIDATION` (400), `INVALID_CREDENTIALS` (401)

---

### POST /auth/refresh

Rotate refresh token — revokes the old one and issues a new pair.

**Request**
```json
{
  "refresh_token": "eyJhbG..."
}
```

**Response** `200 OK` — Same token + user shape as register.

Errors: `VALIDATION` (400), `INVALID_TOKEN` (401), `TOKEN_REVOKED` (401)

---

### DELETE /auth/logout

Revoke a refresh token. Requires `Authorization: Bearer <access_token>`.

**Request**
```json
{
  "refresh_token": "eyJhbG..."
}
```

**Response** `200 OK`
```json
{
  "ok": true,
  "data": { "message": "logged out" }
}
```

Errors: `VALIDATION` (400), `MISSING_TOKEN` (401), `INVALID_TOKEN` (401)

---

## Users

All endpoints require `Authorization: Bearer <access_token>`.

### GET /users/me

Get current user profile.

**Response** `200 OK`
```json
{
  "ok": true,
  "data": {
    "id": "550e8400-...",
    "email": "user@example.com",
    "nickname": "Alice",
    "avatar_url": "https://...",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### PUT /users/me

Update nickname and/or avatar.

**Request**
```json
{
  "nickname": "Alice Updated",
  "avatar_url": "https://example.com/avatar.png"
}
```

| Field | Type | Required |
|---|---|---|
| `nickname` | string | No |
| `avatar_url` | string | No |

**Response** `200 OK` — Full user object.

---

### PUT /users/me/password

Change password (requires current password).

**Request**
```json
{
  "old_password": "currentpassword",
  "new_password": "newsecurepassword"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `old_password` | string | Yes | Current password for verification |
| `new_password` | string | Yes | 8-72 characters |

**Response** `200 OK`
```json
{
  "ok": true,
  "data": { "message": "password changed" }
}
```

Errors: `VALIDATION` (400), `WEAK_PASSWORD` (400), `WRONG_PASSWORD` (400), `NOT_FOUND` (404)

---

## Folders

All endpoints require `Authorization: Bearer <access_token>`.

Folders support a self-referencing tree structure. Each folder belongs to the authenticated user.

### GET /folders

List folders. Supports two modes:

**Flat list:**
```
GET /api/v1/folders
```
Response: `{ "ok": true, "data": [Folder, ...] }`

**Tree:**
```
GET /api/v1/folders?tree=true
```
Response: `{ "ok": true, "data": [FolderTreeNode, ...] }`

**Folder object:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "name": "My Folder",
  "parent_id": "uuid or null",
  "sort_order": 0,
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

**FolderTreeNode** adds `"children": [FolderTreeNode, ...]` for nested display.

---

### POST /folders

Create a folder.

**Request**
```json
{
  "name": "New Folder",
  "parent_id": "uuid or null",
  "sort_order": 0
}
```

| Field | Type | Required |
|---|---|---|
| `name` | string | Yes |
| `parent_id` | string | No |
| `sort_order` | int | No |

**Response** `201 Created` — Full folder object.

---

### GET /folders/:id

Get a single folder. Only accessible by the folder owner.

**Response** `200 OK` — Folder object.

Errors: `NOT_FOUND` (404)

---

### PUT /folders/:id

Update folder name or sort order.

**Request**
```json
{
  "name": "Renamed Folder",
  "sort_order": 5
}
```

| Field | Type | Required |
|---|---|---|
| `name` | string | No |
| `sort_order` | int | No |

**Response** `200 OK` — Updated folder object.

---

### DELETE /folders/:id

Delete a folder. Children folders are NOT cascaded — they remain but with `parent_id` still pointing to the deleted folder. Only the folder owner can delete.

**Response** `204 No Content`

---

### PUT /folders/:id/move

Move a folder to a new parent. Validates against circular references.

**Request**
```json
{
  "parent_id": "target-folder-uuid"
}
```

**Response** `200 OK`
```json
{
  "ok": true,
  "data": { "message": "moved" }
}
```

Errors: `CIRCULAR_REFERENCE` (400)

---

## Canvases

All endpoints require `Authorization: Bearer <access_token>`.

Each canvas stores the full whiteboard scene as a JSONB `data` field. Content is opaque to the server — it matches the Modraw `.mdr` file format.

### GET /canvases

List canvases owned by the current user. Supports pagination and filtering.

```
GET /api/v1/canvases?page=1&limit=20&folder_id=uuid&search=keyword
```

| Query | Type | Default | Notes |
|---|---|---|---|
| `page` | int | 1 | |
| `limit` | int | 20 | Max 100 |
| `folder_id` | string | — | Filter by folder |
| `search` | string | — | ILIKE search on name |

**Response** `200 OK`
```json
{
  "ok": true,
  "data": [
    {
      "id": "uuid",
      "owner_id": "uuid",
      "folder_id": "uuid or null",
      "name": "Untitled",
      "thumbnail": "",
      "file_size": 0,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 42
}
```

Note: list items omit the `data` field for performance. Use GET `/:id` to fetch the full canvas including data.

---

### POST /canvases

Create a new canvas.

**Request**
```json
{
  "name": "My Canvas",
  "folder_id": "uuid or null",
  "data": { "elements": [], "props": {} }
}
```

| Field | Type | Required | Default |
|---|---|---|---|
| `name` | string | No | `"Untitled"` |
| `folder_id` | string | No | `null` |
| `data` | object | No | `{}` |

**Response** `201 Created` — Full canvas object including `data`.

---

### GET /canvases/:id

Get a single canvas with full data. Owner or collaborator access required.

**Response** `200 OK`
```json
{
  "ok": true,
  "data": {
    "id": "uuid",
    "owner_id": "uuid",
    "folder_id": "uuid or null",
    "name": "My Canvas",
    "data": { "elements": [...], "props": {...} },
    "thumbnail": "",
    "file_size": 1024,
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

### PUT /canvases/:id

Update canvas name or move to a different folder.

**Request**
```json
{
  "name": "Renamed Canvas",
  "folder_id": "uuid"
}
```

**Response** `200 OK` — Updated canvas object (includes `data`).

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

### PUT /canvases/:id/data

Save canvas scene data (full replacement).

**Request**
```json
{
  "data": { "elements": [...], "props": {...} },
  "file_size": 2048
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `data` | object | Yes | Full canvas scene JSON |
| `file_size` | int | No | Auto-calculated if omitted |

**Response** `200 OK`
```json
{ "ok": true, "data": { "message": "saved" } }
```

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

### PUT /canvases/:id/move

Move canvas to a folder.

**Request**
```json
{
  "folder_id": "target-folder-uuid"
}
```

**Response** `200 OK`
```json
{ "ok": true, "data": { "message": "moved" } }
```

---

### DELETE /canvases/:id

Delete a canvas. Also removes associated share links and collaborators.

**Response** `204 No Content`

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

## Collaborators

All endpoints require `Authorization: Bearer <access_token>`. Only the canvas owner can manage collaborators.

### GET /canvases/:id/collaborators

List collaborators for a canvas. Accessible by owner and collaborators.

**Response** `200 OK`
```json
{
  "ok": true,
  "data": [
    {
      "id": "uuid",
      "canvas_id": "uuid",
      "user_id": "uuid",
      "permission": "collaborate",
      "added_at": "2025-01-01T00:00:00Z",
      "email": "bob@example.com",
      "nickname": "Bob"
    }
  ]
}
```

---

### POST /canvases/:id/collaborators

Add a collaborator by email. Uses upsert — updating an existing collaborator changes their permission.

**Request**
```json
{
  "email": "bob@example.com",
  "permission": "collaborate"
}
```

| Field | Type | Required | Values |
|---|---|---|---|
| `email` | string | Yes | Registered user email |
| `permission` | string | Yes | `"readonly"` or `"collaborate"` |

**Response** `201 Created` — Collaborator object (includes email and nickname from JOIN).

Errors: `VALIDATION` (400), `FORBIDDEN` (403 — not owner), `INVALID` (400 — user not found)

---

### PUT /canvases/:id/collaborators/:user_id

Update a collaborator's permission.

**Request**
```json
{
  "permission": "readonly"
}
```

**Response** `200 OK`
```json
{ "ok": true, "data": { "message": "updated" } }
```

Errors: `FORBIDDEN` (403), `NOT_FOUND` (404)

---

### DELETE /canvases/:id/collaborators/:user_id

Remove a collaborator.

**Response** `204 No Content`

Errors: `FORBIDDEN` (403), `NOT_FOUND` (404)

---

## Share Links

### GET /canvases/:id/shares

List all share links for a canvas. Requires owner access.

**Response** `200 OK`
```json
{
  "ok": true,
  "data": [
    {
      "code": "a1b2c3d4",
      "permission": "readonly",
      "has_password": true,
      "expires_at": "2025-12-31T23:59:59Z",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

### POST /canvases/:id/shares

Create a share link. Generates a random 8-character hex code.

**Request**
```json
{
  "permission": "readonly",
  "password": "optional-password",
  "expires_at": "2025-12-31T23:59:59Z"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `permission` | string | Yes | `"readonly"` or `"collaborate"` |
| `password` | string | No | bcrypt-hashed; if set, validation will require it |
| `expires_at` | string | No | RFC3339 format; if omitted, never expires |

**Response** `201 Created`
```json
{
  "ok": true,
  "data": {
    "code": "a1b2c3d4",
    "permission": "readonly",
    "has_password": true,
    "expires_at": "2025-12-31T23:59:59Z"
  }
}
```

---

### DELETE /canvases/:id/shares/:share_id

Revoke a share link.

**Response** `204 No Content`

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

### GET /shares/:code

Get public information about a share link. **No authentication required.**

**Response** `200 OK`
```json
{
  "ok": true,
  "data": {
    "code": "a1b2c3d4",
    "permission": "readonly",
    "has_password": true,
    "expires_at": "2025-12-31T23:59:59Z",
    "created_at": "2025-01-01T00:00:00Z",
    "canvas_name": "My Canvas",
    "thumbnail": "",
    "owner_name": "Alice"
  }
}
```

Errors: `NOT_FOUND` (404), `EXPIRED` (410)

---

### POST /shares/:code/validate

Validate a share link and obtain an access token. If the link has a password, it must be provided. **No authentication required.**

**Request**
```json
{
  "password": "optional-password"
}
```

**Response** `200 OK`
```json
{
  "ok": true,
  "data": {
    "canvas_id": "uuid",
    "canvas_name": "My Canvas",
    "thumbnail": "",
    "owner_name": "Alice",
    "permission": "readonly",
    "share_token": "eyJhbG..."
  }
}
```

The `share_token` is a signed JWT (24h TTL) encoding `canvas_id` and `permission`. Pass it as `share_token` in WebSocket `join` messages to access the canvas without being a registered collaborator.

Errors: `NOT_FOUND` (404), `EXPIRED` (410), `INVALID_PASSWORD` (403)

---

## Libraries

All endpoints require `Authorization: Bearer <access_token>`.

Libraries store reusable assets (`.mdrlib` files) as JSONB data. Each library belongs to a single user.

### GET /libraries

List the current user's libraries.

**Response** `200 OK`
```json
{
  "ok": true,
  "data": [
    {
      "id": "uuid",
      "owner_id": "uuid",
      "name": "My Shapes",
      "description": "Common shapes library",
      "data": { "items": [...] },
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /libraries

Create a library.

**Request**
```json
{
  "name": "My Shapes",
  "description": "Common shapes library",
  "data": { "items": [...] }
}
```

| Field | Type | Required | Default |
|---|---|---|---|
| `name` | string | Yes | — |
| `description` | string | No | `""` |
| `data` | object | No | `{}` |

**Response** `201 Created` — Full library object.

---

### GET /libraries/:id

Get a single library. Owner access required.

**Response** `200 OK` — Library object.

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

### PUT /libraries/:id

Update library metadata and/or data.

**Request**
```json
{
  "name": "Updated Shapes",
  "description": "Updated description",
  "data": { "items": [...] }
}
```

All fields optional. Partial updates supported — omitted fields keep their current values.

**Response** `200 OK` — Updated library object.

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

### DELETE /libraries/:id

Delete a library.

**Response** `204 No Content`

Errors: `NOT_FOUND` (404), `FORBIDDEN` (403)

---

## Health

### GET /health

Health check endpoint. No authentication required.

**Response** `200 OK`
```json
{ "status": "ok" }
```

---

## WebSocket

### GET /ws

Upgrade to WebSocket. Pass the access token as a query parameter:

```
ws://localhost:8080/ws?token=<access_token>
```

The server authenticates the token and opens a persistent connection. A single connection can join multiple canvas rooms sequentially (but only one at a time).

### Message Protocol

All messages are JSON text frames with the structure:

```json
{
  "type": "string",
  "payload": { ... }
}
```

### Client → Server

#### join

Join a canvas collaboration room.

```json
{
  "type": "join",
  "payload": {
    "canvas_id": "uuid",
    "share_token": "eyJhbG... (optional, from share link validation)"
  }
}
```

If the user lacks access (not owner, not collaborator, no valid share token), the server sends an `error` message with code `FORBIDDEN`.

#### leave

Leave the current room.

```json
{
  "type": "leave",
  "payload": { "canvas_id": "uuid" }
}
```

#### op

Send a CRDT operation. Only collaborators (not read-only) can send ops.

```json
{
  "type": "op",
  "payload": {
    "canvas_id": "uuid",
    "seq": 1,
    "operation": {
      "op": "element_add",
      "elem_id": "uuid",
      "version": 1,
      "props": { "type": "rectangle", "x": 100, "y": 200 }
    }
  }
}
```

See [CRDT Operations](#crdt-operations) for the full list of operation types.

#### cursor

Broadcast cursor position to other collaborators.

```json
{
  "type": "cursor",
  "payload": {
    "canvas_id": "uuid",
    "position": { "x": 150, "y": 300 }
  }
}
```

#### awareness

Broadcast presence/awareness state (e.g., which element the user is editing).

```json
{
  "type": "awareness",
  "payload": {
    "canvas_id": "uuid",
    "state": { "editing": "element-uuid" }
  }
}
```

#### save

Persist the current CRDT canvas state to the database.

```json
{
  "type": "save",
  "payload": {
    "canvas_id": "uuid",
    "data": { "elements": [...], "props": {...} }
  }
}
```

#### ping

Keep-alive heartbeat.

```json
{ "type": "ping" }
```

### Server → Client

#### joined

Sent after successfully joining a room. Includes the list of current clients.

```json
{
  "type": "joined",
  "payload": {
    "canvas_id": "uuid",
    "clients": [
      { "user_id": "uuid", "nickname": "Alice", "avatar_url": "..." },
      { "user_id": "uuid", "nickname": "Bob", "avatar_url": "..." }
    ]
  }
}
```

#### left

Broadcast when a user leaves the room.

```json
{
  "type": "left",
  "payload": { "canvas_id": "uuid", "user_id": "uuid" }
}
```

#### op

CRDT operation relayed from another collaborator. Read-only clients also receive these.

```json
{
  "type": "op",
  "payload": {
    "canvas_id": "uuid",
    "user_id": "uuid",
    "seq": 1,
    "operation": {
      "op": "element_add",
      "elem_id": "uuid",
      "version": 1,
      "props": { ... }
    }
  }
}
```

#### cursor

Cursor position from another user.

```json
{
  "type": "cursor",
  "payload": {
    "canvas_id": "uuid",
    "user_id": "uuid",
    "position": { "x": 150, "y": 300 }
  }
}
```

#### awareness

Presence state from another user.

```json
{
  "type": "awareness",
  "payload": {
    "canvas_id": "uuid",
    "user_id": "uuid",
    "state": { "editing": "element-uuid" }
  }
}
```

#### ack

Server acknowledgment of an `op` message. The `seq` matches the client's sent sequence number.

```json
{
  "type": "ack",
  "payload": { "canvas_id": "uuid", "seq": 1 }
}
```

#### saved

Confirmation that the canvas state was persisted. Broadcast to all room members.

```json
{
  "type": "saved",
  "payload": { "canvas_id": "uuid", "updated_at": "2025-01-01T00:00:00Z" }
}
```

#### pong

Response to `ping`.

```json
{ "type": "pong" }
```

#### error

Error notification for the current connection.

```json
{
  "type": "error",
  "payload": { "code": "FORBIDDEN", "message": "read-only access" }
}
```

---

## CRDT Operations

The server uses a **state-based CRDT** with **Last-Writer-Wins (LWW)** per-element conflict resolution. Each canvas element has a unique client-generated `elem_id` and a monotonic `version` counter. Operations are **commutative** — they produce the same result regardless of order.

### element_add

Add a new element to the canvas. Ignored if `elem_id` already exists.

```json
{
  "op": "element_add",
  "elem_id": "client-generated-uuid",
  "version": 1,
  "props": {
    "type": "rectangle",
    "x": 100,
    "y": 200,
    "width": 150,
    "height": 100,
    "angle": 0,
    "style": { "fill": "#ffffff", "stroke": "#000000" }
  }
}
```

### element_update

Update properties of an existing element. Only applies if `version > current_version`. Properties are shallow-merged — only the provided keys are updated.

```json
{
  "op": "element_update",
  "elem_id": "client-generated-uuid",
  "version": 2,
  "props": { "x": 120, "y": 220 }
}
```

### element_delete

Remove an element from the canvas.

```json
{
  "op": "element_delete",
  "elem_id": "client-generated-uuid",
  "version": 3
}
```

### elements_reorder

Set the z-order of elements. The `elem_ids` array defines the new order. Elements not listed are appended at the end in their original relative order.

```json
{
  "op": "elements_reorder",
  "elem_ids": ["uuid-3", "uuid-1", "uuid-2"],
  "version": 1
}
```

### canvas_update

Update canvas-level properties (background, grid settings, etc.).

```json
{
  "op": "canvas_update",
  "version": 1,
  "props": { "background": "grid" }
}
```

### Conflict Resolution Rules

1. Operations targeting the same element with **higher version** numbers always win
2. Operations with **version ≤ current** are acknowledged but **not applied** (stale/duplicate detection)
3. Property merging is **shallow per-key**: `element_update` with `{ "x": 120 }` only changes `x` — all other properties retain their current values
4. `element_add` arriving after `element_update` for the same `elem_id` is ignored (element already exists)
5. `element_update` arriving before `element_add` (out-of-order) creates the element from the update data

### Server-Side State Flow

```
Client A sends op → Server applies to room CRDT state → Broadcast to B, C, ...
                       ↓ (on save)
                    Persist to canvases.data (PostgreSQL JSONB)
```

When the first client joins an empty room, the server loads the current `canvases.data` JSONB into the room's in-memory CRDT state. When all clients leave, the in-memory state is discarded (the last `save` already persisted it).

---

## Authentication Flow

```
1. POST /auth/register or /auth/login
   → Receive access_token (15min) + refresh_token (7 days)

2. All subsequent REST requests:
   Authorization: Bearer <access_token>

3. WebSocket connection:
   ws://host/ws?token=<access_token>

4. When access_token expires:
   POST /auth/refresh { refresh_token }
   → Receive new access_token + refresh_token (old one revoked)

5. Share link access (no account required):
   POST /shares/:code/validate { password? }
   → Receive share_token (24h JWT)
   → Use share_token in WS join message
```
