# om-scrum-poker: Planning Review -- Iteration 3

**Version:** 1.0
**Date:** 2026-04-04
**Reviewer Roles:** Product Owner / Chief Architect + Devil's Advocate
**Status:** Review Complete
**Documents Reviewed (all v3.0 except UX):**
- `01-ux-design.md` (UX Design, v1.0 -- unchanged)
- `02-frontend-plan.md` (Frontend Plan, v3.0)
- `03-backend-plan.md` (Backend Plan, v3.0)
- `04-architecture.md` (Architecture, v3.0)
- `06-review-iteration-2.md` (Previous review -- to verify all 18 items addressed)

---

## Part 1: Iteration 2 Follow-up

### Were all 18 actionable items addressed?

| # | Item | Status | Notes |
|---|---|---|---|
| 1 | [ALL] [MUST] Resolve envelope format | ADDRESSED | All three docs now use `{ type, payload }`. Architecture doc changelog, decision log, and CLAUDE.md template all updated. No remaining flat-format references. |
| 2 | [ALL] [MUST] Agree on WebSocket URL path | ADDRESSED | All three docs use `/ws/room/{roomId}`. |
| 3 | [ALL] [MUST] Unify `name_changed` vs `name_updated` | ADDRESSED | All docs use `name_updated`. Backend plan changelog explicitly notes the rename. |
| 4 | [ALL] [MUST] Unify presence event name | ADDRESSED | Frontend plan changed from `presence_update` to `presence`. `usePresence` hook sends `{ type: "presence", payload: { status } }`. |
| 5 | [FRONTEND] [MUST] Fix room ID suffix length | ADDRESSED | `generateRoomUrl` now uses `crypto.getRandomValues(new Uint8Array(6))` producing 12 hex chars. `extractRoomName` regex updated to `/-[a-f0-9]{12}$/`. |
| 6 | [ALL] [MUST] Unify `votes_revealed` payload | ADDRESSED | All docs use flat stats alongside `votes`: `average`, `median`, `uncertainCount`, `totalVoters`, `hasConsensus`, `spread`. `spread` is `[min, max]` array or `null`. Consistent across architecture doc Section 4.5, frontend `RoundResult` interface, and backend `RoundResult` struct. |
| 7 | [ALL] [MUST] Unify `room_state` payload | ADDRESSED | All docs use `roomId`, `roomName`, `phase`, `participants` (with `sessionId`, `userName`, `hasVoted`, `vote`, `status`), `result`. Architecture doc has full examples for both voting and reveal phases. |
| 8 | [BACKEND] [SHOULD] Remove full protocol from backend plan | ADDRESSED | Backend plan Section 5 (WebSocket Protocol) removed. Backend plan now has a protocol reference header and all handler sections reference the architecture doc. `events.go` constants are annotated with "MUST match Architecture Doc v3.0, Section 4.3". |
| 9 | [FRONTEND] [SHOULD] Remove application-level heartbeat | ADDRESSED | No `heartbeat` in outbound/inbound types. No `case "heartbeat"` in `handleMessage`. Comment explicitly says "No heartbeat case". |
| 10 | [BACKEND] [SHOULD] Fix heartbeat interval | ADDRESSED | 5 seconds consistently. Contradictory 30-second reference removed. |
| 11 | [ARCHITECTURE] [SHOULD] Fix send buffer size | ADDRESSED | Architecture doc Section 6.4 now says 32. Backend plan says 32. Consistent. |
| 12 | [ALL] [SHOULD] Unify `participant_joined` payload | ADDRESSED | All docs include `status` field. Backend changelog explicitly notes the addition. |
| 13 | [ALL] [SHOULD] Unify error event payload | ADDRESSED | All docs include `{ code, message }`. Backend plan Section 12.2 and architecture doc Section 4.5 consistent. |
| 14 | [ALL] [SHOULD] Agree on frontend directory structure | ADDRESSED | Architecture doc Section 9 now matches frontend plan: `components/` with PascalCase directories, `ws.ts` single file, `utils/`, `hooks/`, per-component CSS. |
| 15 | [FRONTEND] [COULD] Eliminate `utils/stats.ts` | ADDRESSED | Removed. Frontend plan v3.0 changelog item 8. Server always includes stats. Frontend `handleMessage` reads server-provided values directly. |
| 16 | [ALL] [COULD] Add `roomName` to `join` message | ADDRESSED | All three docs include `roomName` in the `join` payload. Architecture doc Section 4.4 explains first-joiner semantics. Backend plan Section 4.2 updated. Frontend plan Section 5.2 updated. |
| 17 | [FRONTEND] [COULD] Unify CSS approach | ADDRESSED | All docs agree on per-component CSS files with BEM naming. Architecture doc decision log updated. |
| 18 | [ALL] [COULD] Align reconnection backoff caps | ADDRESSED | Backend plan removed its client-side reconnection description. Frontend plan is authoritative: 500ms-10s cap, 30% jitter, 30s wall-clock timeout. Architecture doc Section 4.7 references the same values. |

