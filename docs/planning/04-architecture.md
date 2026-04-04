# om-scrum-poker: System Architecture

**Version:** 3.1
**Date:** 2026-04-04
**Status:** Approved
**Depends on:** [01-ux-design.md](./01-ux-design.md)
**Supersedes:** Version 2.0 (2026-04-04)

---

## Changelog

### v3.0 -> v3.1

| Change | Reason (review iteration 3 ref) |
|---|---|
| **Health endpoint: `/health`** confirmed as canonical path (no change needed -- backend plan aligned to this). | Review iter 3, item 3.2 |
| **`GET /api/rooms/{id}` response: `{ "exists": true }`** confirmed as canonical (no change needed -- backend plan aligned to this). | Review iter 3, item 3.3 |
| **Version bump to v3.1** for reconciliation with backend and frontend plans. | Review iter 3 final reconciliation. |

### v2.0 -> v3.0

| Change | Reason (review iteration 2 ref) |
|---|---|
| **Envelope format: `{ type, payload }` everywhere.** Rewrote ALL message examples to use `{ type, payload: {...} }`. Removed contradictory "flat format" references from changelog, decision log, and CLAUDE.md template. | Review 2.1: Architecture doc contradicted itself -- changelog/CLAUDE.md said "flat", but all detailed examples used `{ type, payload }`. Two of three documents already used envelope format in their specs. |
| **WebSocket URL: `/ws/room/{roomId}`**. Added `/room/` segment to match frontend and backend plans. | Review 2.2: Architecture doc used `/ws/{roomId}`, frontend/backend plans used `/ws/room/:id`. |
| **Canonical event name: `name_updated`** (not `name_changed`). | Review 2.3: Backend plan used `name_changed`, others used `name_updated`. |
| **Canonical presence event: `presence`** (client sends `presence`, not `presence_update`). | Review 2.4: Frontend plan used `presence_update`, others used `presence`. |
| **Heartbeat: protocol-level pings every 5 seconds.** Fixed contradictory 30-second reference. Added explicit "no application-level heartbeat messages" note. | Review 2.5: Section 4.5 said 30s, hardcoded defaults said 5s. Frontend plan still had app-level heartbeat. |
| **Send buffer: 32 messages** (was 64 in Section 6.4). | Review 2.6: Backend plan reduced to 32, architecture doc was not updated. |
| **Room ID: 12 hex chars definitively.** Added generation code example for both browser and Go. | Review 2.7: Frontend plan still generated 8-char suffixes. |
| **Unified `room_state` payload.** Single canonical structure with `roomId`, `roomName`, `phase`, `participants`, `result`. Field names settled. | Review 2.8: Three docs had different field names and structures. |
| **Unified `votes_revealed` payload.** Flat stats alongside `votes` array. Field names: `average`, `median`, `uncertainCount`, `totalVoters`, `hasConsensus`, `spread`. | Review 2.9: Three different nesting/naming conventions. |
| **Error payload includes `code` field.** `{ type, payload: { code, message } }`. | Review 2.10: Backend plan Section 4.4 omitted `code`. |
| **`participant_joined` includes `status` field.** | Review 2.12: Backend plan omitted `status`. |
| **Added `roomName` to `join` message payload.** First joiner provides the display name; eliminates fragile slug-to-name parsing. | Review 3.8: Reverse-engineering display name from slug is lossy and fragile. |
| **Frontend project structure aligned with frontend plan.** `components/` with PascalCase dirs, `ws.ts` single file, `utils/`, per-component CSS. | Review 2.11: Architecture doc had different structure than frontend plan. |
| **Removed `utils/stats.ts` from frontend.** Server always includes stats in `votes_revealed` and `room_state` during reveal. No client-side stats calculation needed. | Review 3.7: Redundant computation, risk of client/server disagreement. |
| **Reconnection backoff: 500ms-10s cap with 30% jitter** (frontend-authoritative). Removed contradictory 30s-cap description. | Review 3.3: Frontend plan (10s cap) and backend plan (30s cap) disagreed. |
| **Protocol section marked as SINGLE SOURCE OF TRUTH** with explicit note. Backend and frontend plans must reference, not redefine. | Review 3.1: Two independent protocol specs caused all remaining inconsistencies. |
| **CSS approach: per-component CSS files** (aligned with frontend plan). | Review 3.6: Architecture doc had single `components.css`, frontend plan had per-component files. |
| **Unvote handling clarified.** `vote` with `value: ""` removes vote; server broadcasts `vote_retracted`. | Review item: Was implicit, now explicit. |

### v1.0 -> v2.0

