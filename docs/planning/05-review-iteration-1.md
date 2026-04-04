# om-scrum-poker: Planning Review -- Iteration 1

**Version:** 1.0
**Date:** 2026-04-04
**Reviewer Roles:** Product Owner / Chief Architect + Devil's Advocate
**Status:** Review Complete
**Documents Reviewed:**
- `01-ux-design.md` (UX Design)
- `02-frontend-plan.md` (Frontend Plan)
- `03-backend-plan.md` (Backend Plan)
- `04-architecture.md` (Architecture)

---

## Part 1: Product Owner / Chief Architect Review

### 1. Consistency Check

**Overall verdict: Good, with several contradictions that must be resolved.**

#### 1.1 Technology Choices -- Aligned

All documents agree on:
- **Backend:** Go with `nhooyr.io/websocket`
- **Frontend:** Preact with Vite
- **Deployment:** Single binary with embedded frontend
- **Storage:** In-memory only, no database
- **Identity:** localStorage-based `userName` + `sessionId` (UUID)

#### 1.2 CSS Approach -- CONTRADICTION

- **Frontend plan (02):** "Plain CSS with CSS Custom Properties" -- explicitly rejects CSS Modules, saying "With only ~15 components and disciplined BEM-like naming, class collisions are not a realistic risk."
- **Architecture doc (04), Section 2.5:** "CSS Modules (via Vite)" -- says each component gets a `.module.css` file with locally scoped class names.
- **Architecture doc (04), Section 10 (Project Structure):** Uses `.module.css` file names throughout (e.g., `home.module.css`, `card-deck.module.css`).