**Summary: 18 of 18 items fully addressed.** No partially addressed items remain.

---

## Part 2: Protocol Consistency Verification (Event-by-Event)

This is the critical check. I went through every event mentioned in any document and verified it against the architecture doc Section 4.3 (the canonical event table).

### 2.1 Client-to-Server Events

| Event | Architecture Doc | Frontend Plan | Backend Plan | Match? |
|---|---|---|---|---|
| `join` | `sessionId`, `userName`, `roomName` | Section 5.2: sends `{ type: "join", payload: { sessionId, userName, roomName } }` | Section 4.2: validates `sessionId`, `userName`, `roomName` from join payload | YES |
| `vote` | `value` (string, or `""` to retract) | Section 5.4: `handleMessage` processes `vote_cast` and `vote_retracted` responses correctly | Section 5.4: dispatches to `room.SetVote(sessionId, value)`, broadcasts `vote_cast` or `vote_retracted` based on value | YES |
| `reveal` | empty payload | Frontend sends `{ type: "reveal", payload: {} }` | Backend dispatches to `room.Reveal()`, broadcasts `votes_revealed` | YES |
| `new_round` | empty payload | Frontend sends `{ type: "new_round", payload: {} }` | Backend dispatches to `room.NewRound()`, broadcasts `round_reset` | YES |
| `clear_room` | empty payload | Frontend sends after ConfirmDialog confirmation | Backend dispatches to `room.Clear()`, broadcasts `room_cleared` | YES |
| `update_name` | `userName` | Section 5.5: sends `{ type: "update_name", payload: { userName: "New Name" } }` | Section 5.4: dispatches to `room.UpdateParticipantName()`, broadcasts `name_updated` | YES |
| `presence` | `status` (`"active"` or `"idle"`) | Section 5.6: `usePresence` sends `{ type: "presence", payload: { status } }` | Section 5.4: updates participant status, broadcasts `presence_changed` | YES |
| `leave` | empty payload | Header settings menu: "Leave room" triggers `wsDisconnect()` and navigate | Section 5.2: removes participant immediately, broadcasts `participant_left`, closes with code 1000 | YES |

### 2.2 Server-to-Client Events

| Event | Architecture Doc | Frontend Plan | Backend Plan | Match? |
|---|---|---|---|---|
| `room_state` | `roomId`, `roomName`, `phase`, `participants`, `result` | Section 5.4 handler reads all five fields. `participants` includes `sessionId`, `userName`, `hasVoted`, `vote`, `status`. | Section 4.2: sends full snapshot on join. Section 9 stats included during reveal. | YES |
| `participant_joined` | `sessionId`, `userName`, `status` | Section 5.4: adds participant with all three fields, shows toast | Section 4.2: broadcasts to all except joiner | YES |
| `participant_left` | `sessionId` | Section 5.4: finds name for toast, filters out participant | Section 5.2/5.4: broadcasts on explicit leave or grace expiry | YES |
| `vote_cast` | `sessionId` | Section 5.4: sets `hasVoted: true` for matching participant | Section 5.4: broadcast when non-empty vote received | YES |
| `vote_retracted` | `sessionId` | Section 5.4: sets `hasVoted: false` for matching participant | Section 5.4: broadcast when `value: ""` received | YES |
| `votes_revealed` | `votes`, `average`, `median`, `uncertainCount`, `totalVoters`, `hasConsensus`, `spread` | Section 5.4: reads all seven fields into `roundResult`, updates participant vote values from `votes` array. `RoundResult` interface in `state.ts` matches. | Section 9: `RoundResult` struct has matching JSON tags. `CalculateResult` computes all fields. | YES |
| `round_reset` | empty payload | Section 5.4: resets phase to `"voting"`, clears `selectedCard`, `roundResult`, resets all participants | Section 5.4: dispatches `room.NewRound()` | YES |
| `room_cleared` | empty payload | Section 5.4: clears participants, resets phase, shows toast | Section 5.4: dispatches `room.Clear()`, closes all connections | YES |
| `presence_changed` | `sessionId`, `status` | Section 5.4: updates matching participant's status | Section 7.3-7.4: broadcast on status change | YES |
| `name_updated` | `sessionId`, `userName` | Section 5.4: updates matching participant's userName | Section 5.4: dispatches `room.UpdateParticipantName()` | YES |
| `error` | `code`, `message` | Section 5.4/5.7: shows toast, handles `room_not_found` specially | Section 12.2: full error code table, uses `{ type, payload: { code, message } }` envelope | YES |

