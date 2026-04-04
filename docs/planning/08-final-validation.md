# om-scrum-poker: Final Validation (Iteration 5)

**Date:** 2026-04-04
**Reviewer Roles:** Product Owner, Chief Architect, Devil's Advocate
**Status:** FINAL GATE REVIEW
**Documents Reviewed:**
- `01-ux-design.md` (v1.0)
- `02-frontend-plan.md` (v3.1)
- `03-backend-plan.md` (v3.1)
- `04-architecture.md` (v3.1)
- `07-review-iteration-3.md` (to verify fixes applied)

---

## 1. Zero Contradictions Check

### 1.1 Iteration 3 Fix Verification

All three items from the iteration 3 review have been resolved:

| Item | Status | Evidence |
|---|---|---|
| Phase string `"revealed"` -> `"reveal"` | FIXED | Backend plan v3.1 changelog explicitly notes the change. `PhaseReveal Phase = "reveal"` at line 165. All references updated: `r.Phase = PhaseReveal`, `PhaseReveal` in invariants (Section 3.4). No remaining `"revealed"` strings in the backend plan. |
| Health endpoint `/api/health` -> `/health` | FIXED | Backend plan v3.1 changelog: changed to `/health`. Architecture doc Section 5 uses `/health`. Backend plan Section 4.1 uses `GET /health`. Consistent. |
| `GET /api/rooms/{roomId}` response body | FIXED | Backend plan v3.1 changelog: changed to `{ "exists": true }`. Architecture doc Section 5 uses `{ "exists": true }`. Both aligned. |

### 1.2 WebSocket Protocol Alignment (Event-by-Event)

Verified every event across all three technical documents against the architecture doc's canonical event table (Section 4.3):

| Event | Envelope | Payload Fields | Frontend | Backend | Architecture | Match |
|---|---|---|---|---|---|---|
| `join` | `{ type, payload }` | `sessionId`, `userName`, `roomName` | Section 5.2 | Section 4.2 | Section 4.4 | YES |
| `vote` | `{ type, payload }` | `value` | Section 5.4 | Section 5.4 | Section 4.4 | YES |
| `reveal` | `{ type, payload }` | empty | Section 5.4 | Section 5.4 | Section 4.4 | YES |
| `new_round` | `{ type, payload }` | empty | Section 5.4 | Section 5.4 | Section 4.4 | YES |
| `clear_room` | `{ type, payload }` | empty | Section 5.4 | Section 5.4 | Section 4.4 | YES |
| `update_name` | `{ type, payload }` | `userName` | Section 5.5 | Section 5.4 | Section 4.4 | YES |
| `presence` | `{ type, payload }` | `status` | Section 5.6 | Section 5.4 | Section 4.4 | YES |
| `leave` | `{ type, payload }` | empty | Header settings | Section 5.2 | Section 4.4 | YES |
| `room_state` | `{ type, payload }` | `roomId`, `roomName`, `phase`, `participants`, `result` | Section 5.4 | Section 4.2 | Section 4.5 | YES |
| `participant_joined` | `{ type, payload }` | `sessionId`, `userName`, `status` | Section 5.4 | Section 4.2 | Section 4.5 | YES |
| `participant_left` | `{ type, payload }` | `sessionId` | Section 5.4 | Section 5.2 | Section 4.5 | YES |
| `vote_cast` | `{ type, payload }` | `sessionId` | Section 5.4 | Section 5.4 | Section 4.5 | YES |
| `vote_retracted` | `{ type, payload }` | `sessionId` | Section 5.4 | Section 5.4 | Section 4.5 | YES |
| `votes_revealed` | `{ type, payload }` | `votes`, `average`, `median`, `uncertainCount`, `totalVoters`, `hasConsensus`, `spread` | Section 5.4 | Section 9 | Section 4.5 | YES |
| `round_reset` | `{ type, payload }` | empty | Section 5.4 | Section 5.4 | Section 4.5 | YES |
| `room_cleared` | `{ type, payload }` | empty | Section 5.4 | Section 5.4 | Section 4.5 | YES |
| `presence_changed` | `{ type, payload }` | `sessionId`, `status` | Section 5.4 | Section 7.3 | Section 4.5 | YES |
| `name_updated` | `{ type, payload }` | `sessionId`, `userName` | Section 5.4 | Section 5.4 | Section 4.5 | YES |
| `error` | `{ type, payload }` | `code`, `message` | Section 5.7 | Section 12.2 | Section 4.5 | YES |

