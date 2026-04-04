# om-scrum-poker: Backend Technical Plan

**Version:** 3.1
**Date:** 2026-04-04
**Status:** Approved (post-review revision)
**Depends on:** [01-ux-design.md](./01-ux-design.md), [04-architecture.md](./04-architecture.md)

---

## Changelog

### v3.0 -> v3.1

| Change | Rationale | Review Ref |
|---|---|---|
| **Phase constant: `PhaseReveal Phase = "reveal"`** (was `PhaseRevealed Phase = "revealed"`). All references updated. | Review iter 3, item 3.1: Wire-level mismatch -- backend sent `"revealed"`, frontend/architecture expected `"reveal"`. | Review iter 3, item 3.1 |
| **Health endpoint: `/health`** (was `/api/health`). | Review iter 3, item 3.2: Architecture doc uses `/health`; simpler path for ops endpoint. | Review iter 3, item 3.2 |
| **`GET /api/rooms/{roomId}` response: `{ "exists": true }`** (was `{ "id", "name", "phase", "participantCount" }`). Aligned to architecture doc. | Review iter 3, item 3.3: Architecture doc is source of truth for endpoint contracts. | Review iter 3, item 3.3 |

### v2.0 -> v3.0

| Change | Rationale | Review Ref |
|---|---|---|
| **Removed Section 5 (WebSocket Protocol).** Backend plan no longer redefines the protocol. All protocol details reference Architecture Doc v3.0, Section 4. | Review 3.1: Two independent protocol specs caused all remaining inconsistencies. Architecture doc is the single source of truth. | Review iter 2, item 8 |
| **Event name: `name_updated`** (was `name_changed` in Section 5.3, 6.4). All code sketches and dispatch tables updated. | Review 2.3: Backend plan used `name_changed`, architecture doc and frontend plan use `name_updated`. | Review iter 2, item 3 |
| **`participant_joined` includes `status` field.** Added to broadcast payload. | Review 2.12: Backend plan omitted `status`. | Review iter 2, item 12 |
| **Error payload includes `code` field everywhere.** Section 4.4 examples and Section 13.2 now consistently use `{ code, message }`. | Review 2.10: Section 4.4 omitted `code`. | Review iter 2, item 13 |
| **`join` payload includes `roomName`.** Room name provided by first joiner, not reverse-engineered from slug. Updated Section 4.2 connection flow. | Review 3.8: Slug-to-name extraction is lossy and fragile. | Review iter 2, item 16 |
| **Heartbeat interval: 5 seconds consistently.** Removed contradictory "30 seconds" from Section 4.5 (was in old numbering). | Review 2.5: Section 4.5 said 30s, Section 6.5 said 5s. | Review iter 2, item 10 |
| **`room_state` and `votes_revealed` structures aligned to architecture doc.** Field names: `roomId`, `roomName`, `hasVoted`, `result`. Stats fields: `average`, `median`, `uncertainCount`, `totalVoters`, `hasConsensus`, `spread`. | Review 2.8, 2.9: Three different structures across docs. | Review iter 2, items 6-7 |
| **Removed reconnection backoff description.** Backend plan described client-side behavior (30s cap). Frontend plan is authoritative for client-side reconnection (10s cap, 30% jitter). | Review 3.3: Contradictory caps. | Review iter 2, item 18 |
| **All code sketches use `{ type, payload }` envelope.** No flat message examples remain. | Review 2.1: Envelope format unified across all docs. | Review iter 2, item 1 |

### v1.0 -> v2.0

| Change | Rationale | Review Item |
|---|---|---|
| Collapsed 4-layer architecture (`domain`/`app`/`infra`/`handler`) to 2 packages: `domain` + `server` | Application layer added indirection without value; single in-memory store needs no interface abstraction | [MUST] #1, Review 1.1/4.1 |
| Removed `github.com/google/uuid` dependency; use `crypto/rand` + `encoding/hex` | UUID package was used only for 8-char hex generation; 4 lines of stdlib replaces it | [MUST] #5 |
| Canonical WebSocket protocol with `{ type, payload }` envelope | Resolved contradictions between all planning docs | [MUST] #2 |
| WebSocket URL changed from `/ws/rooms/{id}` to `/ws/room/{roomId}` | Align with frontend plan and architecture doc path convention | [MUST] #3 |
| Unified all event names across protocol table | Resolved naming discrepancies (room_state vs room_snapshot, etc.) | [MUST] #4 |
| Removed standalone `internal/presence/` package; presence folded into Participant + Room methods | Eliminated parallel data structure anti-pattern | [SHOULD] #6 |
| Added explicit "leave room" vs "disconnect" semantics | Missing spec identified in review | [SHOULD] #7 |
| Added `name_updated` event to protocol | Missing spec identified in review | [SHOULD] #8 |
| Health check now returns room/connection metrics | Upgraded from bare `{"status":"ok"}` | [SHOULD] #9 |
| Room ID suffix changed from 8-char hex (32 bits) to 12-char hex (48 bits) | Better entropy margin against enumeration | [COULD] #10 |
| Removed `POST /api/rooms` endpoint; rooms created implicitly on first join | Aligned with frontend plan's client-side room creation | Review 4.5 |
| Reduced config to `PORT`, `HOST`, `TRUST_PROXY` | Over-specified configuration anti-pattern | Review 5.3 |
| Send buffer reduced from 64 to 32 | Align with architecture doc | Review 1.6 |

---

## Table of Contents