### 2.3 URL Paths

| Path | Architecture Doc | Frontend Plan | Backend Plan | Match? |
|---|---|---|---|---|
| WebSocket | `/ws/room/{roomId}` (Section 4.1) | `/ws/room/{roomId}` (Section 5.2, Vite proxy) | `/ws/room/{roomId}` (Section 4.2) | YES |
| Room check | `/api/rooms/{id}` (Section 5 table) | Not explicitly used (frontend goes straight to WebSocket) | `/api/rooms/{roomId}` (Section 4.1) | YES |
| Health | `/health` (Section 5 table) | Not referenced (backend-only) | `/api/health` (Section 4.1) | SEE 3.1 |

### 2.4 Remaining Issue Found: Phase Value String

| Document | Phase values |
|---|---|
| **Architecture doc** Section 3.1 | `PhaseVoting = "voting"`, `PhaseReveal = "reveal"` |
| **Frontend plan** Section 4.2 | `type Phase = "voting" \| "reveal"` |
| **Backend plan** Section 3.1 | `PhaseVoting = "voting"`, `PhaseRevealed = "revealed"` |

The backend plan uses `"revealed"` (past tense) while the architecture doc and frontend plan use `"reveal"` (present tense). This is a wire-level mismatch: when the server sends `room_state` with `phase: "revealed"`, the frontend's `Phase` type expects `"voting" | "reveal"` and would not match correctly.

This is the only protocol-level contradiction remaining across all documents.

---

## Part 3: Remaining Issues

### 3.1 [MUST] Phase string mismatch: `"reveal"` vs `"revealed"`

**Impact:** Wire-level bug. The backend Go code has `PhaseRevealed Phase = "revealed"` (line 157 of backend plan). This string is serialized into `room_state` and compared by the frontend. The frontend TypeScript type is `"voting" | "reveal"`. If the server sends `"revealed"`, the frontend's phase comparison logic will not match.

**Fix:** Change the backend plan's `PhaseRevealed` constant from `"revealed"` to `"reveal"` to match the architecture doc. This is a one-word change: `PhaseRevealed Phase = "revealed"` becomes `PhaseReveal Phase = "reveal"`. Also rename the constant from `PhaseRevealed` to `PhaseReveal` for consistency with the architecture doc's naming.

### 3.2 [SHOULD] Health endpoint path: `/health` vs `/api/health`

The architecture doc's HTTP endpoints table (Section 5) lists the health endpoint as `/health`. The backend plan (Section 4.1) lists it as `/api/health`. This is minor -- the health endpoint is not used by the frontend -- but it should be consistent. The backend plan's `/api/health` is the more conventional path.

**Fix:** Update the architecture doc's HTTP endpoints table to use `/api/health`.

### 3.3 [SHOULD] `GET /api/rooms/{id}` response body inconsistency

The architecture doc Section 5 says this endpoint returns `200 { "exists": true }`. The backend plan Section 4.1 returns `200 { "id": "...", "name": "...", "phase": "...", "participantCount": 4 }`. These are different response shapes. Since the frontend does not currently use this endpoint (it goes straight to WebSocket), this has zero runtime impact, but a developer implementing both sides would face ambiguity.

**Fix:** Pick one. The backend plan's richer response is more useful. Update the architecture doc to match.

---

## Part 4: Implementation Readiness Assessment

### 4.1 Could a developer start coding now?

**Yes.** With the single exception of the `"reveal"` vs `"revealed"` phase string (Section 3.1 above), every aspect of the system is fully specified:

- Every WebSocket event has a canonical definition with JSON examples.
- Every component has props, state, behavior, and visual states documented.
- Every domain method has a Go code sketch with signatures and behavior.
- Every handler dispatch path is mapped.
- The project structure is file-by-file identical across documents.
- Build, test, and deployment commands are specified.

### 4.2 Design decisions remaining for the developer

None that are architectural. The remaining decisions are implementation-level choices that any developer would make naturally:

| Area | Decision Left to Developer | Risk |
|---|---|---|
| CSS specifics | Exact spacing, font weights, animation easing curves | None -- visual polish, not architecture |
| Error boundary | Whether to wrap the entire app in an error boundary component | Low -- nice to have but not critical |
| TypeScript strictness | Which `tsconfig` strict options to enable beyond `strict: true` | None -- `strict: true` is specified |
| Test file organization | Whether to co-locate tests or use a `__tests__` directory | None -- convention preference |
| `slog` format | JSON vs text format for structured logging | None -- operational preference |

### 4.3 Every UX flow technically specified?

| UX Flow | Frontend Coverage | Backend Coverage | Protocol Coverage |
|---|---|---|---|
| Room creation | `HomePage.tsx` + `generateRoomUrl` | Implicit creation on first `join` | `join` event creates room |
| Joining via link | `RoomPage.tsx` lifecycle + `NameEntryModal` | `AddParticipant` + `room_state` response | `join` + `room_state` |
| Voting | `CardDeck` + `Card` + `wsSend` | `SetVote` + broadcast | `vote` + `vote_cast` / `vote_retracted` |
| Changing vote | Same as voting (radio behavior) | `SetVote` replaces previous | Same |
| Un-voting (deselect) | `Card` toggle: `onSelect('')` | `SetVote` with empty string | `vote` with `value: ""` + `vote_retracted` |
| Reveal | `RoomPage` inline "Show Votes" button | `Reveal` + `CalculateResult` | `reveal` + `votes_revealed` |
| New round | `RoomPage` inline "New Round" button | `NewRound` clears all | `new_round` + `round_reset` |
| Clear room | `ConfirmDialog` + `RoomPage` | `Clear` removes all | `clear_room` + `room_cleared` |
| Name change | `NameEntryModal` edit mode + `Header` settings | `UpdateParticipantName` | `update_name` + `name_updated` |
| Leave room | `Header` settings menu | `RemoveParticipant` | `leave` + `participant_left` |
| Reconnection | `ws.ts` backoff + reconnection banner | Session restore via `sessionId` | `join` with existing `sessionId` |
| Copy link | `Header` inline button | N/A (client-only) | N/A |
| Room not found | `RoomPage` error state + link to `/` | `error` with `room_not_found` | `error` event |
| Presence (active/idle/disconnected) | `usePresence` hook + `ParticipantCard` dot | `CheckDisconnectedParticipants` + `LastSeen` | `presence` + `presence_changed` |

**100% coverage.** Every user flow from the UX doc has a complete technical path.

---

## Part 5: UX Coverage Checklist