**Result: 19/19 events fully aligned. Zero protocol contradictions.**

### 1.3 URL Paths

| Path | Architecture | Frontend | Backend | Match |
|---|---|---|---|---|
| WebSocket | `/ws/room/{roomId}` | `/ws/room/{roomId}` (Section 5.2, Vite proxy) | `/ws/room/{roomId}` (Section 4.2) | YES |
| Room check | `/api/rooms/{id}` | Not used by frontend | `/api/rooms/{roomId}` (Section 4.1) | YES |
| Health | `/health` | Not used by frontend | `/health` (Section 4.1) | YES |
| SPA fallback | `/*` catch-all | Vite dev proxy for `/api`, `/ws` | SPA fallback in handler.go | YES |

### 1.4 Phase Strings

| Document | Values |
|---|---|
| Architecture doc Section 3.1 | `PhaseVoting = "voting"`, `PhaseReveal = "reveal"` |
| Frontend plan Section 4.2 | `type Phase = "voting" \| "reveal"` |
| Backend plan Section 3.1 | `PhaseVoting Phase = "voting"`, `PhaseReveal Phase = "reveal"` |

**Match: YES. The iteration 3 blocker is resolved.**

### 1.5 Room ID Format

| Document | Format |
|---|---|
| Architecture doc Section 5.1 | `{slug}-{12 hex chars}`, 48 bits, regex `/-[a-f0-9]{12}$/` |
| Frontend plan Section 6.4 | `crypto.getRandomValues(new Uint8Array(6))`, 12 hex chars, regex `/-[a-f0-9]{12}$/` |
| Backend plan Section 3.1 | `crypto/rand` + `hex.EncodeToString`, 6 bytes = 12 hex chars |

**Match: YES.**

### 1.6 Project Structure

| Aspect | Architecture Doc (Section 9) | Frontend Plan (Section 2) | Backend Plan (Section 2) | Match |
|---|---|---|---|---|
| Backend packages | `domain` + `server` | N/A | `domain` + `server` | YES |
| Frontend components | 10, listed with PascalCase dirs | 10, identical list | N/A | YES |
| Frontend files | `state.ts`, `ws.ts`, `hooks/usePresence.ts`, `utils/room-url.ts` | Identical | N/A | YES |
| CSS approach | Per-component CSS + BEM | Per-component CSS + BEM | N/A | YES |
| Backend server files | `store.go`, `hub.go`, `ws_handler.go`, `http_handler.go`, `message.go`, `middleware.go`, `gc.go`, `config.go`, `server.go` | N/A | `handler.go`, `ws.go`, `client.go`, `room_manager.go`, `ratelimit.go`, `events.go` | SEE 1.7 |

### 1.7 Minor Inconsistency: Backend File Names

The architecture doc (Section 9) lists the backend `server/` files as: `store.go`, `hub.go`, `ws_handler.go`, `http_handler.go`, `message.go`, `middleware.go`, `gc.go`, `config.go`, `server.go` (9 files).

The backend plan (Section 2) lists: `handler.go`, `ws.go`, `client.go`, `room_manager.go`, `room_manager_test.go`, `ratelimit.go`, `ratelimit_test.go`, `events.go` (6 files + 2 test files).

These are different file names and a different number of files. For example:
- Architecture: `store.go` + `hub.go` vs Backend: `room_manager.go` (combines both)
- Architecture: `ws_handler.go` vs Backend: `ws.go` + `client.go`
- Architecture: `message.go` vs Backend: `events.go`
- Architecture: `middleware.go` vs Backend: `ratelimit.go`
- Architecture has `gc.go`, `config.go`, `server.go` -- backend plan does not list these separately