| Change | Reason |
|---|---|
| Flattened backend from 4 layers to 2 packages (`domain` + `server`) | Review: Application layer added indirection without value. |
| Created single canonical WebSocket protocol spec in this document | Review: Multiple documents had contradictory message formats. |
| Changed CSS approach from CSS Modules to plain CSS with BEM naming | Review: Simpler for ~10 components. |
| Changed file extensions from `.jsx`/`.js` to `.tsx`/`.ts` | Review: TypeScript provides type safety for the WebSocket protocol. |
| Changed room creation to client-side (implicit on first join) | Review: Eliminated `POST /api/rooms` endpoint. |
| Changed room ID suffix from 8 hex chars to 12 hex chars (48 bits) | Review: Better entropy for brute-force resistance. |
| Reduced environment variables from 10 to 3 | Review: Over-specified configuration. |
| Removed application-level `ping`/`pong` messages | Review: Protocol-level pings are sufficient. |
| Removed `google/uuid` dependency | Review: Replaced with `crypto/rand` + `encoding/hex`. |
| Added `@preact/signals` to frontend dependency table | Review: Was missing. |
| Simplified project directory structure to match 2-package backend | Review: Aligned with recommended structure. |
| Added sequence diagram for voting-to-reveal flow | Review: Diagram for most complex flow. |

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Technology Stack](#2-technology-stack)
3. [Backend Architecture](#3-backend-architecture)
4. [WebSocket Protocol (Canonical)](#4-websocket-protocol-canonical)
5. [HTTP Endpoints](#5-http-endpoints)
6. [Security](#6-security)
7. [Deployment](#7-deployment)
8. [Reliability](#8-reliability)
9. [Project Structure](#9-project-structure)
10. [AI Documentation Strategy](#10-ai-documentation-strategy)
11. [Development Workflow](#11-development-workflow)

---

## 1. System Overview

### 1.1 Architecture Principles

- **Single deployable unit.** One binary, one container, one process.
- **Zero external dependencies.** No database, no Redis, no message broker. All state lives in process memory.
- **Ephemeral rooms.** Room data exists only while participants are connected. Server restart clears everything. This is by design.
- **WebSocket-first communication.** HTTP is used only for serving the SPA, the health check, and the room existence check.
- **Two backend packages.** `domain` for pure business logic, `server` for everything else. No interfaces, no DTOs, no service layer.

### 1.2 High-Level Architecture Diagram

```
+-----------------------------------------------------------------+
|                       Client (Browser)                          |
|                                                                 |
|  +-------------+  +--------------+  +------------------------+  |
|  | SPA (Preact |  | WebSocket    |  | localStorage           |  |
|  | + TypeScript|  | Client       |  | (userName, sessionId)  |  |
|  +-------------+  +------+-------+  +------------------------+  |
|                          |                                      |
+--------------------------+--------------------------------------+
                           | wss:// (upgrade from HTTP)
                           |
+--------------------------+--------------------------------------+
|                     Server (Go binary)                          |
|                          |                                      |
|  +-----------------------+-----------------------------------+  |
|  |                 server package                             |  |
|  |  +-------------+  +---------------+  +-----------------+  |  |
|  |  | HTTP Handler|  | WS Handler    |  | Static Files    |  |  |
|  |  | (health,    |  | (upgrade,     |  | (embedded SPA)  |  |  |
|  |  |  room check)|  |  dispatch,    |  |                 |  |  |
|  |  |             |  |  broadcast)   |  |                 |  |  |
|  |  +------+------+  +-------+-------+  +-----------------+  |  |
|  |         |                  |                               |  |
|  |  +------+------+  +-------+-------+  +-----------------+  |  |
|  |  | Room Store  |  | Connection    |  | GC Ticker       |  |  |
|  |  | (in-memory  |  | Hub           |  | (room expiry)   |  |  |
|  |  |  map+mutex) |  | (fan-out)     |  |                 |  |  |
|  |  +------+------+  +-------+-------+  +-----------------+  |  |
|  +---------+-----------------+-------------------------------|  |
|            |                 |                                   |
|  +---------+-----------------+-------------------------------+  |
|  |               domain package                               |  |
|  |  +-------------+  +--------------+  +-----------------+    |  |
|  |  |    Room     |  | Participant  |  | Stats / Vote    |    |  |
|  |  |             |  |              |  | Validation      |    |  |
|  |  +-------------+  +--------------+  +-----------------+    |  |
|  +------------------------------------------------------------+  |
|                                                                  |
+------------------------------------------------------------------+
```

### 1.3 Component Interactions Summary

| Component | Talks to | Via |
|---|---|---|
| Browser SPA | Server HTTP Handler | HTTP GET (page load, room existence check) |
| Browser SPA | Server WS Handler | WebSocket (all real-time events) |
| WS Handler | Domain entities | Go function calls (direct, no service layer) |
| WS Handler | Room Store + Hub | Go function calls (concrete types, no interfaces) |
| GC Ticker | Room Store | Periodic sweep (goroutine with ticker) |

---

## 2. Technology Stack

### 2.1 Backend

| Choice | Rationale |
|---|---|
| **Go** | Single static binary. `embed.FS` for frontend. Goroutines for WebSocket concurrency. Fast builds. |
| **`nhooyr.io/websocket`** | Only external dependency. Stdlib has no WebSocket. Context-aware, idiomatic Go API. |
| **No other dependencies** | Room ID generation uses `crypto/rand` + `encoding/hex`. Rate limiting is hand-rolled (~30 lines). No UUID library needed. |

### 2.2 Frontend

| Choice | Rationale |
|---|---|
| **Preact** (~4 KB gzip) | React-compatible component model. Sufficient for ~10 interactive components. |
| **@preact/signals** (~2 KB gzip) | Fine-grained reactivity without prop drilling. Separate package from `preact`. |
| **TypeScript** | Type safety for WebSocket protocol messages. All files use `.ts`/`.tsx` extensions. |
| **Vite** (dev dependency) | Fast HMR in development, optimized production builds. |
| **@preact/preset-vite** (dev dependency) | Vite plugin for Preact JSX transform. |

### 2.3 CSS Approach

**Decision: Plain CSS with BEM-like naming conventions. Per-component CSS files.**

- CSS Custom Properties for theming (colors, spacing, typography).
- A global `tokens.css` defines the design tokens.
- A global `reset.css` for baseline normalization.
- Each component has a co-located `.css` file (e.g., `CardDeck/card-deck.css`).
- With ~10 components and disciplined naming, class collisions are not a realistic risk.
- No CSS Modules, no Tailwind, no CSS-in-JS. Total CSS will be under 500 lines.

---

## 3. Backend Architecture

### 3.1 Package: `domain` (Pure Business Logic)

The domain package contains all business rules. It has zero imports from the `server` package or any infrastructure concern. It is a set of Go structs and methods that could run in any context.

#### Room

```go
// domain/room.go

type RoomPhase string

const (
    PhaseVoting RoomPhase = "voting"
    PhaseReveal RoomPhase = "reveal"
)

type Room struct {
    ID           string
    Name         string
    Phase        RoomPhase
    Participants map[string]*Participant // keyed by sessionId
    CreatedAt    time.Time
    LastActivity time.Time
}

func (r *Room) AddParticipant(p *Participant)
func (r *Room) RemoveParticipant(sessionID string)
func (r *Room) CastVote(sessionID string, value string) error
func (r *Room) RevealVotes() (*RoundResult, error)
func (r *Room) NewRound()
func (r *Room) Clear()
func (r *Room) ActiveParticipantCount() int
func (r *Room) VotedCount() int
func (r *Room) UpdateParticipantName(sessionID string, name string) error
```

Key behaviors:
- `CastVote` with an empty string `""` removes the participant's vote (un-vote). There is no separate `ClearVote` method.
- `CastVote` returns an error if phase is not `voting` or if the value is not in the allowed set.
- `RevealVotes` transitions phase to `reveal` and computes statistics.
- `NewRound` clears all votes and sets phase to `voting`.

#### Participant

```go
// domain/participant.go

type PresenceStatus string

const (
    StatusActive       PresenceStatus = "active"
    StatusIdle         PresenceStatus = "idle"
    StatusDisconnected PresenceStatus = "disconnected"
)

type Participant struct {
    SessionID string
    Name      string
    Vote      *string        // nil = no vote, "?" = uncertain, "5" = numeric
    Status    PresenceStatus
    LastSeen  time.Time
}

func (p *Participant) HasVoted() bool { return p.Vote != nil }
```

#### RoundResult

```go
// domain/result.go

type VoteEntry struct {
    SessionID string
    Name      string
    Value     *string
}

type RoundResult struct {
    Votes          []VoteEntry
    Average        *float64
    Median         *float64
    UncertainCount int
    TotalVoters    int
    HasConsensus   bool
    Spread         *[2]float64 // [min, max], nil if < 2 numeric votes
}
```

#### Domain Rules

| Rule | Enforcement |
|---|---|
| Votes only during voting phase | `CastVote` returns error if `Phase != PhaseVoting` |
| "?" excluded from average/median | `RevealVotes` filters "?" before calculation |
| Vote values from allowed set only | Allowed: `?`, `0`, `0.5`, `1`, `2`, `3`, `5`, `8`, `13`, `20`, `40`, `100` |
| Vote with `""` removes the vote | `CastVote` sets `Vote = nil` when value is empty |
| A vote replaces any previous vote | `CastVote` overwrites `Participant.Vote` |
| Reveal transitions the phase | `RevealVotes` sets `Phase = PhaseReveal` |
| New round clears all votes | `NewRound` nils all votes, sets `Phase = PhaseVoting` |
| Participant names are not unique | No uniqueness check on `Name` |

### 3.2 Package: `server` (All I/O Concerns)

The server package contains everything that is not pure business logic: HTTP handlers, WebSocket handler, in-memory store, connection hub, rate limit middleware, GC ticker, and config parsing. No interfaces -- concrete types only, since there is exactly one implementation of everything.

#### Room Store (In-Memory)

```go
// server/store.go

type roomEntry struct {
    mu   sync.Mutex
    room *domain.Room
}

type RoomStore struct {
    mu    sync.RWMutex
    rooms map[string]*roomEntry
}

func (s *RoomStore) Get(roomID string) (*domain.Room, bool)
func (s *RoomStore) GetOrCreate(roomID string, name string) *domain.Room
func (s *RoomStore) Delete(roomID string)
func (s *RoomStore) WithRoom(roomID string, fn func(r *domain.Room) error) error
func (s *RoomStore) ExpireInactive(maxAge time.Duration) int
```

`GetOrCreate` supports the client-side room creation flow: the room materializes when the first participant joins via WebSocket. The `name` parameter comes from the `join` message's `roomName` field.

#### Connection Hub (WebSocket Broadcasting)

```go
// server/hub.go

type Hub struct {
    mu    sync.RWMutex
    rooms map[string]map[string]*websocket.Conn // roomID -> sessionID -> conn
}

func (h *Hub) Register(roomID, sessionID string, conn *websocket.Conn)
func (h *Hub) Unregister(roomID, sessionID string)
func (h *Hub) BroadcastToRoom(roomID string, msg any)
func (h *Hub) SendTo(roomID, sessionID string, msg any)
func (h *Hub) ConnectionCount(roomID string) int
```

Each connection has a buffered send channel (capacity 32). If the channel fills (slow client), the connection is closed.

#### WebSocket Handler

The handler:
1. Upgrades HTTP to WebSocket at `/ws/room/{roomId}`.
2. Reads the first message, which must be `join`.
3. Calls `store.GetOrCreate()` to create the room if it does not exist, using the `roomName` from the `join` message.
4. Registers with the Hub and sends `room_state` to the joining client.
5. Enters a read loop dispatching messages to domain methods directly.
6. On disconnect: marks participant as disconnected, broadcasts `presence_changed`.

#### GC Ticker

A background goroutine runs every 10 minutes. Rooms with zero connected participants and `LastActivity` older than 24 hours are deleted. Both values are hardcoded defaults.

#### Rate Limiting

Hand-rolled token bucket per IP address (~30 lines). No external dependency.

| Scope | Limit | Burst |
|---|---|---|
| WebSocket upgrade (`/ws/*`) | 10/minute per IP | 20 |
| WebSocket messages (inbound) | 30/second per connection | 50 |

#### Config

Three environment variables. Everything else uses sensible hardcoded defaults.

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `HOST` | `0.0.0.0` | HTTP listen address |
| `TRUST_PROXY` | `false` | If `true`, read client IP from `X-Forwarded-For` for rate limiting |

---

## 4. WebSocket Protocol (Canonical)

> **THIS IS THE SINGLE SOURCE OF TRUTH for the WebSocket protocol.**
> Frontend and backend plans MUST reference this section, not redefine the protocol.
> If any other document contradicts this section, this section wins.

### 4.1 Connection

- **URL:** `ws(s)://{host}/ws/room/{roomId}`
- **Example:** `wss://poker.example.com/ws/room/sprint-42-a3f1c9b2d4e6`
- **Max message size:** 1 KB (inbound). Messages exceeding this close the connection.

### 4.2 Message Envelope Format

**Decision: `{ type, payload }` envelope for ALL messages.**

Every WebSocket message is a JSON object with exactly two top-level keys:

```json
{ "type": "<event_name>", "payload": { ... } }
```

**Rationale:** All three planning documents' detailed protocol examples already used this format. The "flat" declarations in v2.0 were changelog/summary text that was never reconciled with the actual examples. The envelope format provides a consistent dispatch pattern: read `type`, then deserialize `payload` into the appropriate struct. For events with no data, `payload` is an empty object `{}`.

### 4.3 Canonical Event Table

This table lists ALL WebSocket events. There are no other events.

| Event type | Direction | Payload fields | Description |
|---|---|---|---|
| `join` | client -> server | `sessionId`, `userName`, `roomName` | First message after connection. Creates room if it does not exist. |
| `vote` | client -> server | `value` | Cast or retract a vote. `value: ""` retracts. |
| `reveal` | client -> server | *(empty)* | Trigger vote reveal. |
| `new_round` | client -> server | *(empty)* | Clear all votes, start new voting round. |
| `clear_room` | client -> server | *(empty)* | Remove all participants, reset room. |
| `update_name` | client -> server | `userName` | Change display name. |
| `presence` | client -> server | `status` | Report presence: `"active"` or `"idle"`. |
| `leave` | client -> server | *(empty)* | Explicit room leave (from Settings menu). |
| `room_state` | server -> client | `roomId`, `roomName`, `phase`, `participants`, `result` | Full room snapshot. Sent once on join/reconnect. |
| `participant_joined` | server -> client | `sessionId`, `userName`, `status` | New participant entered the room. |
| `participant_left` | server -> client | `sessionId` | Participant explicitly left. |
| `vote_cast` | server -> client | `sessionId` | Someone voted (no value revealed). |
| `vote_retracted` | server -> client | `sessionId` | Someone retracted their vote. |
| `votes_revealed` | server -> client | `votes`, `average`, `median`, `uncertainCount`, `totalVoters`, `hasConsensus`, `spread` | All votes + statistics. |
| `round_reset` | server -> client | *(empty)* | New round started. |
| `room_cleared` | server -> client | *(empty)* | Room was cleared. |
| `presence_changed` | server -> client | `sessionId`, `status` | Participant presence changed. |
| `name_updated` | server -> client | `sessionId`, `userName` | Participant changed their name. |
| `error` | server -> client | `code`, `message` | Error sent to a single client. |

### 4.4 Client-to-Server Messages (Full Payload Examples)

#### `join` (required first message)

```json
{
  "type": "join",
  "payload": {
    "sessionId": "550e8400-e29b-41d4-a716-446655440000",
    "userName": "Alice",
    "roomName": "Sprint 42 Planning"
  }
}
```

Must be the first message after WebSocket connection. The `sessionId` is a UUID v4 generated by the browser (`crypto.randomUUID()`) and stored in `localStorage`. On reconnect, the same `sessionId` restores the participant's state (including their vote).

The `roomName` is the human-readable display name. The first joiner's `roomName` is used as the room's display name. Subsequent joiners' `roomName` is ignored (they receive the existing name in `room_state`). This avoids fragile slug-to-name reverse-engineering on the server.

If the room does not exist, it is created implicitly. The `roomId` comes from the URL path, not the message.

#### `vote`

```json
{ "type": "vote", "payload": { "value": "5" } }
```

`value` must be one of: `?`, `0`, `0.5`, `1`, `2`, `3`, `5`, `8`, `13`, `20`, `40`, `100`.

To un-vote (retract a vote), send:

```json
{ "type": "vote", "payload": { "value": "" } }
```

This sets the participant's vote to nil. The server broadcasts `vote_retracted` to the room.

#### `reveal`

```json
{ "type": "reveal", "payload": {} }
```

Triggers vote reveal. Transitions room to reveal phase.

#### `new_round`

```json
{ "type": "new_round", "payload": {} }
```

Clears all votes, transitions room to voting phase.

#### `clear_room`

```json
{ "type": "clear_room", "payload": {} }
```

Removes all participants and resets room state.

#### `update_name`

```json
{ "type": "update_name", "payload": { "userName": "Bob" } }
```

Changes the participant's display name. Server validates (1-30 chars, trimmed). Server broadcasts `name_updated` to all participants.

#### `presence`

```json
{ "type": "presence", "payload": { "status": "idle" } }
```

`status` is `"active"` or `"idle"`. Sent when:
- Tab becomes hidden (`visibilitychange`) -> `"idle"`
- 2 minutes of no user interaction -> `"idle"`
- Tab becomes visible or interaction resumes -> `"active"`

#### `leave`

```json
{ "type": "leave", "payload": {} }
```

Explicit room leave (from Settings > "Leave room"). Server immediately removes the participant (no grace period, no dimmed state). This differs from an unexpected disconnect, where the participant is marked as disconnected with their vote preserved.

### 4.5 Server-to-Client Messages (Full Payload Examples)

#### `room_state` (sent once on join)

```json
{
  "type": "room_state",
  "payload": {
    "roomId": "sprint-42-a3f1c9b2d4e6",
    "roomName": "Sprint 42 Planning",
    "phase": "voting",
    "participants": [
      {
        "sessionId": "550e8400-e29b-41d4-a716-446655440000",
        "userName": "Alice",
        "hasVoted": true,
        "vote": null,
        "status": "active"
      }
    ],
    "result": null
  }
}
```

Full room snapshot. During voting phase, `vote` is always `null` (hidden); `hasVoted` indicates whether the participant has voted. During reveal phase, `vote` contains the actual value and `result` contains statistics.

The `result` field uses the same structure as `votes_revealed` payload (when present, during reveal phase):

```json
{
  "type": "room_state",
  "payload": {
    "roomId": "sprint-42-a3f1c9b2d4e6",
    "roomName": "Sprint 42 Planning",
    "phase": "reveal",
    "participants": [
      {
        "sessionId": "...",
        "userName": "Alice",
        "hasVoted": true,
        "vote": "5",
        "status": "active"
      },
      {
        "sessionId": "...",
        "userName": "Bob",
        "hasVoted": true,
        "vote": "8",
        "status": "active"
      }
    ],
    "result": {
      "votes": [
        { "sessionId": "...", "userName": "Alice", "value": "5" },
        { "sessionId": "...", "userName": "Bob", "value": "8" }
      ],
      "average": 6.5,
      "median": 6.5,
      "uncertainCount": 0,
      "totalVoters": 2,
      "hasConsensus": false,
      "spread": [5, 8]
    }
  }
}
```

Because the server always includes stats in `room_state` during reveal phase, the frontend does NOT need its own stats calculation logic. It displays what the server sends.

#### `participant_joined`

```json
{
  "type": "participant_joined",
  "payload": {
    "sessionId": "550e8400-...",
    "userName": "Alice",
    "status": "active"
  }
}
```

Broadcast to all participants in the room (except the joiner, who gets `room_state`).

#### `participant_left`

```json
{
  "type": "participant_left",
  "payload": {
    "sessionId": "550e8400-..."
  }
}
```

Broadcast when a participant explicitly leaves (via `leave` message).

#### `vote_cast`

```json
{
  "type": "vote_cast",
  "payload": {
    "sessionId": "550e8400-..."
  }
}
```

Broadcast when a participant votes. No vote value -- just the fact of voting. All clients update `hasVoted` for this participant.

#### `vote_retracted`

```json
{
  "type": "vote_retracted",
  "payload": {
    "sessionId": "550e8400-..."
  }
}
```

Broadcast when a participant un-votes (sends `vote` with `value: ""`).

#### `votes_revealed`

```json
{
  "type": "votes_revealed",
  "payload": {
    "votes": [
      { "sessionId": "...", "userName": "Alice", "value": "5" },
      { "sessionId": "...", "userName": "Bob", "value": "?" }
    ],
    "average": 5.0,
    "median": 5.0,
    "uncertainCount": 1,
    "totalVoters": 2,
    "hasConsensus": true,
    "spread": null
  }
}
```

Broadcast with full results. `average` and `median` are `null` if there are no numeric votes. `spread` is a `[min, max]` array of two numbers, or `null` if fewer than 2 numeric votes (including when all numeric votes are the same value, i.e. consensus).

Field definitions:
- `votes`: Array of `{ sessionId, userName, value }` for every participant who voted.
- `average`: Arithmetic mean of numeric votes (excludes "?"), or `null`.
- `median`: Median of numeric votes (excludes "?"), or `null`.
- `uncertainCount`: Number of "?" votes.
- `totalVoters`: Total number of participants who voted (including "?").
- `hasConsensus`: `true` if all numeric voters chose the same value.
- `spread`: `[min, max]` of numeric votes, or `null` if < 2 distinct numeric votes.

#### `round_reset`

```json
{ "type": "round_reset", "payload": {} }
```

Broadcast. All votes cleared, phase is voting.

#### `room_cleared`

```json
{ "type": "room_cleared", "payload": {} }
```

Broadcast. All participants removed. Clients receiving this should navigate to the home page or show an appropriate message.

#### `presence_changed`

```json
{
  "type": "presence_changed",
  "payload": {
    "sessionId": "550e8400-...",
    "status": "idle"
  }
}
```

Broadcast. `status` is `"active"`, `"idle"`, or `"disconnected"`.

#### `name_updated`

```json
{
  "type": "name_updated",
  "payload": {
    "sessionId": "550e8400-...",
    "userName": "New Name"
  }
}
```

Broadcast when a participant changes their display name.

#### `error`

```json
{
  "type": "error",
  "payload": {
    "code": "invalid_vote",
    "message": "Invalid vote value"
  }
}
```

Sent to a single client when they perform an invalid action. The client should display this as a toast notification.

Error codes:
- `invalid_vote` -- vote value not in allowed set, or voting during reveal phase.
- `invalid_name` -- name too short, too long, or contains invalid characters.
- `room_not_found` -- room does not exist (e.g., after server restart).
- `rate_limited` -- too many messages.
- `invalid_message` -- malformed JSON or unknown message type.

### 4.6 Heartbeat

Uses **protocol-level WebSocket pings only** (`nhooyr.io/websocket` supports these natively). The server sends protocol-level pings every **5 seconds**. If no pong is received within 10 seconds, the server marks the participant as disconnected and closes the connection.

**There are NO application-level heartbeat messages.** No `ping` type, no `pong` type, no `heartbeat` type in the protocol. The browser's WebSocket implementation handles protocol-level pong responses automatically -- no application code needed on the client side.

If proxy issues arise in production (some proxies strip WebSocket pings), application-level heartbeat can be added later as a new message type.

### 4.7 Reconnection

Client-side reconnection uses exponential backoff: **500ms, 1s, 2s, 4s, 8s, 10s** (capped at 10s), with **30% jitter**. The frontend plan is authoritative for client-side reconnection behavior.

On reconnect, the client sends a `join` message with the same `sessionId`. The server recognizes the existing session and restores the participant's state (including their vote). The server broadcasts `presence_changed { status: "active" }` to the room.

After 30 seconds of failed reconnection (wall-clock timeout), the client shows "Connection lost. [Retry]".

If the room no longer exists on reconnect (e.g., after server restart), the server sends an `error` with code `room_not_found` and the client redirects to the home page.

### 4.8 Sequence Diagram: Vote -> Reveal -> New Round

```
Alice                     Server                    Bob
  |                         |                         |
  |-- { type: "vote",      |                         |
  |     payload:            |                         |
  |     { value: "8" } } ->|                         |
  |                         |                         |
  |                    CastVote("alice-session", "8") |
  |                    Validate value (OK)            |
  |                    Validate phase=voting (OK)     |
  |                    Store vote                     |
  |                         |                         |
  |                         |-- { type: "vote_cast",  |
  |                         |     payload:            |
  |                         |  { sessionId: "..." }}->|
  |                         |                         |
  |                         |<- { type: "vote",       |
  |                         |     payload:            |
  |                         |  { value: "5" } }  -----|
  |                         |                         |
  |                    CastVote("bob-session", "5")   |
  |                         |                         |
  |<- { type: "vote_cast", |                         |
  |     payload:            |                         |
  |  { sessionId: "..." }} |                         |
  |                         |                         |
  |-- { type: "reveal",    |                         |
  |     payload: {} }  --->|                         |
  |                         |                         |
  |                    RevealVotes()                   |
  |                    Phase = reveal                  |
  |                    Compute stats:                  |
  |                      avg=6.5, median=6.5           |
  |                      consensus=false               |
  |                      spread=[5,8]                  |
  |                         |                         |
  |<- { type:               |-- { type:               |
  |     "votes_revealed",   |     "votes_revealed",   |
  |     payload: {          |     payload: {          |
  |       votes: [...],     |       votes: [...],     |
  |       average: 6.5,     |       average: 6.5,     |
  |       ... }}            |       ... }}         -->|
  |                         |                         |
  |                         |                         |
  |-- { type: "new_round", |                         |
  |     payload: {} }  --->|                         |
  |                         |                         |
  |                    NewRound()                      |
  |                    Clear all votes                 |
  |                    Phase = voting                  |
  |                         |                         |
  |<- { type:               |-- { type:               |
  |     "round_reset",      |     "round_reset",      |
  |     payload: {} }       |     payload: {} }   -->|
  |                         |                         |
```

---

## 5. HTTP Endpoints

| Route | Method | Purpose |
|---|---|---|
| `/` | GET | Serve `index.html` (SPA entry point) |
| `/assets/*` | GET | Serve static assets (JS, CSS, images) |
| `/api/rooms/{id}` | GET | Check if a room exists. Returns `200 { "exists": true }` or `404`. Used by the SPA before showing the room page vs. "Room not found." |
| `/ws/room/{roomId}` | GET | WebSocket upgrade endpoint |
| `/health` | GET | Health check. Returns `200 { "status": "ok", "rooms": 42, "connections": 128 }` |
| `/*` (catch-all) | GET | Serve `index.html` for SPA client-side routing (any path not matching above) |

There is no `POST /api/rooms` endpoint. Room creation is purely client-side and WebSocket-based:

1. The frontend generates a slug from the room name and appends a 12-character hex suffix.
2. The frontend navigates to `/room/{slug}-{id}`.
3. The frontend opens a WebSocket to `/ws/room/{slug}-{id}`.
4. The first `join` message includes `roomName` (the display name).
5. The server calls `store.GetOrCreate()` with the room ID and display name. The room materializes.

No REST call is involved in room creation.

### 5.1 Room ID Format

```
sprint-42-a3f1c9b2d4e6
|________|  |__________|
   slug     12 hex chars (48 bits entropy)
```

- **Slug:** Derived from the room name. Lowercase, alphanumeric + hyphens, max 30 chars.
- **Suffix:** 12 hexadecimal characters generated from 6 random bytes. 48 bits = 281,474,976,710,656 possible values.
- **Generation (browser):**
  ```ts
  const bytes = crypto.getRandomValues(new Uint8Array(6));
  const hex = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('');
  // hex is always exactly 12 lowercase hex characters
  ```
- **Generation (Go, for tests/tooling):**
  ```go
  b := make([]byte, 6)
  crypto_rand.Read(b)
  hex := hex.EncodeToString(b) // 12 hex chars
  ```
- **Full ID:** `slugify(roomName) + "-" + hex`
- **Regex to extract suffix:** `/-[a-f0-9]{12}$/`

### 5.2 Static File Serving

The frontend build output is embedded into the Go binary at compile time:

```go
//go:embed all:frontend/dist
var frontendFS embed.FS
```

The server tries to serve the requested file. If not found, it serves `index.html` (SPA fallback for client-side routing).

---

## 6. Security

### 6.1 Room ID Brute-Force Resistance

12 hex chars = 48 bits = 281 trillion possible suffixes. At 100 requests/second (generous for a single server with rate limiting), brute-forcing a specific room takes ~89,000 years on average. Room contents are not sensitive (names and point estimates), so this is vastly sufficient.

### 6.2 Rate Limiting

In-memory token bucket per IP address. See Section 3.2.

### 6.3 Input Validation

| Input | Validation |
|---|---|
| Room name | Max 60 chars. Alphanumeric, spaces, hyphens, underscores. Trimmed. |
| User name | 1-30 chars (after trim). |
| Vote value | Whitelist: `?`, `0`, `0.5`, `1`, `2`, `3`, `5`, `8`, `13`, `20`, `40`, `100`, or `""` (un-vote). |
| Session ID | Valid UUID v4 format. |
| WebSocket messages | Max 1 KB. JSON parsed with strict struct mapping; unknown fields ignored. |

### 6.4 Connection Limits

| Limit | Value |
|---|---|
| Max connections per room | 50 |
| Max total connections | 1000 |
| Max message size | 1 KB |
| Send buffer per connection | 32 messages |
| Protocol-level ping interval | 5 seconds |
| Heartbeat timeout | 10 seconds without pong response |

### 6.5 CORS

- **Production:** No CORS headers needed (same-origin, embedded SPA).
- **Development:** Vite dev server proxies `/api/*` and `/ws/*` to the Go backend, so no CORS configuration is needed in development either.

### 6.6 WebSocket Origin Check

The WebSocket upgrade handler validates the `Origin` header to match the expected host. This prevents cross-site WebSocket hijacking.

### 6.7 No Sensitive Data at Rest

No data written to disk. No cookies. No logs contain vote values or user names. Server restart erases all state.

---

## 7. Deployment

### 7.1 Go Binary

The production artifact is a single Go binary (~15 MB) containing the compiled server and all frontend assets.

```bash
# Build frontend
cd frontend && npm ci && npm run build

# Build Go binary (embeds frontend/dist)
cd .. && CGO_ENABLED=0 go build -ldflags="-s -w" -o om-scrum-poker ./cmd/server

# Run
./om-scrum-poker
# Override port: PORT=3000 ./om-scrum-poker
```

### 7.2 Docker

```dockerfile
# Build stage: frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build stage: Go
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /om-scrum-poker ./cmd/server

# Runtime
FROM scratch
COPY --from=build /om-scrum-poker /om-scrum-poker
EXPOSE 8080
ENTRYPOINT ["/om-scrum-poker"]
```

Final image: ~15 MB. No OS, no shell. `FROM scratch`.

```bash
docker build -t om-scrum-poker .
docker run -p 8080:8080 om-scrum-poker
```

---

## 8. Reliability

### 8.1 Graceful Shutdown

On `SIGTERM` or `SIGINT`:
1. Stop accepting new connections.
2. Send WebSocket close frame (1001 Going Away) to all clients.
3. Wait up to 10 seconds for in-flight operations.
4. Exit.

### 8.2 Room Garbage Collection

Background goroutine runs every 10 minutes. A room is eligible for deletion if it has zero connected participants and `LastActivity` is older than 24 hours.

**Memory estimates:**
- Room with 10 participants: ~1.7 KB
- 1000 rooms: ~1.7 MB
- Memory is never a concern for this application.

### 8.3 Connection Cleanup

When a WebSocket connection closes (any reason):
1. Unregister from Hub.
2. Mark participant as `disconnected` in domain.
3. Broadcast `presence_changed` to room.
4. Goroutines (read/write pumps) exit and are garbage-collected.

No resource leaks. Each connection is self-cleaning.

### 8.4 Server Restart

All in-memory state is lost. Clients in "Reconnecting..." state will fail to rejoin. After 30 seconds, clients show "Connection lost. [Retry]". If the room no longer exists, the client receives an `error` and is redirected to the home page.

This is acceptable: planning poker rooms are ephemeral (minutes to hours). Adding persistence would violate KISS for negligible benefit.

### 8.5 Panic Recovery

HTTP and WebSocket handlers are wrapped in recovery middleware. A panicking connection does not crash the server.

---

## 9. Project Structure

```
om-scrum-poker/
+-- CLAUDE.md                          # AI assistant entry point
+-- README.md                          # Human-readable project overview
+-- Dockerfile
+-- .dockerignore
+-- go.mod
+-- go.sum
+-- Makefile
|
+-- cmd/
|   +-- server/
|       +-- main.go                    # Entry point: config, wiring, start
|
+-- internal/
|   +-- domain/                        # Pure business logic (zero I/O imports)
|   |   +-- room.go                    # Room entity + methods
|   |   +-- participant.go            # Participant entity
|   |   +-- result.go                  # RoundResult + stats calculation
|   |   +-- vote.go                    # Vote value validation
|   |   +-- errors.go                  # Domain error types
|   |   +-- room_test.go
|   |   +-- result_test.go
|   |   +-- vote_test.go
|   |
|   +-- server/                        # All I/O: HTTP, WS, store, hub, GC
|       +-- store.go                   # In-memory RoomStore
|       +-- store_test.go
|       +-- hub.go                     # WebSocket connection hub + broadcast
|       +-- hub_test.go
|       +-- ws_handler.go             # WebSocket upgrade + message dispatch
|       +-- ws_handler_test.go
|       +-- http_handler.go           # HTTP routes (room check, health, static)
|       +-- http_handler_test.go
|       +-- message.go                # Message type constants + structs
|       +-- middleware.go             # Rate limiting, recovery, logging
|       +-- gc.go                      # Room garbage collector
|       +-- config.go                  # Environment variable parsing (3 vars)
|       +-- server.go                  # Server struct, wiring, ListenAndServe
|
+-- frontend/                          # Preact SPA (TypeScript)
|   +-- package.json
|   +-- package-lock.json
|   +-- tsconfig.json
|   +-- vite.config.ts
|   +-- index.html
|   +-- public/
|   |   +-- favicon.svg
|   +-- src/
|   |   +-- main.tsx                   # App entry, router setup
|   |   +-- app.tsx                    # Top-level component, routing
|   |   +-- state.ts                   # All signals + localStorage helpers + types
|   |   +-- ws.ts                      # WebSocket client, handlers, message builders
|   |   |
|   |   +-- components/
|   |   |   +-- HomePage/
|   |   |   |   +-- HomePage.tsx
|   |   |   |   +-- home-page.css
|   |   |   +-- RoomPage/
|   |   |   |   +-- RoomPage.tsx
|   |   |   |   +-- room-page.css
|   |   |   +-- Header/
|   |   |   |   +-- Header.tsx
|   |   |   |   +-- header.css
|   |   |   +-- NameEntryModal/
|   |   |   |   +-- NameEntryModal.tsx
|   |   |   |   +-- name-entry-modal.css
|   |   |   +-- ParticipantList/
|   |   |   |   +-- ParticipantList.tsx
|   |   |   |   +-- participant-list.css
|   |   |   +-- ParticipantCard/
|   |   |   |   +-- ParticipantCard.tsx
|   |   |   |   +-- participant-card.css
|   |   |   +-- CardDeck/
|   |   |   |   +-- CardDeck.tsx
|   |   |   |   +-- card-deck.css
|   |   |   +-- Card/
|   |   |   |   +-- Card.tsx
|   |   |   |   +-- card.css
|   |   |   +-- Toast/
|   |   |   |   +-- Toast.tsx
|   |   |   |   +-- toast.css
|   |   |   +-- ConfirmDialog/
|   |   |   |   +-- ConfirmDialog.tsx
|   |   |   |   +-- confirm-dialog.css
|   |   |
|   |   +-- hooks/
|   |   |   +-- usePresence.ts         # Visibility/activity tracking for presence
|   |   |
|   |   +-- utils/
|   |   |   +-- room-url.ts            # Slug generation, room ID parsing
|   |   |
|   |   +-- styles/
|   |       +-- tokens.css             # CSS custom properties (colors, spacing, radii)
|   |       +-- reset.css              # Minimal CSS reset
|   |       +-- global.css             # Body, typography, base element styles
|   |
|   +-- dist/                          # Build output (git-ignored, embedded by Go)
|
+-- docs/
    +-- planning/
        +-- 01-ux-design.md
        +-- 02-frontend-plan.md
        +-- 03-backend-plan.md
        +-- 04-architecture.md         # THIS DOCUMENT (canonical)
        +-- 05-review-iteration-1.md
        +-- 06-review-iteration-2.md
```

### Why This Structure

| Decision | Rationale |
|---|---|
| 2 backend packages (`domain` + `server`) | Matches the project's actual complexity. No unused abstraction layers. |
| No interfaces | One implementation of everything. Test domain directly (pure logic). Test server with real in-memory store (integration-style). |
| `server/message.go` | Single file for all protocol types/constants. Update one file when protocol changes. |
| 10 frontend components (not 16) | `CopyLinkButton`, `SettingsDropdown`, `ReconnectionBanner`, `ActionButtons` merged into parent components. |
| Single `state.ts` (not 3 files) | 11 signals. One file is perfectly readable. No cross-file import gymnastics. |
| No `types/` directory | ~10 type definitions co-located with the code that uses them (`state.ts`, `ws.ts`). |
| Per-component CSS files (not single `components.css`) | Co-location with components. Each component's styles live next to its code. |
| No `utils/stats.ts` | Server always includes stats in `votes_revealed` and `room_state` during reveal phase. Frontend just displays what server sends. |
| TypeScript (`.tsx`/`.ts`, not `.jsx`/`.js`) | Type safety for WebSocket messages. |
| Frontend structure matches frontend plan | `components/` with PascalCase directories, `ws.ts` single file, `utils/`, `hooks/`. Architecture doc defers to frontend plan on frontend structure details. |

---

## 10. AI Documentation Strategy

### 10.1 Files

| File | Purpose | When to Update |
|---|---|---|
| `CLAUDE.md` (repo root) | Entry point for AI assistants. Build commands, directory map, conventions. Must fit in one screen (~40 lines). | On any structural change. |
| `docs/planning/04-architecture.md` | This document. Canonical architecture, WebSocket protocol, project structure. | On architectural or protocol changes. |

There is no separate `API.md`. The WebSocket protocol lives in Section 4 of this document. One source of truth, not two.

### 10.2 CLAUDE.md Template

```
# CLAUDE.md

## Project
om-scrum-poker: real-time scrum poker web app.
Single Go binary serving embedded Preact SPA. WebSocket for real-time. No database.

## Quick Start
go run ./cmd/server                    # backend on :8080
cd frontend && npm run dev             # frontend on :5173 (proxies to :8080)

## Build
cd frontend && npm ci && npm run build
cd .. && CGO_ENABLED=0 go build -ldflags="-s -w" -o om-scrum-poker ./cmd/server

## Test
go test -race ./...
cd frontend && npm test

## Key Directories
cmd/server/          - Entry point (main.go)
internal/domain/     - Business logic (Room, Participant, Vote). Pure, no I/O.
internal/server/     - HTTP, WebSocket, store, hub, GC. All I/O.
frontend/src/        - Preact SPA (TypeScript)

## Conventions
- Domain package has zero imports from server package.
- All WebSocket messages use envelope format: { type, payload: {...} }.
- Protocol spec: docs/planning/04-architecture.md, Section 4 (SINGLE SOURCE OF TRUTH).
- Config: PORT, HOST, TRUST_PROXY env vars only.
- Single external Go dependency: nhooyr.io/websocket.
```

### 10.3 Code Conventions for AI Comprehension

| Practice | Rationale |
|---|---|
| Package-level doc comments | AI learns package purpose without reading all files. |
| No `init()` functions | Execution traceable from `main()`. |
| Method names correlate to message types | `vote` message -> `CastVote` method. Predictable mapping. |
| Exported functions have doc comments with error conditions | AI can trace error paths precisely. |
| All protocol constants in `server/message.go` | One file to read for the full protocol. |

---

## 11. Development Workflow

### 11.1 Prerequisites

- Go 1.22+
- Node.js 20+
- npm 10+
- (Optional) Docker

### 11.2 Development Mode

Two terminals:

**Terminal 1 -- Go backend:**

```bash
go run ./cmd/server
# Serves on :8080
```

**Terminal 2 -- Frontend dev server:**

```bash
cd frontend && npm run dev
# Vite dev server on :5173
# Proxies /api/* and /ws/* to :8080
```

Vite config:

```ts
// frontend/vite.config.ts
export default {
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': { target: 'http://localhost:8080', ws: true }
    }
  }
}
```

### 11.3 Makefile

```makefile
.PHONY: dev build test clean docker

dev:
	go run ./cmd/server

build:
	cd frontend && npm ci && npm run build
	CGO_ENABLED=0 go build -ldflags="-s -w" -o om-scrum-poker ./cmd/server

test:
	go test -race ./...
	cd frontend && npm test

clean:
	rm -f om-scrum-poker
	rm -rf frontend/dist

docker:
	docker build -t om-scrum-poker .
```

### 11.4 Testing Strategy

| What | How | Focus |
|---|---|---|
| Domain logic | `go test ./internal/domain/...` | Vote validation, phase transitions, stats calculation, edge cases (all "?", empty room, single participant). |
| Server (integration) | `go test ./internal/server/...` | Store concurrency, hub message delivery, handler dispatch, GC behavior. Real in-memory store, no mocks. |
| Frontend components | Vitest + Preact Testing Library | Component rendering per phase, user interactions. |
| Frontend WebSocket | Vitest | State transitions on message receipt. Mock WebSocket. |
| End-to-end | Manual (Playwright optional, deferred) | Full flow: create room, join, vote, reveal. Not needed for initial release. |

---

## Appendix A: Decision Log

| Decision | Alternatives Considered | Rationale |
|---|---|---|
| Go for backend | Rust, Node.js, Python | Single binary, stdlib coverage, goroutines for WS, `embed.FS`. |
| Preact for frontend | Vanilla JS, React, Svelte | 4 KB, React-compatible API, sufficient complexity. |
| `nhooyr/websocket` | `gorilla/websocket`, stdlib | Context-aware, maintained. `gorilla/websocket` in maintenance mode. |
| Plain CSS with BEM | CSS Modules, Tailwind, CSS-in-JS | ~10 components, ~500 lines CSS. BEM is sufficient. No build config needed. |
| `{ type, payload }` envelope format | Flat `{ type, ...fields }` | Consistent dispatch pattern: always read `type`, always deserialize `payload`. All detailed protocol specs already used this format. Flat was declared in summaries but never reflected in examples. |
| Client-side room creation | `POST /api/rooms` | Eliminates an HTTP round-trip. Room materializes on first join. Matches in-memory-only philosophy. |
| 12-char hex room ID suffix | 8-char (UUID), nanoid | 48 bits entropy. No external dependency. `crypto/rand` + hex. |
| Protocol-level pings only | Dual (protocol + app-level) | Simpler protocol. Modern proxies pass WS pings. Add app-level later if needed. |
| 2 backend packages | 4-layer Clean Architecture | One entity, one store, no DB. Clean Architecture adds ceremony without payoff. |
| 3 env vars | 10 env vars | Who tunes heartbeat intervals for a 5-person poker tool? Add config when someone asks. |
| `roomName` in `join` message | Slug-to-name extraction on server | Avoids lossy reverse-engineering of display name. First joiner provides it. One field added to one message. |
| Server-side stats only | Duplicate stats in frontend | Server includes stats in `votes_revealed` and `room_state` during reveal. No client-side `stats.ts` needed. Eliminates risk of disagreement. |
| Per-component CSS files | Single `components.css` | Co-location with components. Matches frontend plan structure. |

## Appendix B: Capacity Estimates

| Metric | Estimate |
|---|---|
| Memory per room (10 participants) | ~1.7 KB |
| 1000 concurrent rooms | ~1.7 MB |
| Goroutines per connection | 2 (read + write pump) |
| WebSocket message size (typical) | 50-200 bytes |
| Messages/second per room (active voting) | ~5-10 |
| Server capacity | 5000+ concurrent connections |

## Appendix C: Future Considerations (Not in Scope)

- **Horizontal scaling:** Would require shared state (Redis) + pub/sub. Not needed -- single server handles thousands of rooms.
- **Persistent rooms:** Would require SQLite. Only if users request "continue yesterday's session."
- **Custom card decks:** Minor change. Add `deck` field to Room.
- **App-level heartbeat:** Add if proxy issues are reported. Two new message types (`ping`/`pong`), ~20 lines of code.