| UX Requirement | Specified? | Notes |
|---|---|---|
| Home page (room creation form) | YES | `HomePage` component, Section 3.3 of frontend plan |
| Name entry modal (join + edit modes) | YES | `NameEntryModal` component with `mode` prop, full ARIA spec |
| Room page (voting phase layout) | YES | `RoomPage` with `ParticipantList`, `CardDeck`, action buttons |
| Room page (reveal phase layout) | YES | Phase-dependent rendering, statistics inline in `RoomPage` |
| Participant cards (all 5 visual states) | YES | `ParticipantCard` Section 3.8 covers not-voted, voted, revealed, did-not-vote, disconnected |
| Card deck (all 4 card states) | YES | `Card` Section 3.10 covers default, hover, selected, disabled |
| Card values: ?, 0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100 | YES | Constant array in `CardDeck`, validated server-side via `ValidVotes` map |
| "?" excluded from stats | YES | Backend `CalculateResult` partitions votes; `uncertainCount` field in results |
| Copy link + "Copied!" toast | YES | `Header` inline copy button with 1.5s label toggle + toast |
| Settings dropdown (change name, leave) | YES | `Header` inline dropdown with two items |
| Toast notifications (all 7 messages) | YES | `Toast` component. Messages: "Link copied!", "{name} joined", "{name} left", "Room cleared", "New round started", "Reconnected", error messages |
| Confirmation dialog (Clear Room only) | YES | `ConfirmDialog` with exact title/body text from UX doc Section 4.5 |
| Reconnection banner | YES | `RoomPage` inline conditional div. "Reconnecting..." (yellow) and "Connection lost. [Retry]" (red) |
| Consensus highlight ("Consensus! Everyone voted X") | YES | `hasConsensus` field in `votes_revealed`; frontend displays consensus/spread status inline in statistics area |
| Spread indicator ("High spread X to Y") | YES | `spread: [min, max]` in `votes_revealed`; frontend renders spread info |
| Statistics: average, median, vote count, uncertain count | YES | All included in `votes_revealed` and `room_state` during reveal |
| Presence: active (green), idle (yellow), disconnected (red) | YES | 10px dot with CSS transitions. Timing thresholds: 2min idle, 10s disconnect, 5s ping. |
| Disconnected user behavior (dimmed, vote preserved) | YES | `opacity: 0.6`, vote stays. Participant remains until Clear Room or GC. |
| Responsive design (3 breakpoints) | YES | Frontend plan Section 7: mobile <640px, tablet 640-1024px, desktop >1024px. Mobile-first approach. |
| Mobile card deck (2 rows of 6) | YES | Frontend plan Section 7.2: `flex-wrap: wrap`, 2 rows. Sticky positioning. |
| Touch targets (44px minimum) | YES | Frontend plan Section 7.4: 44x44px minimum with padding extension technique. |
| Landscape mobile | YES | Frontend plan Section 7 inherits from wrapping behavior (per UX doc Section 8.5). |
| Duplicate names allowed | YES | Backend allows it (keyed by `sessionId`). UX doc Section 9.6 requirement met. |
| Room not found page | YES | `RoomPage` renders error state with link to home page on `room_not_found` error. |
| Browser back/forward | YES | Frontend plan Section 6: `popstate` listener, signal-based routing. No "are you sure?" prompts. |
| Room GC (24h) | YES | Backend plan Section 6.4: 10-minute sweep, 24h threshold. |
| Keyboard navigation | YES | Frontend plan Section 8.4: all elements focusable, arrow keys in radiogroup, Escape closes modals. |
| Screen reader support | YES | Frontend plan Section 8.3/8.5: ARIA roles, live regions, state announcements. |
| Reduced motion | YES | Frontend plan Section 8.6: global `prefers-reduced-motion: reduce` rule. |
| Color contrast (WCAG AA) | YES | Frontend plan Section 8.7: contrast ratios verified against the palette. |

**UX coverage: 100%.** No missing screens, interactions, or edge cases.

---

## Part 6: Devil's Advocate Final Pass

### 6.1 Is there anything left that could be simpler?

No. The v3.0 documents represent the simplest viable architecture for this problem:

- 2 backend packages (down from 4 in v1.0)
- 1 external Go dependency
- 2 runtime frontend dependencies (~6 KB)
- 10 frontend components (down from 16)
- Single `state.ts` file (down from 4 state files)
- Single `ws.ts` file (down from 3 WS files)
- No `stats.ts` (server-side only)
- No application-level heartbeat
- 3 environment variables
- 20-line custom router (no library)

The only place I considered trimming further was the `ConfirmDialog` component (used once), but it was explicitly kept in iteration 2 as a reasonable separation. It is 15-20 lines of JSX and its own small CSS file. Inlining it would save a directory but add visual noise to `RoomPage`. The current split is correct.

### 6.2 Any remaining premature abstractions?

No. There are:
- No interfaces with single implementations
- No service layer
- No DTOs
- No event bus
- No middleware chain (just rate limiting + recovery)
- No configuration framework
- No router library
- No state management library (signals are 2 KB, not a framework)

### 6.3 Code volume assessment

| Component | Estimated Lines | Reasonable? |
|---|---|---|
| Backend `domain/` | ~250 | YES. Room, Participant, Stats, Vote validation. Straightforward. |
| Backend `server/` | ~700 | YES. Handlers, WS, room manager, rate limiter, events. 6 files averaging ~117 lines each. |
| Backend `cmd/server/` | ~50 | YES. Config, wiring, signal handling. |
| Backend tests | ~400-500 | YES. Table-driven tests for domain, integration tests for server. |
| **Backend total** | **~1,400-1,500** | REASONABLE |
| Frontend components (10) | ~500-600 | YES. Averaging ~50-60 lines each including JSX and handlers. |
| Frontend `state.ts` + `ws.ts` | ~200 | YES. 11 signals + 12 message handlers. |
| Frontend routing + utils | ~50 | YES. Minimal. |
| Frontend CSS (13 files) | ~400-500 | YES. Under 500 lines total as estimated. |
| **Frontend total** | **~1,150-1,350** | REASONABLE |
| **Grand total** | **~2,550-2,850** | WITHIN TARGET |