These directly contradict each other. The frontend plan argues BEM is sufficient; the architecture doc mandates CSS Modules. One must win. (See Devil's Advocate section for recommendation.)

#### 1.3 WebSocket Message Format -- CONTRADICTIONS

Multiple inconsistencies exist between the three documents that define WebSocket messages:

**Envelope structure:**
- **Backend plan (03):** Messages use `{ "type": "event_name", "payload": { ... } }` envelope with an explicit `payload` wrapper.
- **Frontend plan (02):** Outbound messages are flat -- `{ type: "join", sessionId: "...", userName: "...", roomId: "..." }` with no `payload` wrapper. Inbound messages are also flat.
- **Architecture doc (04), Section 4.1:** Uses `{ "type": "...", "payload": { ... } }` with the wrapper.
- **UX doc (01), Appendix A:** Uses flat format -- `{ sessionId, userName, roomId }` in the payload column, ambiguous about wrapper.

This must be resolved. Either all messages use an envelope or none do.

**Event naming discrepancies:**

| Concept | UX Doc (01) | Frontend (02) | Backend (03) | Architecture (04) |
|---|---|---|---|---|
| Un-vote | (not named) | `vote` with `value: ""` | `vote` with `value: ""` | `clear_vote` (separate type) |
| Vote cast ack | `vote_cast` | `vote_cast` | `vote_cast` + `vote_retracted` | `vote_cast` + `vote_cleared` |
| Room snapshot on join | (not defined) | `room_state` | `room_state` | `room_snapshot` |
| Name change | (pencil icon, broadcast) | `presence_update` with name | (not defined) | `update_name` (separate type) |
| Presence event name | `presence_update` / `presence_changed` | `presence_update` / `presence_changed` | `presence_update` / `presence_changed` | `presence` (client) / `presence_changed` (server) |
| Heartbeat | `heartbeat` (bidirectional) | `heartbeat` | `heartbeat` | `ping` / `pong` (app-level) + protocol-level |

These naming mismatches will cause bugs if developers implement from different docs. A single canonical message catalog is needed.

#### 1.4 WebSocket URL Path -- CONTRADICTION

- **Frontend plan (02):** `/ws/room/{roomId}` (singular "room")
- **Backend plan (03):** `/ws/rooms/{id}` (plural "rooms")
- **Architecture doc (04):** `/ws/{roomID}` (no "rooms" prefix)

Must be one canonical path.

#### 1.5 Room Creation Flow -- INCONSISTENCY

- **Frontend plan (02):** Room creation happens entirely client-side. The frontend generates the slug and UUID, then navigates directly to `/room/{slug}-{id}`. No API call. The room is implicitly created when the first WebSocket connection joins.
- **Backend plan (03):** Room creation is an explicit `POST /api/rooms` endpoint that returns the room ID.
- **Architecture doc (04):** Also shows `POST /api/rooms` for creation.

This is a significant disagreement. The frontend plan creates rooms lazily (on first WebSocket join), while the backend expects explicit creation via REST. This affects whether the backend needs the `POST /api/rooms` endpoint at all.

#### 1.6 Send Buffer Size -- MINOR INCONSISTENCY

- **Backend plan (03):** `sendBufferSize = 64`
- **Architecture doc (04):** `cap: 32`

Minor, but should be consistent.

#### 1.7 Vite Dev Server Port -- MINOR INCONSISTENCY

- **Frontend plan (02):** Dev server on port `3000`
- **Architecture doc (04):** Dev server on port `5173` (Vite default)

#### 1.8 `@preact/signals` Dependency

- **Frontend plan (02):** Lists `@preact/signals` as a runtime dependency (~2 KB).
- **Architecture doc (04), Section 2.4:** Does not list `@preact/signals` in the frontend dependency table -- lists only `preact` and `preact/hooks`, and says "State management (included in preact)." Signals are NOT included in the `preact` package; they are a separate package.

This means the architecture doc understates the dependency count.

#### 1.9 File Extensions -- TypeScript vs JavaScript

- **Frontend plan (02):** Uses `.tsx` and `.ts` extensions throughout. Explicitly chooses TypeScript.
- **Architecture doc (04), Section 10:** Uses `.jsx` and `.js` extensions throughout (e.g., `main.jsx`, `card-deck.jsx`, `use-websocket.js`). Also uses `vite.config.js` instead of `vite.config.ts`.

This is a meaningful contradiction. TypeScript provides type safety for the WebSocket protocol -- the frontend plan makes a strong case for it.

---

### 2. Completeness Check

**What from the UX spec is covered in technical plans?**

| UX Requirement | Frontend Plan | Backend Plan | Architecture |
|---|---|---|---|
| Room creation flow | Yes | Yes | Yes |
| Name entry modal | Yes | -- | Yes |
| Room page (voting + reveal) | Yes | Yes | Yes |
| Participant cards with presence | Yes | Yes | Yes |
| Card deck with all states | Yes | -- | Yes |
| Copy Link button | Yes | -- | Mentioned |
| Settings dropdown | Yes | -- | -- |
| Toast notifications | Yes | -- | Yes |
| Confirm dialog (Clear Room) | Yes | -- | Yes |
| Reconnection banner | Yes | -- | Yes |
| Statistics (avg, median, consensus) | Yes | Yes | Yes |
| Consensus/spread highlight | Yes | Yes (in stats) | Yes |
| "?" vote handling | Yes | Yes | Yes |
| Presence states (active/idle/disconnected) | Yes | Yes | Yes |
| Reconnection with vote preservation | Yes | Yes | Yes |
| Room GC (24h) | -- | Yes | Yes |
| Rate limiting | -- | Yes | Yes |
| Responsive design (breakpoints, touch targets) | Yes | -- | Partial |
| Accessibility (ARIA, keyboard, reduced motion) | Yes | -- | -- |
| Room not found page | Yes | Yes | Yes |
| Duplicate names allowed | Implicit | Yes | Yes |
| Browser back/forward | Yes | -- | -- |

**Gaps identified:**

1. **Accessibility:** The UX doc has detailed accessibility requirements (Appendix C): keyboard navigation, screen reader announcements, ARIA roles, WCAG AA contrast. The frontend plan covers some ARIA attributes on individual components (Card, ConfirmDialog) but there is no systematic accessibility plan. The backend and architecture docs ignore accessibility entirely (which is correct -- it is a frontend concern). The frontend plan should have a dedicated accessibility section mapping each UX accessibility requirement to implementation.

2. **Name change broadcast:** The UX doc specifies that changing a name "broadcasts update to all participants." The frontend plan mentions this via `presence_update`, but neither the backend plan nor the architecture doc defines a clear event for name changes. The architecture doc has an `update_name` / `name_updated` event pair, but the backend plan does not mention it. The frontend plan routes it through `presence_update`, which is semantically wrong.

3. **"Leave room" behavior:** The UX doc's settings dropdown has "Leave room." The frontend plan's SettingsDropdown covers this (disconnect WebSocket, navigate to `/`). But neither backend doc specifies a `leave` event distinct from a disconnect. Should "Leave room" send a specific event so the server can immediately remove the participant (vs. waiting for heartbeat timeout)?

4. **Toast content specification:** The UX doc lists specific toast messages: "Link copied!", "Room cleared", "[Name] joined", "[Name] left", "New round started", "Reconnected." The frontend plan implements some of these in the handler code but does not systematically list them. Missing from the handler code: "Room cleared", "Reconnected", "[Name] left".

---

### 3. Architecture Alignment

**Frontend plan alignment with architecture:**

The frontend plan is largely consistent with the architecture doc's frontend section, with these exceptions:
- CSS approach (plain CSS vs CSS Modules) -- as noted above.
- File extensions (.tsx vs .jsx) -- as noted above.
- Component organization: The frontend plan has a flat `components/` directory with subdirectories per component. The architecture doc has `pages/` and `components/` as separate directories. Both are fine, but they should agree.
- The frontend plan uses `@preact/signals` for state; the architecture doc mentions hooks-based state in some places and signals in others.

**Backend plan alignment with architecture:**

The backend plan and architecture doc are well-aligned on the overall structure (domain/app/infra/handler layers), but differ in details:
- The backend plan uses `internal/room/manager.go` with a `Manager` struct that combines room storage and WebSocket client tracking. The architecture doc separates these into `RoomStore` (infra/store) and `Hub` (infra/hub) with distinct interfaces. The architecture doc's separation is cleaner.
- The backend plan's project structure has `internal/room/`, `internal/ws/`, `internal/api/`, `internal/presence/`, `internal/ratelimit/`. The architecture doc has `internal/domain/`, `internal/app/`, `internal/infra/`, `internal/handler/`, `internal/config/`. These are different package organizations. The architecture doc follows a Clean Architecture style; the backend plan follows a feature-based grouping. They cannot both be correct.
- The backend plan introduces `golang.org/x/time/rate` in the architecture doc's rate limiting section, adding a third Go dependency. The backend plan's own dependency list says "Three dependencies total" (nhooyr/websocket + google/uuid + presumably x/time/rate). The architecture doc says "Two dependencies total." This needs reconciliation.

**Key alignment finding:** The backend plan and architecture doc essentially describe two different package structures. This must be resolved before development begins, or developers will be confused about where to put code.

---

### 4. Missing Pieces

1. **Error handling on the frontend.** The backend plan defines structured error responses (`{ error: { code, message } }`) and WebSocket error events. The frontend plan does not describe how these are handled. What happens when the server sends an `error` event? Is it shown as a toast? Logged? Ignored? The `handleMessage` switch statement in the frontend plan does not include a case for `"error"`.

2. **Name change protocol.** As noted in Section 2, the full round-trip for name change (client sends new name, server validates, server broadcasts to all, all clients update participant list) is not fully specified in any single document.

3. **"Leave room" vs disconnect semantics.** When a user clicks "Leave room," should the server treat this differently from a tab close? The UX doc implies they are the same (no "are you sure" prompt, no special behavior). But a deliberate leave could immediately remove the participant, whereas a disconnect keeps them dimmed. This behavioral distinction is not specified.

4. **SPA fallback routing.** Both the frontend and architecture docs mention that the Go server must serve `index.html` for any path not matching `/api/*` or `/ws/*`. The backend plan does not mention this SPA fallback behavior.

5. **Vote value handling for `0.5`.** The backend plan uses `VoteValue string` type and validates against a set. But the slug generation uses `0.5` as a string. The frontend `generateRoomUrl` regex strips non-alphanumeric characters except hyphens -- the dot in "0.5" would be stripped. This is fine because room names and vote values are unrelated, but it is worth noting that `0.5` serialization should be tested.

6. **Max WebSocket message size.** The architecture doc specifies 1 KB max. The backend plan does not mention this limit. They should agree, and the frontend should be aware of it.

7. **Testing plan for the frontend.** The architecture doc mentions Vitest + Preact Testing Library but the frontend plan has no testing section at all. For a real-time collaborative app, the WebSocket reconnection logic and state management are the highest-risk areas and need test coverage defined upfront.

---

### 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| WebSocket message format disagreements between FE and BE developers | High (contradictions exist now) | High (integration failures) | Resolve all message format contradictions before coding. Create a single canonical message catalog in a shared location. |
| Reconnection logic bugs | Medium | High (users lose votes, see stale state) | Extensive testing of reconnection scenarios: reconnect during voting, during reveal, after new round, after clear room. |
| Mobile browser address bar eating viewport height | Medium | Medium (card deck hidden) | The frontend plan correctly uses `100dvh`. Test on real devices. |
| Race conditions in concurrent room mutations | Low (mutexes are well-designed) | High (corrupted state) | Run Go tests with `-race` flag (already planned). Keep critical sections short (already planned). |
| Proxy/CDN stripping WebSocket frames | Low | High (presence breaks) | The architecture doc's dual ping/pong strategy (protocol + app level) mitigates this well. |
| Memory leak from abandoned rooms | Low | Low (GC handles it) | GC is well-specified. Monitor room count via health endpoint. |
| Browser localStorage quota or private mode | Low | Low (graceful degradation) | Not addressed. `sessionId` and `userName` are tiny. Should work even in private mode. Worth a note in frontend plan. |

---

## Part 2: Devil's Advocate Review

### 1. Over-engineering Alerts

#### 1.1 The Architecture Doc Has Three Layers Too Many

The architecture doc defines four server layers: Presentation, Application, Domain, Infrastructure. For a service with:
- 1 entity (Room, which contains Participants)
- 6 operations (create, join, vote, reveal, new round, clear)
- 0 database calls
- 0 external service calls

...a Clean Architecture with `RoomStore` interfaces, `Hub` interfaces, `RoomService` use cases, `RoomSnapshot` DTOs, and `ParticipantSnapshot` DTOs is ceremony without payoff.

**The Application Layer adds nothing.** Every method on `RoomService` does: (1) call a domain method, (2) broadcast. This is a trivial delegation that could live in the WebSocket handler itself. The `RoomService` exists to satisfy the pattern, not a real need.

When you have a single in-memory store and will never swap it for a database (the doc explicitly says persistence violates KISS), coding to a `RoomStore` interface is premature abstraction. You are writing an interface with exactly one implementation that will never have a second.

**Recommended structure:** Two packages. `domain` (Room, Participant, stats calculation -- pure logic) and `server` (HTTP handler, WebSocket handler, in-memory store, broadcast). That is it. The WebSocket handler calls domain methods directly and broadcasts. No interfaces, no DTOs, no service layer.

#### 1.2 Frontend Component Count is Excessive

The frontend plan lists 16 components. Several should be merged:

- **CopyLinkButton:** This is a `<button>` with 5 lines of clipboard logic. It does not need its own component + CSS file. Inline it in `Header`.
- **SettingsDropdown:** A gear icon that opens two menu items. Inline it in `Header`.
- **ActionButtons:** This is a layout container for three buttons. The buttons themselves are `<button>` elements. The "Show Votes" button could be part of `RoomPage` directly. "New Round" and "Clear Room" are one-liners.
- **ReconnectionBanner:** A conditional `<div>` with text. 3 lines of JSX. Inline in `RoomPage`.
- **Statistics:** A `<div>` showing 3-4 computed values. Could be part of `RoomPage` or `ParticipantList`.

**Recommended component count: 8-10.** `App`, `HomePage`, `RoomPage`, `NameEntryModal`, `ParticipantList`, `ParticipantCard`, `CardDeck`, `Card`, `Toast`, `ConfirmDialog`. Everything else is a `<div>` or a `<button>` that does not warrant its own file.

#### 1.3 Three State Modules is Overkill

The frontend plan splits state into `state/room.ts`, `state/connection.ts`, `state/ui.ts`. The total state for this app is approximately:
- `roomId`, `roomName`, `phase`, `participants`, `revealedVotes` (room)
- `connectionStatus`, `reconnectAttempts` (connection)
- `selectedCard`, `toasts`, `confirmDialog`, `showNameModal`, `nameModalMode` (UI)

That is 11 signals. Put them in one file. Three files with 3-4 signals each create cross-file import gymnastics for no benefit. A single `state.ts` is perfectly readable at this size.

#### 1.4 The `hooks/` Directory Has Marginal Hooks

- **`useClickOutside.ts`:** A legitimate reusable hook, but used only by SettingsDropdown. If SettingsDropdown is inlined into Header (as recommended), this hook is called once. Just write the 8 lines of `addEventListener`/`removeEventListener` inline.
- **`usePresence.ts`:** Legitimate. Keeps presence tracking logic separated.
- **`useWebSocket.ts`:** The frontend plan also has `ws/client.ts`. Is the hook a wrapper around the client? If so, the hook is the unnecessary layer -- the client module already manages the lifecycle. Components can import `wsConnect`/`wsSend`/`wsDisconnect` directly.

#### 1.5 The `utils/` Directory

- **`clipboard.ts`:** A wrapper around `navigator.clipboard.writeText()` with a fallback. This is 10 lines of code used in one place. Inline it.
- **`room-url.ts`:** Slug generation + room ID parsing. Legitimate utility (used by both HomePage and RoomPage).
- **`stats.ts`:** Statistics calculation. Legitimate utility.

Two of three utils are justified. The third is a single-use helper.

#### 1.6 Backend Presence Tracker as a Separate Package

The `internal/presence/` package with its own `Tracker` struct, `sessionState`, `graceTimer`, and check loop is a separate subsystem for tracking 3 states per participant. The room already has participants with a `PresenceStatus` field. The tracker duplicates this data in a parallel data structure.

**Simpler approach:** Presence is a property of a Participant in a Room. The WebSocket handler updates `Participant.Status` and `Participant.LastSeen` directly when it receives presence events or detects disconnection. A single goroutine per room (or a global ticker) checks `LastSeen` and marks participants as disconnected. No separate package, no separate data structure, no synchronization between two representations of the same state.

#### 1.7 Rate Limiter as a Separate Package

The `internal/ratelimit/` package implements a token bucket rate limiter. The architecture doc also suggests using `golang.org/x/time/rate` (a Go sub-repository). Pick one:
- If using `golang.org/x/time/rate`, you need approximately 20 lines of middleware code, not a package.
- If hand-rolling, the backend plan's implementation is ~50 lines.

Either way, this does not need its own package. It is middleware that wraps a handler. Put it in the handler package or as a top-level middleware file.

---

### 2. Dependency Audit

#### 2.1 Backend Dependencies

| Dependency | Justified? | Verdict |
|---|---|---|
| `nhooyr.io/websocket` | Yes | stdlib does not include WebSocket. This is the right choice. |
| `github.com/google/uuid` | Questionable | Used only to generate 8-char hex suffixes for room IDs. `crypto/rand` + hex encoding achieves the same in 4 lines: `b := make([]byte, 4); rand.Read(b); hex.EncodeToString(b)`. Session IDs (from clients) are generated by the browser's `crypto.randomUUID()` and only need format validation on the server, not generation. Eliminating this dependency is trivial. |
| `golang.org/x/time/rate` (architecture doc only) | Questionable | A token bucket is ~30 lines of code. For two rate limits (room creation, WS upgrade), bringing in a dependency is debatable. However, `x/time/rate` is an official Go sub-repository with a well-tested, correct implementation. The call is close. If the team is comfortable writing their own (the backend plan already does), skip it. |

**Verdict:** The backend can run with 1 dependency (`nhooyr.io/websocket`). UUID generation is replaceable with stdlib. Rate limiting is replaceable with a small hand-rolled implementation (already written in the backend plan).

#### 2.2 Frontend Dependencies

| Dependency | Justified? | Verdict |
|---|---|---|
| `preact` | Yes | The reactive UI complexity (participant list mutations, presence transitions, phase changes) justifies a small framework. Vanilla JS would be messier. |
| `@preact/signals` | Questionable | Preact already ships with hooks (`useState`, `useReducer`). Signals provide finer-grained reactivity, but for 11 pieces of state and ~10 components, the difference is negligible. Using `useState` in a top-level component with prop-passing (the component tree is max 3 levels deep) avoids an additional dependency. Counter-argument: Signals are ~2 KB and simplify the code by eliminating prop drilling. Acceptable either way. |

**Verdict:** 1-2 runtime dependencies is excellent. No bloat here.

---

### 3. Code Volume Estimate

**Backend:**

The backend plan + architecture doc describe roughly:
- `domain/`: ~200 lines (Room, Participant, stats, validation)
- `app/` (service): ~150 lines (if kept; recommend eliminating)
- `infra/` (store, hub, GC): ~250 lines
- `handler/` (HTTP, WS, middleware): ~350 lines
- `config/`: ~50 lines
- `main.go`: ~50 lines

**Estimated total: ~1000-1100 lines of Go** (excluding tests).

This is appropriate for the problem. If the service layer is eliminated, it drops to ~900 lines.

**Frontend:**

- Components (10-16 depending on merges): ~800-1200 lines of TSX
- State management: ~80 lines
- WebSocket client + handlers: ~150 lines
- Utilities: ~50 lines
- CSS: ~400-500 lines

**Estimated total: ~1500-2000 lines of TS/TSX + ~500 lines of CSS.**

This is appropriate. The higher end (2000 lines) reflects the 16-component plan; the lower end (1500) reflects the recommended 10-component plan.

**Total project: ~2500-3000 lines of code.** This is a healthy size for a real-time collaborative tool with reconnection, presence, and responsive design. Not bloated, not anemic.

---

### 4. Simplification Proposals

#### 4.1 Flatten the Backend to 3 Packages

Replace the 6-package backend (`domain`, `app`, `infra`, `handler`, `config`, `cmd/server`) with 3:
- `domain`: Room, Participant, stats, vote validation. Pure logic. (~200 lines)
- `server`: HTTP handler, WebSocket handler, in-memory store, broadcast hub, rate limit middleware, GC ticker, config parsing. All I/O concerns. (~700 lines)
- `main.go`: Wire and start. (~50 lines)

The `app` layer is eliminated. The `infra`, `handler`, `config`, and `ratelimit` packages merge into `server`. Interfaces are eliminated because there is exactly one implementation of everything.

**What you lose:** The ability to swap storage backends without touching the WebSocket handler.
**What you gain:** ~30% fewer files, no interface ceremony, one less layer of indirection, faster navigation for developers and AI assistants alike.

#### 4.2 Merge Small Frontend Components

Merge `CopyLinkButton`, `SettingsDropdown`, and `ReconnectionBanner` into their parent components. Merge `ActionButtons` into `RoomPage`. This reduces the component count from 16 to 10 and eliminates 6 CSS files.

#### 4.3 Unify State into a Single File

Merge `state/room.ts`, `state/connection.ts`, `state/ui.ts`, and `state/storage.ts` into a single `state.ts`. The total state fits comfortably in one file (~80 lines).

#### 4.4 Eliminate the `types/` Directory

The frontend plan has `types/room.ts`, `types/ws-messages.ts`, `types/ui.ts`. Move type definitions to the files that use them. `Participant` and `Vote` types go in `state.ts`. WebSocket message types go in `ws/client.ts` (or a single `ws.ts`). UI types like `ToastItem` are 3-line interfaces -- define them next to the `toasts` signal.

With ~10 types total, a dedicated `types/` directory with 3 files is organizational overhead.

#### 4.5 Drop the REST API for Room Creation

Consider the frontend plan's approach: the client generates the room slug + UUID locally and navigates directly to `/room/{slug}-{id}`. The room is created implicitly on the first WebSocket `join`. This eliminates:
- `POST /api/rooms` endpoint
- The HTTP room creation handler
- An HTTP round-trip before the WebSocket connection
- The `GET /api/rooms/{id}` pre-check (the WebSocket `join` can return an error if the room does not exist, or create it)

The only benefit of the REST endpoint is server-side slug generation and validation. But the client already validates the room name (same regex). Slug generation is deterministic. The only thing the server adds is the UUID suffix, which the client can also generate via `crypto.randomUUID().slice(0, 8)`.

**Trade-off:** Without `POST /api/rooms`, a room exists only when someone is connected to it. This matches the in-memory-only philosophy. The `GET /api/rooms/{id}` endpoint can be kept as a lightweight room-existence check if desired.

#### 4.6 Simplify Presence to Two States

The UX doc defines Active, Idle, and Disconnected. But consider:
- **Active vs Idle:** The only visible difference is a green vs yellow dot. The behavioral difference is zero -- idle users' votes still count, they are still in the participant list, they are still counted in "X of Y voted."
- **The value of Idle:** It tells the team "Bob has switched tabs." This is marginally useful but adds: (1) client-side visibility/activity tracking, (2) `presence_update` messages on every tab switch, (3) debounced activity listeners on `mousemove`/`keydown`/`touchstart`, (4) a 2-minute inactivity timer.

This is a judgment call. The Idle state is specified in the UX doc and serves a real purpose ("should we wait for Bob?"). But if simplicity is paramount, Active + Disconnected (two states) eliminates all client-side activity tracking. The presence dot is green when connected, red when disconnected. Done.

**Recommendation:** Keep all three states as specified. The implementation cost is modest (one hook, one event type), and the UX value is real. But acknowledge this is the most complex piece of the presence system and test it well.

---

### 5. Anti-patterns

#### 5.1 Parallel Data Structures for Presence

The backend plan has presence state in both `Participant.Status` (in the Room) and `presence.Tracker.sessions[sessionId].status`. Two copies of the same data that must be kept in sync is a classic consistency bug waiting to happen. The tracker should either BE the source of truth (and Participant reads from it) or not exist (and Participant.Status is updated directly).

#### 5.2 Interface-Driven Design Without Multiple Implementations

`RoomStore` interface with one implementation (`memoryStore`). `Hub` interface with one implementation. These interfaces exist for testability, which is a valid reason. But Go's implicit interface satisfaction means you can define the interface in the test file and have the concrete type satisfy it without declaring it upfront. This is more idiomatic: define interfaces where they are consumed, not where they are produced.

If the service layer is eliminated (as recommended), testability is achieved by testing the domain layer directly (it is pure logic) and testing the handlers with real in-memory stores (integration-style, which is more valuable than mock-based unit tests for this size of project).

#### 5.3 Over-specified Configuration

The architecture doc defines 10 environment variables including `HEARTBEAT_INTERVAL`, `HEARTBEAT_TIMEOUT`, `MAX_CONNECTIONS`, `MAX_ROOM_CONNECTIONS`, `LOG_LEVEL`, and `GC_INTERVAL`. For a self-hosted tool aimed at teams of 5-15 people, who is tuning heartbeat intervals?

**Recommendation:** Start with 3 config variables: `PORT`, `HOST`, `TRUST_PROXY`. Everything else gets sensible hardcoded defaults. Add configurability when someone asks for it, not before.

#### 5.4 Dual Heartbeat Mechanism

The architecture doc implements BOTH protocol-level WebSocket pings AND application-level ping/pong messages. The justification ("some proxies strip protocol pings") is valid in theory but rare in practice. Modern reverse proxies (nginx, Caddy, Traefik, Cloudflare) all pass WebSocket pings through.

**Recommendation:** Use protocol-level pings only (`nhooyr.io/websocket` supports them natively). Add application-level pings later if users report issues with specific proxies. This eliminates two message types from the protocol and simplifies both client and server.

---

## Part 3: Actionable Items

### Message Protocol and Consistency

1. **[ALL] [MUST]** Create a single canonical WebSocket message catalog in a shared document (e.g., `docs/API.md`). Resolve all naming discrepancies: `room_state` vs `room_snapshot`, `vote` with empty value vs `clear_vote`, `presence_update` vs `presence`, `heartbeat` vs `ping/pong`. All three technical plans must reference this single source of truth.

2. **[ALL] [MUST]** Decide on envelope format: either all messages use `{ type, payload }` or all messages are flat `{ type, ...fields }`. Apply consistently to both client-to-server and server-to-client messages. Recommend flat format -- it is simpler and avoids unnecessary nesting for small payloads.

3. **[ALL] [MUST]** Agree on a single WebSocket URL path. Recommend `/ws/{roomId}` (simplest).

4. **[ALL] [MUST]** Decide whether room creation is client-side (frontend generates slug + UUID, room created implicitly on first join) or server-side (`POST /api/rooms`). If client-side: remove `POST /api/rooms` from backend and architecture docs. If server-side: update frontend plan to make the API call before navigating.

5. **[FRONTEND] [MUST]** Decide TypeScript (.tsx) or JavaScript (.jsx). Recommend TypeScript as the frontend plan argues. Update architecture doc Section 10 to use .ts/.tsx extensions.

6. **[ALL] [MUST]** Decide CSS approach: plain CSS with BEM naming or CSS Modules. Recommend plain CSS with BEM -- it is simpler, matches the frontend plan's reasoning, and avoids Vite CSS Module configuration. Update whichever doc loses.

### Backend Simplification

7. **[BACKEND] [SHOULD]** Eliminate the Application Layer (`internal/app/`). Move the orchestration logic (call domain method + broadcast) directly into the WebSocket handler. The service layer adds indirection without value for a project with one storage implementation and no external service calls.

8. **[BACKEND] [SHOULD]** Merge `internal/presence/`, `internal/ratelimit/`, and `internal/config/` into the main server package. These are small, single-purpose utilities that do not justify separate packages. The final backend structure should be: `domain/`, `server/` (or `internal/server/`), `cmd/server/main.go`.

9. **[BACKEND] [SHOULD]** Eliminate `github.com/google/uuid`. Use `crypto/rand` + `encoding/hex` for generating 8-char room ID suffixes. Session IDs come from clients and only need validation, not generation.

10. **[BACKEND] [SHOULD]** Remove the parallel presence data structure. Presence status should live only on `Participant.Status` in the Room, updated directly by the WebSocket handler on connect/disconnect and by presence events from the client.

11. **[BACKEND] [COULD]** Reduce environment variables to `PORT`, `HOST`, and `TRUST_PROXY`. Hardcode all other values with sensible defaults. Add configuration later when a real user requests it.

### Frontend Simplification

12. **[FRONTEND] [SHOULD]** Merge `CopyLinkButton` and `SettingsDropdown` into `Header`. Merge `ReconnectionBanner` into `RoomPage`. Merge `ActionButtons` into `RoomPage`. Target 10 components instead of 16.

13. **[FRONTEND] [SHOULD]** Consolidate `state/room.ts`, `state/connection.ts`, `state/ui.ts`, and `state/storage.ts` into a single `state.ts` file. Eleven signals fit comfortably in one file.

14. **[FRONTEND] [SHOULD]** Eliminate the `types/` directory. Co-locate the ~10 type definitions with the code that uses them (`state.ts` and `ws/client.ts`).

15. **[FRONTEND] [SHOULD]** Eliminate `utils/clipboard.ts`. Inline the 10 lines of clipboard logic into the Header component (or wherever Copy Link lives after merging).

16. **[FRONTEND] [COULD]** Evaluate whether `@preact/signals` is justified vs plain hooks (`useState` + props). For 10 components with a max tree depth of 3, prop-passing is trivial. If signals are kept (reasonable), document the trade-off.

### Missing Specifications

17. **[FRONTEND] [MUST]** Add error handling for WebSocket `error` events from the server. At minimum, show the error message as a toast. Add the `"error"` case to the `handleMessage` switch.

18. **[ALL] [MUST]** Specify the name-change protocol end-to-end: client sends `update_name` (or whatever it is called after item 1), server validates, server broadcasts `name_updated`, all clients update the participant list. Add this to the canonical message catalog.

19. **[FRONTEND] [SHOULD]** Add a section on accessibility implementation, mapping UX doc Appendix C requirements to specific implementation details: focus management in modals, ARIA attributes on interactive elements, screen reader announcements for state changes, `prefers-reduced-motion` support.

20. **[BACKEND] [SHOULD]** Specify explicit "leave room" behavior. When a user clicks "Leave room," the client should send a `leave` event (or close the WebSocket with a specific close code like 1000). The server should immediately remove the participant from the room (no grace period, no dimmed state). This differs from an unexpected disconnect.

21. **[FRONTEND] [SHOULD]** Add a frontend testing plan. At minimum: what testing framework (Vitest is mentioned in the architecture doc), what to test (WebSocket reconnection state machine, statistics calculation, component rendering per phase), and what NOT to test (pure CSS, trivial components).

### Architecture Doc Updates

22. **[ARCHITECTURE] [MUST]** Reconcile the project structure in Section 10 with the backend plan's project structure. They describe two different package organizations. Pick one and update the other.

23. **[ARCHITECTURE] [SHOULD]** Update Section 2.4 (Frontend Dependencies) to include `@preact/signals` if it is a chosen dependency. It is not part of the `preact` package.

24. **[ARCHITECTURE] [COULD]** Simplify the heartbeat mechanism. Start with protocol-level WebSocket pings only (supported by `nhooyr.io/websocket`). Remove application-level `ping`/`pong` messages from the protocol. Add them back only if proxy issues are reported in production.

25. **[ARCHITECTURE] [COULD]** Reduce the layer diagram from 4 layers (Presentation, Application, Domain, Infrastructure) to 2 (Domain, Server) to match the recommended simplified structure.

---

## Summary

The planning documents are thorough and demonstrate strong engineering judgment. The core design decisions -- Go backend, Preact frontend, in-memory storage, WebSocket-first communication, single binary deployment -- are sound and well-justified.

The primary issues are:
1. **Consistency:** Multiple contradictions between documents on message formats, event names, URL paths, CSS approach, file extensions, and project structure. These must be resolved before development begins.
2. **Over-engineering:** The backend has too many layers and packages for a service this small. The frontend has too many components for the UI complexity. Both can be simplified without losing any capability.
3. **Missing specs:** Name change protocol, error handling on frontend, accessibility implementation, and frontend testing plan need to be added.

None of these issues are architectural showstoppers. The fundamental design is correct. The fixes are about removing unnecessary complexity, resolving contradictions, and filling gaps -- not rethinking the approach.