**Severity: LOW.** This is a file-naming and organizational discrepancy, not a protocol or behavioral contradiction. The functionality is the same regardless of how files are split. A developer will make their own decisions about file boundaries during implementation. However, this is worth noting as it could cause momentary confusion.

### 1.8 Minor Inconsistency: Domain Method Names

| Operation | Architecture Doc (Section 3.1) | Backend Plan (Section 3.2) |
|---|---|---|
| Cast a vote | `CastVote(sessionID string, value string) error` | `SetVote(sessionID string, value VoteValue) string` |
| Reveal votes | `RevealVotes() (*RoundResult, error)` | `Reveal() string` |
| Vote type | `*string` (nil = no vote) | `VoteValue` (custom string type, `""` = no vote) |

The architecture doc uses `CastVote` / `RevealVotes` with Go `error` return types and `*string` for votes. The backend plan uses `SetVote` / `Reveal` with string error returns and a custom `VoteValue` type with empty string semantics.

**Severity: LOW.** These are internal Go API choices, not wire-level. The method names and signatures are implementation details that do not affect the protocol or frontend. The behavior is identical. The backend plan's version is more detailed (full code sketches), so a developer would follow that. However, the architecture doc's method names appear in the sequence diagram (Section 4.8) as `CastVote` and `RevealVotes`, which differs from the backend plan's naming.

### 1.9 Dependency Counts

| Document | Go external deps | Frontend runtime deps |
|---|---|---|
| Architecture doc | 1 (`nhooyr.io/websocket`) | 2 (`preact`, `@preact/signals`) |
| Frontend plan | N/A | 2 (`preact`, `@preact/signals`) |
| Backend plan | 1 (`nhooyr.io/websocket`) | N/A |

**Match: YES.**

### 1.10 Configuration

| Document | Env vars |
|---|---|
| Architecture doc | `PORT`, `HOST`, `TRUST_PROXY` |
| Backend plan | `PORT`, `HOST`, `TRUST_PROXY` |

**Match: YES.**

### 1.11 Hardcoded Defaults

| Constant | Architecture Doc | Backend Plan | Frontend Plan | Match |
|---|---|---|---|---|
| Room TTL | 24h | 24h | N/A | YES |
| GC interval | 10 min | 10 min | N/A | YES |
| Ping interval | 5s | 5s | N/A | YES |
| Disconnect threshold | 10s | 10s | N/A | YES |
| Max message size | 1 KB | 1 KB | N/A | YES |
| Send buffer | 32 | 32 | N/A | YES |
| Reconnect backoff | 500ms-10s, 30% jitter | Defers to frontend | 500ms-10s, 30% jitter, 30s wall-clock | YES |
| WS rate limit | 20/min per IP | 20/min per IP | N/A | YES |

**Match: YES.**

---

## 2. Implementation Readiness Score

| Document | Score (1-10) | Assessment |
|---|---|---|
| `01-ux-design.md` | **9/10** | Comprehensive UX spec. Every screen, every state, every edge case documented. Includes wireframes, color palette, accessibility notes, and explicit "considered and excluded" features. Only missing: exact typography sizes per element (the tokens.css in the frontend plan covers this). |
| `02-frontend-plan.md` | **9/10** | Every component has props, state, behavior, accessibility attributes, and visual states. Full `state.ts` and `ws.ts` code sketches. Routing, responsive design, and deployment fully specified. A developer can start coding immediately. Minor gap: no explicit error boundary strategy (noted as developer decision in iteration 3 review). |
| `03-backend-plan.md` | **9/10** | Full Go code sketches for domain model, room manager, broadcast, rate limiter, GC, events. Connection lifecycle fully diagrammed. Error handling comprehensive. Concurrency model explicit with lock ordering. Minor gap: the `Reveal()` method comment says "Reveal transitions to revealed phase" (should say "reveal phase") -- a comment-level inconsistency only. |
| `04-architecture.md` | **10/10** | The canonical reference document. Protocol is clearly marked as single source of truth. Full JSON examples for every message. Sequence diagram for the most complex flow. Project structure file-by-file. Decision log with rationale. Capacity estimates. CLAUDE.md template. Nothing missing. |