The original target was ~1,000 lines backend and ~800 lines frontend (code only, excluding tests and CSS). The actual estimates are slightly higher but include test code and CSS. Production code is well within the ballpark.

### 6.4 Dependency audit

| Dependency | Runtime? | Justified? |
|---|---|---|
| `nhooyr.io/websocket` | Yes (Go) | YES. Stdlib has no WebSocket. Only option besides archived gorilla. |
| `preact` | Yes (browser) | YES. ~4 KB. Reactive rendering without DOM spaghetti. |
| `@preact/signals` | Yes (browser) | YES. ~2 KB. Eliminates prop drilling for 11 signals. |
| `vite` | Dev only | YES. Industry-standard build tool. |
| `@preact/preset-vite` | Dev only | YES. Required Vite plugin. |
| `typescript` | Dev only | YES. Type safety for protocol types. |

**Total runtime dependencies: 3 (1 Go, 2 browser).** This is minimal. Nothing can be cut.

### 6.5 What could go wrong?

This section is speculative -- none of these are plan deficiencies, but things a developer should be aware of:

1. **WebSocket proxy stripping pings.** Some corporate proxies and older load balancers strip protocol-level WebSocket pings. The plans acknowledge this (architecture doc Section 4.6) and document the escape hatch: add application-level heartbeat later if needed. Good.

2. **`prefers-reduced-motion` blanket rule.** The global `* { animation-duration: 0.01ms !important }` rule is aggressive. It will prevent ALL CSS animations, not just the ones we define. If any future third-party component relies on CSS animations, they would be suppressed. At this scale (no third-party UI components), this is fine.

3. **No `localStorage` quota handling.** If `localStorage` is full or disabled (private browsing in some browsers), `setItem` will throw. The code sketches do not show try/catch. Minor -- the developer should wrap the calls.

These are implementation-level concerns, not plan deficiencies.

---

## Part 7: Final Actionable Items

### Must Fix (1 item)

1. **[BACKEND] [MUST] Fix phase string: `"revealed"` -> `"reveal"`.** Backend plan Section 3.1 uses `PhaseRevealed Phase = "revealed"`. Architecture doc and frontend plan use `"reveal"`. This is a wire-level mismatch that would cause the frontend to fail to detect the reveal phase. Change the backend plan constant to `PhaseReveal Phase = "reveal"`.

### Should Fix (2 items)

2. **[ARCHITECTURE] [SHOULD] Health endpoint path: `/health` -> `/api/health`.** Architecture doc Section 5 HTTP endpoints table says `/health`. Backend plan says `/api/health`. Update the architecture doc to match.

3. **[ARCHITECTURE] [SHOULD] `GET /api/rooms/{id}` response body.** Architecture doc says `{ "exists": true }`. Backend plan says `{ "id", "name", "phase", "participantCount" }`. Update the architecture doc to match the backend plan's richer response.

### That is it.

Three items, one of which is a real bug (the phase string). The other two are minor documentation inconsistencies on endpoints the frontend does not currently use.

---

## Part 8: Verdict

**The plans are ready to start coding.**

The v3.0 documents have resolved the fundamental problem identified in iteration 2: protocol specification drift across three documents. The architecture doc is now the single source of truth, and the frontend and backend plans reference it rather than redefining it. This structural change eliminated all 7 of the "MUST" items and most of the "SHOULD" items from the previous review in one pass.

What remains:
- One phase string typo (`"revealed"` vs `"reveal"`) in the backend plan -- a 10-second fix.
- Two minor endpoint documentation mismatches on routes the frontend does not call.

The architecture is clean, the protocol is consistent, the UX coverage is complete, the code volume is reasonable, and the dependency count is minimal. A developer can pick up these documents and start implementing without ambiguity.

**Recommendation: Fix the phase string in the backend plan, then begin implementation.**