1. [Technology Choice](#1-technology-choice)
2. [Project Structure](#2-project-structure)
3. [Domain Model](#3-domain-model)
4. [API Design](#4-api-design)
5. [WebSocket Handler](#5-websocket-handler)
6. [Room Manager](#6-room-manager)
7. [Presence System](#7-presence-system)
8. [Rate Limiting](#8-rate-limiting)
9. [Statistics Calculation](#9-statistics-calculation)
10. [Concurrency](#10-concurrency)
11. [Deployment](#11-deployment)
12. [Error Handling](#12-error-handling)

---

## 1. Technology Choice

### Decision: Go (stdlib `net/http` + `nhooyr.io/websocket`)

### Justification

| Criterion | Go | Node.js (Fastify + ws) | Rust (Axum + Tungstenite) |
|---|---|---|---|
| **Single binary** | `go build` produces one static binary. Zero runtime dependencies. | Requires Node.js runtime, `node_modules`, a bundler, or `pkg`/`sea` with caveats. | Single binary via `cargo build --release`. Excellent. |
| **KISS** | Simple language, small stdlib surface, easy to read. | Familiar, fast iteration, but async callback patterns add accidental complexity for WebSocket state management. | Steep learning curve. Borrow checker fights concurrent mutable state (rooms). Lifetimes add ceremony for no business value here. |
| **Concurrency model** | Goroutines + channels are a natural fit for per-room fan-out and heartbeat loops. | Single-threaded event loop works but requires discipline for CPU-bound stats or many timers. | Tokio is powerful but overkill. `Arc<Mutex<>>` everywhere for shared room state. |
| **WebSocket support** | `nhooyr.io/websocket` -- production-quality, context-aware, supports compression, maintained. Lighter than gorilla (which is archived). | `ws` is mature and fast. | `tokio-tungstenite` is solid. |
| **Performance** | More than sufficient. 100 rooms x 20 users = 2,000 connections. Go handles 100K+ trivially. | Sufficient but higher memory per connection due to V8 overhead. | Best raw performance, but irrelevant at this scale. |
| **Deployment simplicity** | `COPY binary` in a `FROM scratch` Dockerfile. ~10MB image. | Multi-stage build, still needs Node runtime layer. ~100MB+ image. | Similar to Go. ~5-10MB image from scratch. |

**Bottom line:** Go gives us the best balance of simplicity, single-binary deployment, and natural concurrency primitives for a WebSocket-heavy application.

### Key Dependencies

| Dependency | Purpose | Version Policy |
|---|---|---|
| `nhooyr.io/websocket` | WebSocket server with context support | Latest stable |
| Go stdlib `net/http` | HTTP server, routing, static file serving | Go 1.22+ (for enhanced ServeMux routing) |

**One external dependency.** Room ID generation uses `crypto/rand` + `encoding/hex` (stdlib). Rate limiting is hand-rolled (~50 lines). No frameworks, no ORMs, no external databases.

---

## 2. Project Structure

```
om-scrum-poker/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: config, wiring, signal handling, graceful shutdown
├── internal/
│   ├── domain/
│   │   ├── room.go              # Room, Participant, Vote, Phase types + methods
│   │   ├── room_test.go         # Pure logic tests (vote, reveal, new round, presence)
│   │   ├── stats.go             # Statistics calculation (average, median, consensus)
│   │   └── stats_test.go        # Stats edge cases
│   └── server/
│       ├── handler.go           # HTTP route registration, SPA fallback, health check
│       ├── ws.go                # WebSocket upgrade, connection lifecycle, read/write pumps
│       ├── client.go            # Client struct, send buffer, per-client goroutines
│       ├── room_manager.go      # In-memory room store, GC loop, broadcast
│       ├── room_manager_test.go # Room manager integration tests
│       ├── ratelimit.go         # Token-bucket rate limiter per IP (~50 lines)
│       ├── ratelimit_test.go
│       └── events.go            # Event type constants, message marshaling helpers
├── web/
│   └── static/                  # Frontend build output (served by Go via embed)
├── go.mod
├── go.sum
├── Dockerfile
└── Makefile
```

### Package Responsibilities

| Package | Responsibility | Depends On |
|---|---|---|
| `cmd/server` | Parse config (env vars), wire domain + server, start HTTP, handle signals | `domain`, `server` |
| `internal/domain` | Pure types and business logic. Room, Participant, Vote, Phase, Stats. No I/O, no dependencies, no concurrency primitives. | Nothing |
| `internal/server` | HTTP handlers, WebSocket handler, room manager (in-memory store + client tracking + broadcast), rate limiter, GC, event marshaling. All I/O and concurrency lives here. | `domain` |

The dependency graph is strictly unidirectional: `cmd/server` -> `server` -> `domain`. No cycles. No interfaces with single implementations. No DTOs. No service layer.

**Why 2 packages, not 4:**
- The `domain` package is pure logic -- easy to test in isolation with table-driven tests, no mocks needed.
- The `server` package is everything else. With ~700 lines across 6 files, splitting further into `handler`/`infra`/`app` packages creates import ceremony for zero benefit. When you have one entity, one store, and no external dependencies, a service layer is indirection without value.

---

## 3. Domain Model

### 3.1 Core Types

```go
// internal/domain/room.go

package domain

import (
    "crypto/rand"
    "encoding/hex"
    "time"
)

// Phase represents the current state of a voting round.
type Phase string

const (
    PhaseVoting   Phase = "voting"
    PhaseReveal Phase = "reveal"
)

// VoteValue represents a participant's vote.
// Valid values: "?", "0", "0.5", "1", "2", "3", "5", "8", "13", "20", "40", "100"
// Empty string means "has not voted".
type VoteValue string

// ValidVotes is the set of allowed vote values.
var ValidVotes = map[VoteValue]bool{
    "?": true, "0": true, "0.5": true, "1": true, "2": true, "3": true,
    "5": true, "8": true, "13": true, "20": true, "40": true, "100": true,
}

// NumericValue returns the float64 value of a vote, and whether it is numeric.
// "?" returns (0, false).
func (v VoteValue) NumericValue() (float64, bool) {
    // Lookup table approach; avoids strconv for known fixed values.
    switch v {
    case "0":
        return 0, true
    case "0.5":
        return 0.5, true
    case "1":
        return 1, true
    case "2":
        return 2, true
    case "3":
        return 3, true
    case "5":
        return 5, true
    case "8":
        return 8, true
    case "13":
        return 13, true
    case "20":
        return 20, true
    case "40":
        return 40, true
    case "100":
        return 100, true
    default:
        return 0, false
    }
}

// PresenceStatus represents a participant's connection/activity state.
type PresenceStatus string

const (
    PresenceActive       PresenceStatus = "active"
    PresenceIdle         PresenceStatus = "idle"
    PresenceDisconnected PresenceStatus = "disconnected"
)

// Participant represents one person in a room.
type Participant struct {
    SessionID   string         `json:"sessionId"`
    Name        string         `json:"userName"`
    Status      PresenceStatus `json:"status"`
    Vote        VoteValue      `json:"-"`                    // Never serialized directly; controlled per-phase
    ConnectedAt time.Time      `json:"connectedAt"`
    LastSeen    time.Time      `json:"-"`                    // Internal: last heartbeat/activity
}

// HasVoted returns whether the participant has cast a vote in the current round.
func (p *Participant) HasVoted() bool {
    return p.Vote != ""
}

// MarkActive sets the participant to active and updates LastSeen.
func (p *Participant) MarkActive() {
    p.Status = PresenceActive
    p.LastSeen = time.Now()
}

// MarkIdle sets the participant to idle.
func (p *Participant) MarkIdle() {
    p.Status = PresenceIdle
}

// MarkDisconnected sets the participant to disconnected.
func (p *Participant) MarkDisconnected() {
    p.Status = PresenceDisconnected
}

// IsDisconnected returns whether the participant is disconnected.
func (p *Participant) IsDisconnected() bool {
    return p.Status == PresenceDisconnected
}

// Room represents a scrum poker room.
type Room struct {
    ID           string                  `json:"id"`           // Full slug: "sprint-42-a1b2c3d4e5f6"
    Name         string                  `json:"name"`         // Display name: "Sprint 42"
    Phase        Phase                   `json:"phase"`
    Participants map[string]*Participant `json:"-"`            // Keyed by sessionId; serialized manually per phase
    CreatedAt    time.Time               `json:"createdAt"`
    LastActivity time.Time               `json:"-"`            // Updated on any state-mutating event
}

// GenerateRoomID creates a room ID by appending a 12-char hex suffix to a slug.
// Uses crypto/rand for 48 bits of entropy.
func GenerateRoomID(slug string) (string, error) {
    b := make([]byte, 6) // 6 bytes = 48 bits = 12 hex chars
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return slug + "-" + hex.EncodeToString(b), nil
}
```

### 3.2 Room Methods (Pure Logic)

```go
// NewRoom creates a new room with the given ID and display name.
func NewRoom(id, name string) *Room {
    now := time.Now()
    return &Room{
        ID:           id,
        Name:         name,
        Phase:        PhaseVoting,
        Participants: make(map[string]*Participant),
        CreatedAt:    now,
        LastActivity: now,
    }
}

// AddParticipant adds a new participant or restores a disconnected one.
// Returns the participant and whether this was a reconnection.
func (r *Room) AddParticipant(sessionID, name string) (*Participant, bool) {
    r.LastActivity = time.Now()
    if p, exists := r.Participants[sessionID]; exists {
        // Reconnection: restore participant, update name if changed
        p.Name = name
        p.MarkActive()
        return p, true
    }
    p := &Participant{
        SessionID:   sessionID,
        Name:        name,
        Status:      PresenceActive,
        ConnectedAt: time.Now(),
        LastSeen:    time.Now(),
    }
    r.Participants[sessionID] = p
    return p, false
}

// RemoveParticipant removes a participant entirely (used for explicit "leave room").
func (r *Room) RemoveParticipant(sessionID string) {
    delete(r.Participants, sessionID)
    r.LastActivity = time.Now()
}

// SetVote sets a participant's vote. Returns an error string if invalid.
func (r *Room) SetVote(sessionID string, value VoteValue) string {
    if r.Phase != PhaseVoting {
        return "PHASE_MISMATCH"
    }
    p, ok := r.Participants[sessionID]
    if !ok {
        return "PARTICIPANT_NOT_FOUND"
    }
    if value != "" && !ValidVotes[value] {
        return "INVALID_VOTE"
    }
    p.Vote = value
    r.LastActivity = time.Now()
    return ""
}

// Reveal transitions to revealed phase. Returns error string if already revealed.
func (r *Room) Reveal() string {
    if r.Phase != PhaseVoting {
        return "PHASE_MISMATCH"
    }
    r.Phase = PhaseReveal
    r.LastActivity = time.Now()
    return ""
}

// NewRound clears all votes and resets to voting phase.
func (r *Room) NewRound() {
    for _, p := range r.Participants {
        p.Vote = ""
    }
    r.Phase = PhaseVoting
    r.LastActivity = time.Now()
}

// Clear removes all participants and resets the room.
func (r *Room) Clear() {
    r.Participants = make(map[string]*Participant)
    r.Phase = PhaseVoting
    r.LastActivity = time.Now()
}

// UpdateParticipantName changes a participant's display name.
func (r *Room) UpdateParticipantName(sessionID, name string) string {
    p, ok := r.Participants[sessionID]
    if !ok {
        return "PARTICIPANT_NOT_FOUND"
    }
    p.Name = name
    r.LastActivity = time.Now()
    return ""
}

// CheckDisconnectedParticipants scans participants and marks any
// whose LastSeen exceeds the given threshold as disconnected.
// Returns a list of sessionIDs that transitioned to disconnected.
func (r *Room) CheckDisconnectedParticipants(threshold time.Duration) []string {
    now := time.Now()
    var transitioned []string
    for id, p := range r.Participants {
        if p.Status != PresenceDisconnected && now.Sub(p.LastSeen) > threshold {
            p.MarkDisconnected()
            transitioned = append(transitioned, id)
        }
    }
    return transitioned
}

// ActiveParticipantCount returns the number of non-disconnected participants.
func (r *Room) ActiveParticipantCount() int {
    count := 0
    for _, p := range r.Participants {
        if !p.IsDisconnected() {
            count++
        }
    }
    return count
}
```

### 3.3 Design Decisions

- **Room ID is the slug.** The URL path `/room/sprint-42-a1b2c3d4e5f6` directly maps to room ID `sprint-42-a1b2c3d4e5f6`. No separate slug-to-ID lookup.
- **Room ID suffix is 12-char hex (48 bits).** This provides ~281 trillion possible suffixes per slug, compared to ~4.3 billion with 8-char (32 bits). At 20 requests/minute rate limit, brute-forcing takes ~26 billion years.
- **Room display name provided by first joiner.** The `join` message includes a `roomName` field. The first joiner's value is stored; subsequent joiners' values are ignored. This avoids lossy slug-to-name reverse-engineering.
- **Participants are keyed by `sessionId`**, not by WebSocket connection. This allows a participant to disconnect and reconnect while preserving their vote and identity.
- **`Vote` is a string, not a float.** This avoids floating-point representation issues with `0.5` and naturally handles `"?"`. Conversion to numeric happens only during statistics calculation.
- **`LastSeen` is not serialized.** It is internal server state for presence tracking, not sent to clients.
- **No `RoundNumber` field.** A "new round" simply clears all votes and sets phase to `voting`. There is no history to track, so a counter adds no value.
- **Presence lives on Participant, not in a separate tracker.** The `Participant` struct owns its `Status` and `LastSeen`. The `Room` has a method (`CheckDisconnectedParticipants`) that scans for stale connections. No parallel data structure, no synchronization between two representations of the same state.

### 3.4 Room State Invariants

These invariants must hold at all times:

1. If `Phase == PhaseVoting`, no client receives any vote values (votes are secret).
2. If `Phase == PhaseReveal`, all clients receive all vote values (including `""` for non-voters).
3. A participant's vote is preserved across disconnect/reconnect within the same round.
4. `LastActivity` is updated on every state-mutating event (join, vote, reveal, new round, clear).

---

## 4. API Design

### 4.1 REST Endpoints

#### `GET /health`

Returns server health with room and connection metrics.

**Response (200 OK):**
```json
{
    "status": "ok",
    "rooms": 12,
    "connections": 47,
    "uptime": "3h24m12s"
}
```

Useful for Docker health checks, load balancer probes, and operational monitoring.

#### `GET /api/rooms/{roomId}`

Check if a room exists before opening WebSocket.

**Response (200 OK):**
```json
{
    "exists": true
}
```

**Error responses:**
- `404 Not Found` -- room does not exist or has been garbage-collected.

Note: This endpoint returns only existence confirmation. Full room state is delivered over WebSocket.

#### Room Creation: No REST Endpoint

Rooms are **not** created via a REST API. Instead, the client generates the room slug + ID locally (using `crypto.getRandomValues(new Uint8Array(6))` hex-encoded for the 12-char suffix) and navigates directly to `/room/{slug}-{id}`. The room is created implicitly when the first WebSocket `join` event arrives. This eliminates an HTTP round-trip and aligns with the in-memory-only philosophy: a room exists only when someone is connected to it.

The `GET /api/rooms/{roomId}` endpoint can be used as a lightweight existence check if the frontend wants to distinguish "room not found" from "room exists, connect now" before opening the WebSocket.

### 4.2 WebSocket Endpoint

> **See Architecture Doc v3.0, Section 4 for the canonical WebSocket protocol.**
> This section describes the server-side implementation of that protocol, not the wire format.

#### `GET /ws/room/{roomId}` (upgrade to WebSocket)

**Connection flow:**
1. Client sends HTTP upgrade request to `/ws/room/{roomId}`.
2. Server checks rate limits for the connecting IP. If exceeded, rejects with HTTP 429.
3. Server upgrades connection.
4. Client must send a `join` event within 5 seconds, or the server closes the connection with close code 4001.
5. Server validates the `join` payload (`sessionId`, `userName`, `roomName`).
6. If the room ID does not exist, the server creates it. The room display name comes from the `roomName` field of the `join` message.
7. Server adds or restores the participant in the room.
8. Server sends `room_state` (full snapshot) to the joining client.
9. Server broadcasts `participant_joined` (with `sessionId`, `userName`, `status`) to all other clients.

### 4.3 Static File Serving / SPA Fallback

Any path that does not match `/api/*` or `/ws/*` and does not correspond to a static file serves `index.html`. This allows the frontend router to handle `/room/{slug}` paths client-side.

```go
// Serve frontend for all non-API, non-WS routes (SPA fallback)
mux.Handle("/", spaHandler(staticFiles))
```

The `spaHandler` first attempts to serve the requested file from the embedded filesystem. If the file does not exist (e.g., `/room/sprint-42-abc123`), it serves `index.html` instead.

---

## 5. WebSocket Handler

> **Protocol reference:** See Architecture Doc v3.0, Section 4 for the canonical WebSocket protocol,
> including the full event table, message envelope format (`{ type, payload }`), and all payload structures.
> This section covers Go implementation details only.

### 5.1 Connection Lifecycle

Each WebSocket connection is managed by a `Client` struct with two goroutines: a read pump and a write pump.

```
Client connects to /ws/room/{roomId}
    |
    v
[Rate limit check] --> 429 if exceeded
    |
    v
[Upgrade HTTP -> WebSocket]
    |
    v
[Start read pump goroutine]  <-- reads messages from client, dispatches to room
[Start write pump goroutine]  <-- reads from send channel, writes to client
    |
    v
[Wait for "join" event within 5s timeout]
    |
    +-- timeout --> close with code 4001 ("join timeout")
    |
    +-- valid join --> create-or-get room, register client, send room_state
    |
    v
[Normal operation: read/write loop]
    |
    +-- "leave" event --> remove participant immediately, broadcast, close 1000
    |
    v
[Connection closes (client disconnect, error, or server shutdown)]
    |
    v
[Unregister client from room]
[Mark participant as disconnected, broadcast presence_changed]
[Start disconnect grace period (10s)]
    |
    +-- client reconnects with same sessionId within 10s --> cancel timer, restore
    |
    +-- grace period expires --> broadcast participant_left
```

### 5.2 "Leave Room" vs "Disconnect" Semantics

These are two distinct behaviors:

| Scenario | Trigger | Server Behavior |
|---|---|---|
| **Leave room** | Client sends `leave` event (user clicks "Leave room" in settings) | Immediately remove participant from room. Broadcast `participant_left`. Close WebSocket with code 1000. No grace period. Vote is discarded. |
| **Disconnect** | WebSocket closes unexpectedly (tab close, network loss, browser crash) | Mark participant as `disconnected`. Broadcast `presence_changed`. Start 10-second grace period. If no reconnection, broadcast `participant_left`. Vote is preserved during grace period (and beyond -- disconnected participants stay in the room with their vote until "Clear Room" or room GC). |
| **Reconnect** | Client sends `join` with an existing `sessionId` after a disconnect | Cancel grace timer if running. Restore participant to `active`. Preserve existing vote. Broadcast `participant_joined` (with `sessionId`, `userName`, `status`). Send fresh `room_state` to reconnecting client. |

**Key distinction:** A deliberate "Leave room" is permanent and immediate. An unexpected disconnect is temporary -- the participant's card stays visible (dimmed) and their vote is preserved, per UX doc Section 6.4.

### 5.3 Client Struct

```go
// internal/server/client.go

type Client struct {
    conn      *websocket.Conn
    roomID    string
    sessionID string
    send      chan []byte     // Buffered channel for outbound messages
    manager   *RoomManager
}

const (
    sendBufferSize    = 32           // Max queued outbound messages per client
    writeTimeout      = 10 * time.Second
    readTimeout       = 60 * time.Second  // Must be > ping interval (5s)
    maxMessageSize    = 1024              // 1 KB max inbound message
    joinTimeout       = 5 * time.Second
    pingInterval      = 5 * time.Second   // Protocol-level WebSocket pings
    disconnectThreshold = 10 * time.Second
)
```

### 5.4 Read Pump

```
loop:
    set read deadline (readTimeout)
    read message from WebSocket (max 1 KB)
    |
    +-- error (timeout, close, etc.) --> break loop, trigger cleanup
    |
    +-- valid message --> parse JSON envelope { type, payload }
        |
        +-- parse error --> send error event { code: "invalid_message", message: "..." }
        |
        +-- valid event --> dispatch to handler by type:
            |
            +-- "vote"        --> room.SetVote(sessionId, value)
            |                     if value == "": broadcast vote_retracted
            |                     else: broadcast vote_cast
            +-- "reveal"      --> room.Reveal(); broadcast votes_revealed with stats
            +-- "new_round"   --> room.NewRound(); broadcast round_reset
            +-- "clear_room"  --> room.Clear(); broadcast room_cleared; close all connections
            +-- "update_name" --> room.UpdateParticipantName(); broadcast name_updated
            +-- "presence"    --> update participant status; broadcast presence_changed
            +-- "leave"       --> room.RemoveParticipant(); broadcast participant_left; close 1000
            +-- unknown       --> send error event { code: "invalid_message", message: "..." }
```

On any domain error (e.g., `SetVote` returns `"PHASE_MISMATCH"`), the server sends an error event to the client:

```json
{ "type": "error", "payload": { "code": "phase_mismatch", "message": "Cannot vote during reveal phase" } }
```

The connection remains open. See Architecture Doc v3.0, Section 4.5 for the error code list.

### 5.5 Write Pump

```
start protocol-level ping ticker (5s interval)
loop:
    select:
        case message from send channel:
            set write deadline (writeTimeout)
            write message to WebSocket
            |
            +-- error --> break loop, trigger cleanup
        case ping tick:
            send WebSocket protocol-level ping (via nhooyr.io/websocket)
            |
            +-- error --> break loop, trigger cleanup
```

**Heartbeat strategy:** Protocol-level WebSocket pings only (5-second interval). The `nhooyr.io/websocket` library handles ping/pong natively. No application-level `heartbeat` messages in the protocol. The server uses the pong responses to update `Participant.LastSeen`. If a user reports proxy issues with protocol pings being stripped, application-level pings can be added later.

### 5.6 Broadcast Mechanism

Each room maintains a set of active `Client` references via the `RoomManager`. Broadcasting iterates over clients and sends to each client's `send` channel. If the channel is full (client is slow), the client is disconnected.

```go
func (rm *RoomManager) Broadcast(roomID string, msg []byte, excludeSessionID string) {
    rm.mu.RLock()
    mr, ok := rm.rooms[roomID]
    rm.mu.RUnlock()
    if !ok {
        return
    }

    mr.mu.Lock()
    defer mr.mu.Unlock()
    for sessionID, client := range mr.clients {
        if sessionID == excludeSessionID {
            continue
        }
        select {
        case client.send <- msg:
            // Sent successfully
        default:
            // Client's send buffer is full; disconnect them.
            close(client.send)
            delete(mr.clients, sessionID)
        }
    }
}
```

This is a non-blocking fan-out. The `send` channel buffer (32 messages) absorbs transient slowness. If a client falls permanently behind, they are dropped -- the client-side reconnection logic handles recovery.

### 5.7 Message Marshaling

The `events.go` file contains all message type constants and marshaling helpers. Every outbound message uses the `{ type, payload }` envelope format.

```go
// internal/server/events.go

// Event type constants -- these MUST match Architecture Doc v3.0, Section 4.3
const (
    // Client -> Server
    EventJoin       = "join"
    EventVote       = "vote"
    EventReveal     = "reveal"
    EventNewRound   = "new_round"
    EventClearRoom  = "clear_room"
    EventUpdateName = "update_name"
    EventPresence   = "presence"
    EventLeave      = "leave"

    // Server -> Client
    EventRoomState        = "room_state"
    EventParticipantJoined = "participant_joined"
    EventParticipantLeft   = "participant_left"
    EventVoteCast         = "vote_cast"
    EventVoteRetracted    = "vote_retracted"
    EventVotesRevealed    = "votes_revealed"
    EventRoundReset       = "round_reset"
    EventRoomCleared      = "room_cleared"
    EventPresenceChanged  = "presence_changed"
    EventNameUpdated      = "name_updated"
    EventError            = "error"
)

// Envelope is the standard message wrapper.
type Envelope struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

// NewMessage creates a JSON-encoded message with the envelope format.
func NewMessage(eventType string, payload interface{}) ([]byte, error) {
    return json.Marshal(Envelope{Type: eventType, Payload: payload})
}
```

---

## 6. Room Manager

### 6.1 Structure

```go
// internal/server/room_manager.go

type RoomManager struct {
    mu    sync.RWMutex
    rooms map[string]*ManagedRoom  // Keyed by room ID (slug)
}

type ManagedRoom struct {
    Room       *domain.Room
    clients    map[string]*Client  // Active WebSocket connections by sessionId
    graceTimers map[string]*time.Timer // Disconnect grace timers by sessionId
    mu         sync.Mutex             // Per-room lock for mutations
}
```

### 6.2 Operations

| Operation | Lock | Description |
|---|---|---|
| `GetOrCreate(id, name) -> ManagedRoom` | Manager write lock (if creating) or read lock (if exists) | Lookup by ID; create if not found. The `name` parameter comes from the `join` message's `roomName` field. |
| `Get(id) -> ManagedRoom` | Manager read lock | Lookup by ID, return nil if not found |
| `Delete(id)` | Manager write lock | Remove room from map, close all client connections |
| `RegisterClient(roomID, client)` | Per-room lock | Add client to room's client set; cancel grace timer if reconnecting |
| `UnregisterClient(roomID, sessionID)` | Per-room lock | Remove client from room's client set; start grace timer |
| `Broadcast(roomID, msg, excludeSessionID)` | Per-room lock | Send message to all clients in a room |
| `RoomCount() int` | Manager read lock | Return number of rooms (for health check) |
| `ConnectionCount() int` | Manager read lock + per-room locks | Return total connections across all rooms (for health check) |

### 6.3 Two-Level Locking

The `RoomManager` has a `sync.RWMutex` that protects the `rooms` map itself (creation/deletion/lookup of rooms). Each `ManagedRoom` has its own `sync.Mutex` that protects mutations within a single room (adding/removing participants, changing votes, etc.).

This means:
- Room creation/deletion takes a write lock on the manager but does not block operations in other rooms.
- Operations within a room (voting, revealing, etc.) only lock that specific room.
- Reading the room map (e.g., checking if a room exists) uses a read lock and does not block other readers.

**Lock ordering:** Always acquire Manager lock first, then Room lock. Never acquire the manager lock while holding a room lock. This prevents deadlocks.

### 6.4 Garbage Collection

A background goroutine runs every 10 minutes and removes stale rooms.

```go
func (rm *RoomManager) gc(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            rm.mu.Lock()
            now := time.Now()
            for id, mr := range rm.rooms {
                mr.mu.Lock()
                hasClients := len(mr.clients) > 0
                stale := now.Sub(mr.Room.LastActivity) > 24*time.Hour
                mr.mu.Unlock()
                if !hasClients && stale {
                    delete(rm.rooms, id)
                }
            }
            rm.mu.Unlock()
        }
    }
}
```

**GC rules:**
- A room is eligible for GC if it has zero connected clients AND `LastActivity` is older than 24 hours.
- Rooms with active connections are never GC'd, regardless of age.
- `LastActivity` is updated on every state-mutating event (join, vote, reveal, new round).

---

## 7. Presence System

### 7.1 Overview

Presence tracking is built directly into the `Participant` struct and `Room` methods. There is no separate tracker package or parallel data structure.

Three states per participant, as specified in UX doc Section 6:

| State | Meaning | Transition To |
|---|---|---|
| **Active** | Tab focused, user interacting | -> Idle (client sends `presence` with `"idle"`) |
| **Idle** | Tab open but unfocused/inactive | -> Active (client sends `presence` with `"active"`) |
| **Disconnected** | WebSocket connection lost | -> Active (reconnection with same `sessionId`) |

### 7.2 Client-Side Responsibility

The client detects active/idle transitions and reports them via `presence` events:
- On `visibilitychange` (tab hidden): send `{ "type": "presence", "payload": { "status": "idle" } }`.
- On `visibilitychange` (tab visible) or any user interaction after 2min idle: send `{ "type": "presence", "payload": { "status": "active" } }`.

The client never sends `"disconnected"` -- that is a server-side detection only.

### 7.3 Server-Side Responsibility

1. **LastSeen tracking.** The server updates `Participant.LastSeen` on every received message (any type) and on WebSocket pong responses. This is the single source of truth for connection liveness.

2. **Disconnect detection.** A single goroutine per `RoomManager` (not per room) runs a ticker every 5 seconds. It iterates all rooms and calls `room.CheckDisconnectedParticipants(10 * time.Second)`. Any participant whose `LastSeen` exceeds 10 seconds and is not already disconnected transitions to `disconnected`, and a `presence_changed` event is broadcast.

3. **Disconnect grace period.** When a WebSocket connection closes (not via explicit `leave`):
   - The participant is marked `disconnected`.
   - A `presence_changed` event is broadcast.
   - A 10-second grace timer starts. If the participant reconnects (same `sessionId`) within this window, the timer is cancelled and the participant is restored to `active`.
   - If the timer expires, a `participant_left` event is broadcast. However, the participant remains in the `Room.Participants` map as `disconnected` -- their card stays dimmed but visible, and their vote is preserved. They are NOT removed from the participant list, per UX doc Section 6.4. Disconnected participants persist until "Clear Room" or room GC.

4. **Presence broadcast.** Any status change triggers a `presence_changed` event to all clients in the room.

### 7.4 Why No Separate Tracker

The v1.0 plan had a `presence.Tracker` with its own `sessions map[string]*sessionState` that duplicated `Participant.Status` and `Participant.LastSeen`. This created two copies of the same data that must be kept in sync -- a classic consistency bug. In v2.0, presence state lives exclusively on the `Participant` struct, and presence logic is expressed as methods on `Room`. One data structure, one source of truth.

---

## 8. Rate Limiting

### 8.1 Strategy: Token Bucket per IP

A simple in-memory token bucket rate limiter keyed by IP address. No external dependencies. ~50 lines of code in `internal/server/ratelimit.go`.

```go
// internal/server/ratelimit.go

type RateLimiter struct {
    mu      sync.Mutex
    buckets map[string]*bucket
}

type bucket struct {
    tokens    float64
    lastTime  time.Time
    maxTokens float64
    refillRate float64  // tokens per second
}
```

### 8.2 Rate Limits

| Resource | Limit | Bucket Config | Rationale |
|---|---|---|---|
| **WebSocket connections** (`/ws/room/{roomId}`) | 20 connections per minute per IP | Max: 20, Refill: 20/60s | Allows a user to reconnect multiple times (exponential backoff) but prevents abuse. Multiple team members behind NAT are covered by the generous limit. |

Room creation is no longer a separate REST endpoint, so the WebSocket rate limit covers it implicitly (creating a room requires a WebSocket connection + join).

### 8.3 Room ID Enumeration Prevention

The room URL format is `/room/{slug}-{12-hex-chars}`. The 12-character hex suffix provides:

- **16^12 = ~281 trillion possible suffixes** per slug. Even knowing the slug (e.g., "sprint-42"), an attacker must guess the suffix.
- At 20 requests/minute rate limit, brute-forcing takes ~281 trillion / 20 = ~14 trillion minutes = **~26 billion years**.
- The slug portion adds further entropy since it must also match.
- Rooms are ephemeral (24h TTL), shrinking the attack window.

**Additional measures:**
- `GET /api/rooms/{roomId}` returns only `{ "exists": true }` (no participant details, no votes, no room name). An attacker who guesses a room ID learns only that it exists.
- No room listing endpoint. There is no way to enumerate rooms.

### 8.4 Limiter GC

The limiter periodically (every 5 minutes) removes buckets that have been idle for more than 10 minutes to prevent unbounded memory growth.

### 8.5 HTTP Middleware

Rate limiting is applied as HTTP middleware wrapping the WebSocket upgrade handler:

```go
func (rl *RateLimiter) Middleware(maxTokens float64, refillRate float64, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := extractIP(r) // Handles X-Forwarded-For when TRUST_PROXY is true
        if !rl.Allow(ip, maxTokens, refillRate) {
            http.Error(w, `{"error":{"code":"rate_limited","message":"Too many requests"}}`, http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

IP extraction: uses `X-Forwarded-For` header (first entry) if `TRUST_PROXY` is true, otherwise `r.RemoteAddr`.

---

## 9. Statistics Calculation

### 9.1 When Calculated

Statistics are calculated once when `reveal` is triggered, included in the `votes_revealed` payload, and sent to all clients. They are also included in the `room_state` snapshot during the reveal phase (so late joiners see stats). Stats are not recalculated dynamically.

### 9.2 Implementation

> **See Architecture Doc v3.0, Section 4.5 for the canonical `votes_revealed` payload structure.**

```go
// internal/domain/stats.go

// RoundResult holds the reveal-phase data sent in votes_revealed and room_state.
type RoundResult struct {
    Votes        []VoteEntry `json:"votes"`
    Average      *float64    `json:"average"`       // nil if no numeric votes
    Median       *float64    `json:"median"`        // nil if no numeric votes
    UncertainCount int       `json:"uncertainCount"` // Number of "?" votes
    TotalVoters    int       `json:"totalVoters"`   // Total participants who voted (including "?")
    HasConsensus   bool      `json:"hasConsensus"`
    Spread         *[2]float64 `json:"spread"`      // [min, max], null if < 2 numeric votes
}

type VoteEntry struct {
    SessionID string `json:"sessionId"`
    Name      string `json:"userName"`
    Value     string `json:"value"`
}

// CalculateResult computes statistics from a room's participants.
func CalculateResult(participants map[string]*Participant) *RoundResult { ... }
```

### 9.3 Calculation Rules

**Input:** all participants' votes in the room.

1. **Partition votes** into numeric votes and "?" votes. Empty votes (participant did not vote) are excluded entirely.

2. **Average:** arithmetic mean of numeric vote values. `nil` if no numeric votes.

3. **Median:**
   - Sort numeric values.
   - If odd count: middle value.
   - If even count: average of two middle values.
   - `nil` if no numeric votes.

4. **Consensus detection:**
   - `true` if all numeric votes are identical AND there is at least one numeric vote.
   - "?" votes do not break consensus (they are simply excluded).
   - If only "?" votes exist, `consensus` is `false`.

5. **Spread detection** (for UX doc Section 10.3):
   - Compute `min` and `max` of numeric votes.
   - `spread` is `[min, max]`, or `null` if fewer than 2 numeric votes (including when all numeric votes are the same value, i.e., consensus).

### 9.4 Example Calculations

| Votes | Average | Median | Consensus | Spread |
|---|---|---|---|---|
| 5, 5, 5 | 5.0 | 5.0 | true | null (consensus) |
| 3, 5, 8 | 5.33 | 5.0 | false | [3, 8] |
| 5, 5, ? | 5.0 | 5.0 | true | null (consensus) |
| ?, ?, ? | null | null | false | null |
| (no votes) | null | null | false | null |

---

## 10. Concurrency

### 10.1 Concurrency Model

The application uses Go's goroutine-per-connection model with shared state protected by mutexes:

```
                         ┌──────────────────────┐
                         │    Room Manager       │
                         │  sync.RWMutex (map)   │
                         └──────┬───────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              │                 │                  │
        ┌─────┴─────┐    ┌─────┴─────┐     ┌─────┴─────┐
        │  Room A    │    │  Room B    │     │  Room C    │
        │ sync.Mutex │    │ sync.Mutex │     │ sync.Mutex │
        └─────┬──────┘   └─────┬──────┘    └─────┬──────┘
              │                 │                  │
        ┌─────┼─────┐    ┌─────┼─────┐     ┌─────┼─────┐
        │     │     │    │     │     │     │     │     │
       C1    C2    C3   C4    C5    C6    C7    C8    C9
    (goroutine pairs: read pump + write pump per client)
```

### 10.2 Lock Ordering

To prevent deadlocks, locks are always acquired in this order:
1. Manager `RWMutex` (if needed)
2. Room `Mutex`

Never acquire the manager lock while holding a room lock.

### 10.3 Critical Sections

| Operation | Lock(s) Held | Duration |
|---|---|---|
| Create room | Manager write lock | Microseconds (map insert) |
| Lookup room | Manager read lock | Microseconds (map read) |
| Join/leave | Room mutex | Microseconds (map insert/delete + marshal + broadcast) |
| Vote | Room mutex | Microseconds (set value + marshal + broadcast) |
| Reveal | Room mutex | Sub-millisecond (compute stats + marshal + broadcast) |
| GC sweep | Manager write lock | Milliseconds (iterate map, delete stale entries) |

All critical sections are extremely short. No I/O is performed under lock. Broadcast writes to buffered channels (non-blocking), so the lock is not held while waiting for slow clients.

### 10.4 Channel Usage

- **`Client.send chan []byte`** -- buffered (32). Decouples event production (under room lock) from WebSocket writes (in the write pump goroutine). If the channel fills, the client is dropped.
- No other channels needed. The design avoids the "goroutine per room" pattern (a single event loop goroutine that serializes all room operations). While that pattern eliminates mutexes, it adds complexity for questionable benefit at this scale. Mutexes with short critical sections are simpler and sufficient.

### 10.5 Background Goroutines

| Goroutine | Count | Lifecycle | Purpose |
|---|---|---|---|
| Read pump | 1 per client | Connection open -> close | Read and dispatch inbound messages |
| Write pump | 1 per client | Connection open -> close | Write outbound messages + send protocol pings (5s) |
| GC | 1 global | Server start -> shutdown | Remove stale rooms every 10 minutes |
| Presence checker | 1 global | Server start -> shutdown | Check LastSeen every 5s, mark disconnected participants |
| Rate limiter GC | 1 global | Server start -> shutdown | Remove stale rate limit buckets every 5 minutes |

Total goroutines at 2,000 connections: ~4,003 (2 per client + 3 global). Well within Go's capability.

### 10.6 Graceful Shutdown

1. `main.go` listens for `SIGINT`/`SIGTERM`.
2. Cancel the root context.
3. Call `http.Server.Shutdown(ctx)` with a 10-second deadline.
4. This closes the listener (no new connections), then waits for active requests to finish.
5. The GC goroutine, presence checker, and rate limiter GC exit on context cancellation.
6. Active WebSocket connections receive a close frame with code 1001 (Going Away).

---

## 11. Deployment

### 11.1 Single Binary

```makefile
# Makefile

build:
	cd frontend && npm ci && npm run build
	CGO_ENABLED=0 go build -ldflags="-s -w" -o om-scrum-poker ./cmd/server

run: build
	./om-scrum-poker
```

The binary embeds static frontend files using Go's `embed` package:

```go
//go:embed frontend/dist
var staticFiles embed.FS
```

This produces a single binary that serves both the API and the frontend. No separate web server needed.

### 11.2 Configuration

All configuration via environment variables with sensible defaults:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `HOST` | `0.0.0.0` | HTTP listen address |
| `TRUST_PROXY` | `false` | Trust `X-Forwarded-For` for rate limiting IP extraction |

**That is it.** Three variables. All other values (room TTL, GC interval, ping interval, rate limits, buffer sizes) are hardcoded constants with sensible defaults. If a real user requests configurability for a specific value, it can be promoted to an environment variable at that time. Until then, YAGNI.

Hardcoded defaults:
- Room TTL: 24 hours
- GC interval: 10 minutes
- Protocol-level ping interval: 5 seconds
- Disconnect threshold: 10 seconds
- Grace period: 10 seconds
- Max message size: 1 KB
- Send buffer: 32 messages
- WebSocket rate limit: 20/minute per IP

### 11.3 Docker

```dockerfile
# Build stage: frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build stage: Go
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /om-scrum-poker ./cmd/server

# Runtime stage
FROM scratch
COPY --from=builder /om-scrum-poker /om-scrum-poker
EXPOSE 8080
ENTRYPOINT ["/om-scrum-poker"]
```

Final image size: ~10-15MB. No OS, no shell, no attack surface.

---

## 12. Error Handling

### 12.1 HTTP Error Responses

All HTTP errors use a consistent JSON format:

```json
{
    "error": {
        "code": "room_not_found",
        "message": "Room does not exist."
    }
}
```

| HTTP Status | Code | When |
|---|---|---|
| 404 | `room_not_found` | Room ID does not exist (GET /api/rooms/{roomId}) |
| 429 | `rate_limited` | Rate limit exceeded |
| 500 | `internal_error` | Unexpected server error (logged, not exposed) |

### 12.2 WebSocket Error Events

> **See Architecture Doc v3.0, Section 4.5 (error codes) for the canonical error code list.**

For errors that occur during an active WebSocket session, the server sends an `error` event using the standard `{ type, payload }` envelope:

```json
{
    "type": "error",
    "payload": {
        "code": "invalid_vote",
        "message": "Vote value '99' is not valid."
    }
}
```

Error codes used by the backend:

| Error Code | When | Action |
|---|---|---|
| `invalid_vote` | Vote value not in allowed set, or voting during reveal phase | Event rejected, client notified |
| `invalid_message` | Unknown event type or malformed JSON | Event rejected, client notified |
| `invalid_name` | Name is empty or exceeds 30 chars | Event rejected, client notified |
| `room_not_found` | Room does not exist (e.g., after server restart, on reconnect) | Client should redirect to home page |
| `rate_limited` | Too many messages from this connection | Event rejected, client notified |

These are non-fatal. The connection remains open.

### 12.3 WebSocket Close Codes

For terminal errors, the server closes the WebSocket with a specific close code:

| Code | Meaning | When |
|---|---|---|
| 1000 | Normal closure | Clean disconnect (explicit `leave` event) |
| 1001 | Going away | Server shutdown |
| 1008 | Policy violation | Rate limit exceeded after connection |
| 4001 | Join timeout | Client did not send `join` within 5 seconds |
| 4002 | Invalid join | `join` payload failed validation |
| 4003 | Room cleared | Room was cleared by another participant |

Codes 4000-4999 are reserved for application use by the WebSocket RFC.

### 12.4 Server-Side Logging

- All errors are logged with structured fields (room ID, session ID, IP, error).
- Use Go's `slog` package (stdlib, structured logging, zero dependencies).
- Log levels: `INFO` for connections/disconnections/room lifecycle, `WARN` for rate limit hits and validation failures, `ERROR` for unexpected panics or I/O errors.
- No sensitive data in logs (no vote values, no user names -- only IDs).

### 12.5 Panic Recovery

Each WebSocket handler goroutine is wrapped in a `defer recover()` that:
1. Logs the panic with stack trace.
2. Closes the WebSocket connection with code 1011 (Internal Error).
3. Cleans up the client registration.

This prevents a single malformed message from crashing the entire server.

---

## Appendix A: Capacity Estimation

**Target:** 100 concurrent rooms, 20 users each = 2,000 simultaneous WebSocket connections.

| Resource | Estimate |
|---|---|
| Memory per connection | ~10KB (goroutine stacks + send buffer) |
| Memory per room | ~5KB (participant map + metadata) |
| Total connection memory | 2,000 x 10KB = ~20MB |
| Total room memory | 100 x 5KB = ~500KB |
| Protocol-level pings | 2,000 pings/5s = 400 msg/s (protocol-level, minimal overhead) |
| Peak broadcast (all reveal simultaneously) | 100 rooms x 20 participants = 2,000 messages (one-time burst) |

**Total estimated memory:** < 50MB including Go runtime overhead. This runs comfortably on the smallest cloud VM (512MB).

**CPU:** Negligible. JSON marshaling of small payloads and arithmetic for stats. A single core handles this easily.

## Appendix B: Code Volume Estimate

| Component | Estimated Lines |
|---|---|
| `internal/domain/` (Room, Participant, Stats, methods) | ~250 |
| `internal/server/` (handlers, WS, room manager, rate limiter, events) | ~700 |
| `cmd/server/main.go` | ~50 |
| **Total (excluding tests)** | **~1,000** |
| Tests | ~400-500 |
| **Grand total** | **~1,400-1,500** |

This is appropriate for the problem. Two packages, one dependency, one thousand lines.