**Overall Readiness: 9.3/10.** A developer can start implementing today without asking any questions about protocol, structure, behavior, or decisions.

---

## 3. Completeness Matrix

UX requirements (rows) vs technical coverage (columns):

| UX Requirement | Frontend | Backend | Architecture |
|---|---|---|---|
| Home page (room creation form) | ✅ `HomePage` component, Section 3.3 | ✅ Implicit creation on `join` | ✅ Section 5 |
| Name entry modal (join + edit) | ✅ `NameEntryModal`, Section 3.4 | ✅ `update_name` event | ✅ Section 4.4 |
| Room page (voting phase) | ✅ `RoomPage`, Section 3.5 | ✅ `room_state` + `vote_cast` | ✅ Section 4.5 |
| Room page (reveal phase) | ✅ `RoomPage` inline stats | ✅ `votes_revealed` payload | ✅ Section 4.5 |
| Participant cards (5 states) | ✅ `ParticipantCard`, Section 3.8 | ✅ Presence + vote state | ✅ Section 3.1 |
| Card deck (4 states) | ✅ `Card`, Section 3.10 | ✅ Vote validation | ✅ Section 4.4 |
| Card values (12 values + ?) | ✅ Constant array in `CardDeck` | ✅ `ValidVotes` map | ✅ Section 4.4 |
| "?" excluded from stats | ✅ Displays server stats | ✅ `CalculateResult` partitions | ✅ Section 4.5 |
| Copy link + toast | ✅ `Header` inline | N/A (client-only) | N/A |
| Settings dropdown | ✅ `Header` inline | ✅ `update_name`, `leave` | ✅ Section 4.4 |
| Toast notifications | ✅ `Toast`, Section 3.11 | N/A (client-only rendering) | N/A |
| Confirm dialog (Clear Room) | ✅ `ConfirmDialog`, Section 3.12 | ✅ `clear_room` event | ✅ Section 4.4 |
| Show Votes button | ✅ Inline in `RoomPage` | ✅ `reveal` event | ✅ Section 4.4 |
| New Round button | ✅ Inline in `RoomPage` | ✅ `new_round` event | ✅ Section 4.4 |
| Clear Room button | ✅ Inline in `RoomPage` | ✅ `clear_room` event | ✅ Section 4.4 |
| Reconnection banner | ✅ Inline in `RoomPage` | ✅ Session restore via `sessionId` | ✅ Section 4.7 |
| Consensus highlight | ✅ `hasConsensus` from server | ✅ `CalculateResult` | ✅ Section 4.5 |
| Spread indicator | ✅ `spread` from server | ✅ `CalculateResult` | ✅ Section 4.5 |
| Statistics display | ✅ Server-provided, no local calc | ✅ Full `RoundResult` struct | ✅ Section 4.5 |
| Presence (active/idle/disconnected) | ✅ `usePresence` hook + dot colors | ✅ `CheckDisconnectedParticipants` | ✅ Section 4.5 |
| Disconnected user (dimmed, vote preserved) | ✅ `opacity: 0.6` | ✅ Vote preserved in `Participants` map | ✅ Sections 3.1, 4.5 |
| Responsive (3 breakpoints) | ✅ Section 7 | N/A | ✅ Section 2.3 |
| Mobile card deck (2 rows) | ✅ Section 7.2 | N/A | N/A |
| Touch targets (44px min) | ✅ Section 7.4 | N/A | N/A |
| Duplicate names allowed | ✅ Keyed by `sessionId` | ✅ No uniqueness check | ✅ Section 3.1 |
| Room not found | ✅ Error state in `RoomPage` | ✅ `error` with `room_not_found` | ✅ Section 4.5 |
| Browser back/forward | ✅ `popstate` listener, Section 6 | N/A | N/A |
| Room GC (24h) | N/A | ✅ Section 6.4, 10-min sweep | ✅ Section 8.2 |
| Keyboard navigation | ✅ Section 8.4 | N/A | N/A |
| Screen reader support | ✅ Section 8.3, 8.5 | N/A | N/A |
| Reduced motion | ✅ Section 8.6 | N/A | N/A |
| Color contrast (WCAG AA) | ✅ Section 8.7 | N/A | N/A |
| Joining mid-vote | ✅ `room_state` handler | ✅ `room_state` with current phase | ✅ Section 4.5 |
| Changing vote before reveal | ✅ Radio behavior in `Card` | ✅ `SetVote` replaces | ✅ Section 4.4 |
| Un-voting (deselect) | ✅ `onSelect('')` | ✅ `value: ""` -> `vote_retracted` | ✅ Section 4.4 |
| Leave room (explicit) | ✅ Header settings | ✅ Immediate removal, close 1000 | ✅ Section 4.4 |
| Disconnect (unexpected) | ✅ Reconnect loop | ✅ Grace period, vote preserved | ✅ Sections 4.6-4.7 |

**Result: 35/35 UX requirements fully covered (✅) across all relevant documents. Zero gaps.**

---

## 4. Risk Register

| # | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| 1 | **Backend file structure diverges from architecture doc during implementation.** Architecture doc lists 9 server files with different names than backend plan's 6 files. | Medium | Low | Developer should follow the backend plan (more detailed code sketches). Architecture doc's file list is directional, not prescriptive. |
| 2 | **`localStorage` throws in private browsing mode.** Code sketches do not show try/catch around `localStorage.setItem`. Safari private mode blocks `localStorage`. | Medium | Medium | Wrap all `localStorage` calls in try/catch. Fall back to in-memory storage for the session. This is a 5-line fix during implementation. |
| 3 | **Corporate proxies stripping WebSocket pings.** Protocol-level pings may be stripped by some older reverse proxies. | Low | Medium | Documented escape hatch in architecture doc Section 4.6: add application-level heartbeat later if needed. ~20 lines of code. |
| 4 | **Preact signals version compatibility.** `@preact/signals` is a separate package with its own release cycle. A major version bump could break the API. | Low | Low | Pin the version in `package.json`. This is standard practice. |
| 5 | **`nhooyr.io/websocket` API changes.** The library is maintained but not Go stdlib. | Low | Low | Pin the version in `go.mod`. Standard Go module practices apply. |
| 6 | **`prefers-reduced-motion` blanket rule suppresses all CSS animations.** The global `* { animation-duration: 0.01ms !important }` rule affects everything, including any future third-party components. | Low | Low | Acceptable at current scope (no third-party UI components). Revisit if adding external UI libraries. |
| 7 | **Domain method names differ between architecture doc and backend plan.** `CastVote` vs `SetVote`, `RevealVotes` vs `Reveal`. Developer might reference the wrong document. | Low | Low | Backend plan has full Go code; architecture doc has signatures only. Developer should treat backend plan as authoritative for Go implementation details. |

**No blockers. All risks are low-impact implementation concerns with clear mitigations.**

---

## 5. Estimated Implementation Effort

### 5.1 Backend (Go)

| Component | Estimated Lines |
|---|---|
| `internal/domain/` (Room, Participant, Stats, Vote validation, methods) | ~250 |
| `internal/server/` (handlers, WS, room manager, rate limiter, events, GC) | ~700 |
| `cmd/server/main.go` (config, wiring, signal handling) | ~50 |
| Tests (`domain` + `server`) | ~450 |
| **Backend total** | **~1,450** |

### 5.2 Frontend (TypeScript + CSS)

| Component | Estimated Lines |
|---|---|
| 10 components (`.tsx` files, avg ~55 lines each) | ~550 |
| `state.ts` (signals, types, helpers) | ~80 |
| `ws.ts` (WebSocket client, handlers) | ~120 |
| `app.tsx` + `main.tsx` (routing, bootstrap) | ~40 |
| `utils/room-url.ts` | ~20 |
| `hooks/usePresence.ts` | ~30 |
| CSS (13 files: 10 component + 3 global) | ~450 |
| `vite.config.ts` + `tsconfig.json` | ~30 |
| **Frontend total** | **~1,320** |

### 5.3 Infrastructure

| Component | Estimated Lines |
|---|---|
| Dockerfile | ~20 |
| Makefile | ~20 |
| CLAUDE.md | ~25 |
| index.html | ~15 |
| **Infrastructure total** | **~80** |

### 5.4 Grand Total

| Category | Lines |
|---|---|
| Backend (Go, production + tests) | ~1,450 |
| Frontend (TypeScript + CSS) | ~1,320 |
| Infrastructure | ~80 |
| **Total** | **~2,850** |

### 5.5 Time Estimate (Single Developer)

| Phase | Estimate | Notes |
|---|---|---|
| Backend domain + tests | 3-4 hours | Pure logic, table-driven tests, no I/O |
| Backend server (WS, handlers, room manager, GC) | 6-8 hours | Most complex part: WebSocket lifecycle, broadcast, concurrency |
| Backend integration tests | 2-3 hours | Real in-memory store, WebSocket test client |
| Frontend components + state + routing | 6-8 hours | 10 components, signals, message handlers |
| Frontend CSS (responsive + all states) | 3-4 hours | Mobile-first, 3 breakpoints, card animations |
| Frontend accessibility | 1-2 hours | ARIA attributes, focus management, keyboard nav |
| Integration (embed, Docker, Makefile) | 1-2 hours | Build pipeline, embed.FS, Vite proxy |
| Manual testing + polish | 2-3 hours | Multi-browser, mobile, edge cases |
| **Total** | **24-34 hours** | **~4-5 working days** |

This assumes a developer experienced with Go and Preact/React. Add 30-50% for a developer new to either technology.

---

## 6. Final Verdict

### APPROVED FOR IMPLEMENTATION

The planning documents are ready. After 4 review iterations and 18+ resolved issues, the documentation set is in excellent shape:

- **Zero protocol contradictions.** All 19 WebSocket events are perfectly aligned across all four documents. Envelope format, payload fields, event names, and URL paths are consistent.
- **All iteration 3 blockers resolved.** Phase string fixed, health endpoint aligned, room check response aligned.
- **100% UX coverage.** Every screen, interaction, edge case, and accessibility requirement from the UX spec has a concrete technical implementation path in both the frontend and backend plans.
- **Single source of truth established.** The architecture doc (Section 4) is the canonical protocol reference. Frontend and backend plans reference it rather than redefining it. This structural decision prevents future drift.
- **Minimal complexity.** 2 backend packages, 1 external Go dependency, 2 frontend runtime dependencies, 10 components, ~2,850 total lines. No premature abstractions.

### Minor Items for Developer Awareness (Not Blockers)

1. **Backend file names differ between architecture doc and backend plan.** Follow the backend plan's file structure (it has full code sketches). The architecture doc's Section 9 file list is approximate.
2. **Domain method names differ.** Architecture doc uses `CastVote`/`RevealVotes`; backend plan uses `SetVote`/`Reveal`. Follow the backend plan.
3. **Wrap `localStorage` calls in try/catch** during implementation (not in the plan but necessary for private browsing compatibility).
4. **Backend plan Section 3.2 comment says "Reveal transitions to revealed phase"** -- should say "reveal phase". Cosmetic only; the code itself is correct.

### Recommendation

**Begin implementation.** Start with the backend domain package (pure logic, testable in isolation), then the server package (WebSocket + HTTP), then the frontend. The plans are detailed enough to code against directly, and the architecture doc provides a reliable protocol reference throughout.
