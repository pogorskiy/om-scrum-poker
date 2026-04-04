# om-scrum-poker: Planning Review -- Iteration 2

**Version:** 1.0
**Date:** 2026-04-04
**Reviewer Roles:** Product Owner / Chief Architect + Devil's Advocate
**Status:** Review Complete
**Documents Reviewed (all v2.0):**
- `01-ux-design.md` (UX Design, v1.0 -- unchanged)
- `02-frontend-plan.md` (Frontend Plan, v2.0)
- `03-backend-plan.md` (Backend Plan, v2.0)
- `04-architecture.md` (Architecture, v2.0)
- `05-review-iteration-1.md` (Previous review -- to verify items addressed)

---

## Part 1: Iteration 1 Follow-up

### Were all 25 actionable items addressed?

The v2.0 documents show significant improvement. Here is the status of each item from the iteration 1 review:

| # | Item | Status | Notes |
|---|---|---|---|
| 1 | [ALL] [MUST] Create canonical WebSocket message catalog | ADDRESSED | Architecture doc Section 4 is now "THE single source of truth." Backend plan Section 5 also defines a complete protocol. Frontend plan references the backend plan. |
| 2 | [ALL] [MUST] Decide envelope format | PARTIALLY ADDRESSED | Architecture doc chose flat `{ type, ...fields }`. Backend plan chose `{ type, payload }` envelope. Frontend plan chose `{ type, payload }` envelope. **Still contradicts.** See Consistency Check below. |
| 3 | [ALL] [MUST] Agree on WebSocket URL path | PARTIALLY ADDRESSED | Backend plan and frontend plan both use `/ws/room/:id`. Architecture doc uses `/ws/{roomId}` (no "room" segment). **Still contradicts.** |
| 4 | [ALL] [MUST] Room creation: client-side vs server-side | ADDRESSED | All three docs agree: client-side room creation, implicit on first WebSocket join. `POST /api/rooms` is removed. |
| 5 | [FRONTEND] [MUST] Decide TypeScript vs JavaScript | ADDRESSED | All docs use `.ts` / `.tsx`. |
| 6 | [ALL] [MUST] Decide CSS approach | ADDRESSED | All docs agree on plain CSS with BEM-like naming. |
| 7 | [BACKEND] [SHOULD] Eliminate Application Layer | ADDRESSED | Backend collapsed to 2 packages: `domain` + `server`. |
| 8 | [BACKEND] [SHOULD] Merge small packages into server | ADDRESSED | `presence/`, `ratelimit/`, `config/` merged into `server`. |
| 9 | [BACKEND] [SHOULD] Eliminate `google/uuid` | ADDRESSED | Uses `crypto/rand` + `encoding/hex`. Single external dependency. |
| 10 | [BACKEND] [SHOULD] Remove parallel presence data structure | ADDRESSED | Presence lives on `Participant.Status`. No separate tracker. |
| 11 | [BACKEND] [COULD] Reduce env vars to 3 | ADDRESSED | `PORT`, `HOST`, `TRUST_PROXY` only. |
| 12 | [FRONTEND] [SHOULD] Merge small components | ADDRESSED | Down from 16 to 10 components. |
| 13 | [FRONTEND] [SHOULD] Consolidate state into single file | ADDRESSED | Single `state.ts` with all signals. |
| 14 | [FRONTEND] [SHOULD] Eliminate `types/` directory | ADDRESSED | Types co-located in `state.ts` and `ws.ts`. |
| 15 | [FRONTEND] [SHOULD] Eliminate `utils/clipboard.ts` | ADDRESSED | Inlined into Header. |
| 16 | [FRONTEND] [COULD] Evaluate signals vs hooks | ADDRESSED | Decision documented with explicit trade-off note. |
| 17 | [FRONTEND] [MUST] Add error handling for WS errors | ADDRESSED | `error` case added to `handleMessage` switch. Error codes documented. |
| 18 | [ALL] [MUST] Specify name-change protocol | ADDRESSED | Frontend plan Section 5.6 has full end-to-end specification. Backend plan Section 5 includes `update_name` / `name_changed`. |
| 19 | [FRONTEND] [SHOULD] Add accessibility section | ADDRESSED | Frontend plan Section 8 maps all UX doc Appendix C requirements. |
| 20 | [BACKEND] [SHOULD] Specify "leave" vs "disconnect" | ADDRESSED | Backend plan Section 6.2 has clear distinction. Architecture doc Section 4.3 includes `leave` event. |
| 21 | [FRONTEND] [SHOULD] Add frontend testing plan | PARTIALLY ADDRESSED | Architecture doc Section 11.4 has a testing strategy table. Frontend plan itself does not have a dedicated testing section, but the architecture doc covers it. Acceptable. |
| 22 | [ARCHITECTURE] [MUST] Reconcile project structure | ADDRESSED | Architecture and backend plan now both show `domain/` + `server/` 2-package structure. |
| 23 | [ARCHITECTURE] [SHOULD] Add `@preact/signals` to dependency table | ADDRESSED | Architecture doc Section 2.2 lists it explicitly as a separate package. |
| 24 | [ARCHITECTURE] [COULD] Simplify heartbeat | ADDRESSED | Protocol-level pings only. No application-level `ping`/`pong`. |
| 25 | [ARCHITECTURE] [COULD] Reduce layer diagram to 2 layers | ADDRESSED | Architecture doc Section 1.2 shows `domain` + `server`, no intermediate layers. |

**Summary: 22 of 25 fully addressed, 3 partially addressed.** The partially addressed items are the remaining envelope format contradiction (#2), WebSocket URL path mismatch (#3), and testing plan location (#21). Items #2 and #3 are significant and must be fixed.

---

## Part 2: Consistency Check

The v2.0 plans are vastly more consistent than v1.0. Most contradictions have been resolved. However, several remain.

### 2.1 WebSocket Message Envelope Format -- CONTRADICTION

This is the most important remaining inconsistency. The iteration 1 review specifically asked for a single format, and the three documents chose differently:

| Document | Format | Evidence |
|---|---|---|
| **Architecture doc (04)** Section 4.2 | **Flat** `{ type, ...fields }` | "All messages are flat JSON objects with a `type` field for dispatch and remaining fields as data. No `payload` wrapper." |
| **Architecture doc (04)** Sections 4.4-4.5, 5.4 | **Envelope** `{ type, payload }` | The `room_state` example in Section 4.4 uses `{ "type": "room_state", "payload": { ... } }`. The `votes_revealed` example in Section 4.5 also uses `payload`. |
| **Backend plan (03)** Section 5 | **Envelope** `{ type, payload }` | Section 5.1: `"type": "event_name", "payload": { ... }`. All examples use `payload`. |
| **Backend plan (03)** Section 4.4 | **Flat** `{ type, ...fields }` | The `room_state` example has `"type": "room_state", "roomId": "...", "roomName": "..."` -- no payload wrapper. Similarly `votes_revealed` is flat. The `error` event is flat: `{ "type": "error", "message": "..." }`. |
| **Frontend plan (02)** Section 5.4 | **Envelope** `{ type, payload }` | All OutboundMessage and InboundMessage types use `payload`. |
| **Architecture doc (04)** CLAUDE.md template, Section 10.2 | **Flat** | `"All WebSocket messages are flat JSON: { type, ...fields }. No payload wrapper."` |
| **Architecture doc (04)** Decision Log | **Flat** | `"Flat message format | { type, payload } envelope | Simpler. No unnecessary nesting."` |

**The architecture doc contradicts itself.** The changelog says "Unified envelope format: flat" and the CLAUDE.md template says "flat", but the protocol specification sections (4.3-4.5) consistently use `{ type, payload }`. The backend plan is similarly split: Section 5 uses envelopes, Section 4.4 uses flat format.

**Verdict:** Two of the three documents use `{ type, payload }` in their detailed protocol sections. The flat format declarations appear to be changelog/summary text that was not reconciled with the actual protocol examples. The codebase will follow whatever the detailed examples show. This needs one clean decision and a pass through all examples.

### 2.2 WebSocket URL Path -- CONTRADICTION

| Document | Path |
|---|---|
| **Frontend plan (02)** | `/ws/room/:id` (Section 5.2, 5.9, and Vite proxy config) |
| **Backend plan (03)** | `/ws/room/:id` (Section 4.2, 6.1, 6.3, 9.2) |
| **Architecture doc (04)** | `/ws/{roomId}` (Sections 4.1, 5.1) |
| **Architecture doc (04)** | `/ws/{roomId}` in HTTP endpoints table (Section 5) |

The frontend and backend plans agree on `/ws/room/:id`. The architecture doc uses `/ws/{roomId}` (no "room" segment). The architecture doc must be updated to match.

### 2.3 Event Name Discrepancy: `name_changed` vs `name_updated`

| Document | Client sends | Server broadcasts |
|---|---|---|
| **Frontend plan (02)** Section 5.4 | `update_name` | `name_updated` |
| **Backend plan (03)** Section 5.2-5.3 | `update_name` | `name_changed` |
| **Architecture doc (04)** Section 4.3-4.4 | `update_name` | `name_updated` |

The backend plan uses `name_changed` for the server broadcast. The frontend plan and architecture doc use `name_updated`. Must agree on one name.

### 2.4 Presence Client Event Name Discrepancy

| Document | Client sends |
|---|---|
| **Frontend plan (02)** Section 5.4 | `presence_update` |
| **Backend plan (03)** Section 5.2 | `presence` |
| **Architecture doc (04)** Section 4.3 | `presence` |

The frontend plan uses `presence_update` while the other two use `presence`. The backend plan and architecture doc agree, so the frontend plan should change.

### 2.5 Heartbeat: Application-Level vs Protocol-Level -- CONTRADICTION

| Document | Approach |
|---|---|
| **Backend plan (03)** Section 4.5 | Protocol-level pings only. "No application-level `ping`/`pong` messages." Server pings every 30 seconds. |
| **Backend plan (03)** Section 6.5 | Write pump sends protocol-level pings every 5 seconds. |
| **Architecture doc (04)** Section 4 | Protocol-level pings only. No `heartbeat` in the message catalog. |
| **Frontend plan (02)** Section 5.4 | Includes `heartbeat` in both OutboundMessage and InboundMessage type definitions. Section 5.5 handler responds to server `heartbeat` messages. Section 5.7 says "server pings every 5 seconds" and client responds to `heartbeat` messages. |

The frontend plan still implements application-level heartbeat messages, which the backend plan and architecture doc have explicitly removed. The frontend plan's `OutboundMessage` type includes `{ type: "heartbeat", payload: {} }` and the `handleMessage` function has a `case "heartbeat"` handler.

Additionally, the backend plan contradicts itself on ping interval: Section 4.5 says 30 seconds, Section 6.5 says 5 seconds.

### 2.6 Send Buffer Size -- MINOR INCONSISTENCY

| Document | Buffer size |
|---|---|
| **Backend plan (03)** Section 6.3 | `sendBufferSize = 32` |
| **Architecture doc (04)** Section 6.4 | `Send buffer per connection: 64 messages` |

The backend plan changelog says "Send buffer reduced from 64 to 32. Align with architecture doc." But the architecture doc Section 6.4 still says 64. The architecture doc was not updated.

### 2.7 Room ID Hex Suffix Length

| Document | Suffix |
|---|---|
| **Backend plan (03)** | 12 hex chars (48 bits) |
| **Architecture doc (04)** | 12 hex chars (48 bits) |
| **Frontend plan (02)** Section 6.4 | `crypto.randomUUID().slice(0, 8)` -- **8 chars** |

The frontend plan's `generateRoomUrl` function still generates 8-character hex suffixes, while the backend and architecture docs specify 12 characters. Since room IDs are generated client-side, this is a real bug: the frontend would produce shorter IDs than the backend expects.

Additionally, `crypto.randomUUID().slice(0, 8)` produces UUID characters (hex + hyphens), not pure hex. The first 8 characters of a UUID v4 like `550e8400-...` are `550e8400` which happens to be hex, but this is fragile and semantically wrong. The `extractRoomName` regex in the same file uses `/-[a-f0-9]{8}$/` to strip the suffix -- this would not match a 12-character suffix.

### 2.8 `room_state` Payload Structure Discrepancy

During the reveal phase, the `room_state` snapshot should include vote values. The documents describe this differently:

| Document | Reveal-phase `room_state` |
|---|---|
| **Frontend plan (02)** Section 5.4 | `room_state` payload includes `id`, `name`, `phase`, `participants` array. During reveal, participants include `vote: string`. No `stats` or `result` in the snapshot. |
| **Backend plan (03)** Section 4.4 | `room_state` includes `roomId`, `roomName`, `phase`, `participants`, `result`. During reveal, `result` contains votes + stats. |
| **Architecture doc (04)** Section 4.4 | `room_state` payload includes `id`, `name`, `phase`, `participants`. During reveal, stats are "included at the top level of the payload." |

Three different structures. The field names also differ: `id` vs `roomId`, `name` vs `roomName`. And whether stats are included in the snapshot (and how) varies.

### 2.9 `votes_revealed` Payload Structure Discrepancy

| Document | `votes_revealed` structure |
|---|---|
| **Frontend plan (02)** Section 5.4 | `{ votes: Vote[], stats: { average, median } }` -- stats nested under `stats` key, with only `average` and `median`. |
| **Backend plan (03)** Section 4.4 | `{ votes: [...], average, median, uncertainCount, totalVoters, hasConsensus, spread }` -- stats are flat alongside `votes`, and `spread` is `[min, max]` array or `null`. |
| **Architecture doc (04)** Section 4.5 | `{ votes: [...], stats: { average, median, numericVoteCount, uncertainCount, totalParticipants, consensus, spread: { min, max, label } } }` -- stats nested under `stats` key, with full stats including `spread` as an object. |

Three different structures for the same payload. The field names differ (`hasConsensus` vs `consensus`, `totalVoters` vs `totalParticipants`), the nesting differs (flat vs nested `stats`), and the `spread` representation differs (array vs object vs null).

### 2.10 `error` Event Payload

| Document | Error payload |
|---|---|
| **Frontend plan (02)** Section 5.4 | `{ code: string, message: string }` (inside `payload`) |
| **Backend plan (03)** Section 4.4 | `{ message: string }` (flat, no `code` field) |
| **Backend plan (03)** Section 13.2 | `{ code: string, message: string }` (inside `payload`) |
| **Architecture doc (04)** Section 4.4 | `{ code: string, message: string }` |

The backend plan Section 4.4 omits the `code` field from the error event, but Section 13.2 includes it. The `code` field is needed by the frontend to handle specific errors (e.g., `room_not_found`, `invalid_name`).

### 2.11 Frontend Project Structure Discrepancy

| Document | Frontend structure |
|---|---|
| **Frontend plan (02)** | `src/components/HomePage/`, `src/components/RoomPage/`, `src/ws.ts`, `src/utils/` |
| **Architecture doc (04)** Section 9 | `src/pages/home.tsx`, `src/pages/room.tsx`, `src/ws/client.ts`, `src/lib/`, `src/styles/components.css` |

Key differences:
- Frontend plan puts `HomePage` and `RoomPage` in `components/` with PascalCase directories. Architecture doc puts them in a separate `pages/` directory with kebab-case filenames.
- Frontend plan has `ws.ts` (single file). Architecture doc has `ws/client.ts` (directory + file).
- Frontend plan has `utils/`. Architecture doc has `lib/`.
- Frontend plan has per-component CSS files. Architecture doc has a single `styles/components.css`.
- Architecture doc includes `lib/storage.ts` as a separate file. Frontend plan puts localStorage helpers in `state.ts`.

### 2.12 `participant_joined` Payload

| Document | Payload |
|---|---|
| **Frontend plan (02)** Section 5.4 | `{ sessionId, userName, status }` -- includes `status` field |
| **Backend plan (03)** Section 4.4 | `{ sessionId, userName }` -- no `status` field |
| **Architecture doc (04)** Section 4.3 | `{ sessionId, userName, status }` -- includes `status` field |

The backend plan omits the `status` field from `participant_joined`.

---

## Part 3: Devil's Advocate Round 2

The plans are now in much better shape. The major structural over-engineering has been eliminated. What remains is smaller.

### 3.1 The Two Protocol Definitions Are Redundant

The backend plan (Section 5) and the architecture doc (Section 4) both define the full WebSocket protocol, and they disagree. This is the root cause of most remaining inconsistencies. The architecture doc claims to be "THE single source of truth" for the protocol, but the backend plan also has a complete independent specification.

**Recommendation:** The backend plan should NOT redefine the protocol. It should reference the architecture doc's Section 4 and focus on implementation details (how the Go code dispatches messages, error handling, etc.). Having two "canonical" protocol specs guarantees they will drift.

### 3.2 Frontend `handleMessage` Still Has Application-Level Heartbeat

The frontend plan implements a `case "heartbeat"` handler that responds to server heartbeat messages. But both the backend plan and architecture doc have removed application-level heartbeats in favor of protocol-level pings. The frontend should remove the `heartbeat` case from `handleMessage`, the `heartbeat` entry from `OutboundMessage`, and the `heartbeat` entry from `InboundMessage`.

### 3.3 Reconnection Backoff Sequence Discrepancy

| Document | Backoff sequence |
|---|---|
| **Frontend plan (02)** Section 5.3 | 500ms, 1s, 2s, 4s, 8s, 10s (cap at 10s), with 30% jitter |
| **Backend plan (03)** Section 4.6 | 1s, 2s, 4s, 8s, 16s, 30s (cap at 30s) |

These should match. The frontend plan's lower cap (10s) is more user-friendly. The backend plan Section 4.6 describes client-side behavior, which should defer to the frontend plan.

### 3.4 `ConfirmDialog` -- Is It Needed as a Separate Component?

`ConfirmDialog` is used exactly once, for "Clear Room." It is a generic reusable modal, but with only one use case, inlining it into `RoomPage` (conditional render of a simple modal div) would save a component. However, the separation is clean, the component is well-specified, and extracting it makes the `RoomPage` template cleaner. This is borderline. Keeping it is acceptable.

### 3.5 `extractRoomName` Is Fragile

The `extractRoomName` function strips the hex suffix and replaces hyphens with spaces: `sprint-42-a3f1c9b2` becomes `sprint 42`. But what if the room name itself contains numbers that look like hex? E.g., room name "Team A3F" produces slug `team-a3f-a1b2c3d4e5f6`, and `extractRoomName` would try to strip `-a3f` as part of the suffix pattern. The regex `/-[a-f0-9]{8}$/` (or 12-char variant) would only match the actual suffix, but the resulting display name would still lose information about the original name's casing and special characters.

A simpler approach: the server sends `roomName` (the display name) in the `room_state` message. The client does not need to reverse-engineer the name from the slug. The `extractRoomName` function is only needed before the WebSocket connection is established (to show something in the header while connecting). This is a minor edge case.

### 3.6 CSS: Per-Component Files vs Single File

The frontend plan uses per-component CSS files (10 files). The architecture doc uses a single `styles/components.css`. With ~500 lines of total CSS and BEM naming preventing collisions, a single CSS file is simpler to manage and avoids 10 import statements. But per-component files provide better co-location. Either approach works at this scale. They should agree on one.

### 3.7 `Stats` Computed Client-Side or Server-Side?

The backend computes stats when votes are revealed (included in `votes_revealed`). The frontend has `utils/stats.ts` for stats calculation. Why does the frontend need its own stats calculation if the server sends them?

The frontend `stats.ts` may be needed for:
1. Displaying stats from the `room_state` snapshot (if a user joins during reveal phase).
2. Re-computing stats locally for immediate display before the server confirms.

If the server always includes stats in both `votes_revealed` and in the `room_state` snapshot during reveal phase, the frontend does not need `stats.ts` at all -- it can just display what the server sends. This would eliminate a file and avoid the risk of client/server stats disagreeing.

The architecture doc says `room_state` during reveal phase includes stats. If that is guaranteed, `utils/stats.ts` is redundant.

### 3.8 Room Name Extraction on Server Side

The backend plan Section 4.2 says: "The room name is extracted from the slug portion of the ID (everything before the last `-{12hexchars}`)." This reverse-engineering of the display name from the slug is fragile for the same reasons as `extractRoomName` on the frontend. But the server has no other source for the room name -- it does not receive it in the `join` message.

**Option A:** Add `roomName` to the `join` message payload. The first joiner provides the display name. Simple, no parsing.
**Option B:** Keep slug-to-name extraction. Accept that "Sprint 42" becomes "sprint 42" (lowercase, no recovery of original casing).

Option A is cleaner and adds one field to one message.

---

## Part 4: Product Completeness

### 4.1 UX Spec Coverage

| UX Requirement | Frontend | Backend | Architecture | Covered? |
|---|---|---|---|---|
| Room creation flow | Yes | Yes | Yes | Yes |
| Name entry modal (join + edit modes) | Yes | Yes | Yes | Yes |
| Room page (voting + reveal phases) | Yes | Yes | Yes | Yes |
| Participant cards with presence states | Yes | Yes | Yes | Yes |
| Card deck with all states | Yes | -- | Yes | Yes |
| Copy Link button + "Copied!" toast | Yes | -- | -- | Yes |
| Settings dropdown (change name, leave) | Yes | Yes | Yes | Yes |
| Toast notifications (all 7 messages) | Yes | -- | -- | Yes |
| Confirmation dialog (Clear Room) | Yes | -- | -- | Yes |
| Reconnection banner | Yes | -- | Yes | Yes |
| Statistics (avg, median, consensus, spread) | Yes | Yes | Yes | Yes |
| "?" vote handling (excluded from stats) | Yes | Yes | Yes | Yes |
| Presence: active/idle/disconnected | Yes | Yes | Yes | Yes |
| Reconnection with vote preservation | Yes | Yes | Yes | Yes |
| Room GC (24h) | -- | Yes | Yes | Yes |
| Rate limiting | -- | Yes | Yes | Yes |
| Room not found page | Yes | Yes | Yes | Yes |
| Duplicate names allowed | Implicit | Yes | Yes | Yes |
| Browser back/forward | Yes | -- | -- | Yes |
| Responsive design (breakpoints, mobile card deck) | Yes | -- | -- | Yes |
| Accessibility (ARIA, keyboard, reduced motion) | Yes | -- | -- | Yes |
| SPA fallback routing | Yes | Yes | Yes | Yes |

**Coverage: 100% of UX spec features are technically specified.** No missing user flows.

### 4.2 Reconnection Behavior: End-to-End

The reconnection flow is now fully specified across all documents:

1. **Client detects disconnect** (frontend plan Section 5.2-5.3): `onclose`/`onerror` fires, `connectionStatus` set to `"reconnecting"`, banner appears.
2. **Client retries** (frontend plan Section 5.3): Exponential backoff 500ms-10s with jitter, wall-clock timeout at 30s.
3. **Client reconnects** (frontend plan Section 5.3): Sends `join` with same `sessionId`.
4. **Server recognizes session** (backend plan Section 3.2 `AddParticipant`): Checks if `sessionId` exists, restores participant, marks active.
5. **Server cancels grace timer** (backend plan Section 6.1): If reconnection within grace period.
6. **Server sends `room_state`** (backend plan Section 4.2): Full snapshot including current phase and votes.
7. **Server broadcasts `participant_joined`** (backend plan Section 4.2): Other clients see the participant return.
8. **Client renders** (frontend plan Section 5.5): Updates all signals from `room_state`, shows "Reconnected" toast.

This is complete.

### 4.3 Deployment Story

Complete and simple:
- `npm run build` to build frontend into `dist/`.
- `go build` embeds `dist/` into binary via `embed.FS`.
- Single binary, single port, 3 env vars.
- Docker: multi-stage build, `FROM scratch`, ~15 MB image.
- Development: Vite on 5173 proxies to Go on 8080.

No gaps.

---

## Part 5: Final Actionable Items

### Critical: Protocol Consistency

1. **[ALL] [MUST] Resolve envelope format once and for all.** The architecture doc's changelog and CLAUDE.md say "flat", but all detailed protocol examples across all three documents use `{ type, payload }`. Pick one. If flat: rewrite all protocol examples in all three docs. If envelope: update the architecture doc changelog entry, decision log, and CLAUDE.md template. Recommendation: go with `{ type, payload }` since that is what all the detailed specs already show.

2. **[ALL] [MUST] Agree on WebSocket URL path.** Frontend and backend plans both use `/ws/room/:id`. Architecture doc uses `/ws/{roomId}`. Update the architecture doc Sections 4.1, 5, and all examples to use `/ws/room/{roomId}`.

3. **[ALL] [MUST] Unify `name_changed` vs `name_updated`.** Backend plan uses `name_changed` (Section 5.3). Frontend plan and architecture doc use `name_updated`. Pick one, update the other.

4. **[ALL] [MUST] Unify client-side presence event name.** Frontend plan uses `presence_update`. Backend plan and architecture doc use `presence`. Update the frontend plan's `OutboundMessage` type and `usePresence` hook to use `presence`.

5. **[FRONTEND] [MUST] Fix room ID suffix length.** `generateRoomUrl` uses `crypto.randomUUID().slice(0, 8)` producing 8 characters. Must produce 12 hex characters to match the 48-bit requirement. Use `crypto.getRandomValues(new Uint8Array(6))` and hex-encode, or `Array.from(crypto.getRandomValues(new Uint8Array(6)), b => b.toString(16).padStart(2, '0')).join('')`. Also update `extractRoomName` regex from `/-[a-f0-9]{8}$/` to `/-[a-f0-9]{12}$/`.

6. **[ALL] [MUST] Unify `votes_revealed` payload structure.** Three different structures exist. Pick one (recommend the architecture doc's version with nested `stats` object and explicit field names) and update the other two docs.

7. **[ALL] [MUST] Unify `room_state` payload structure.** Field names differ (`id` vs `roomId`, `name` vs `roomName`) and the structure of reveal-phase data differs. Pick one canonical structure and update all docs.

### Important: Remove Redundancy and Fix Remaining Discrepancies

8. **[BACKEND] [SHOULD] Remove full protocol definition from backend plan.** The backend plan Section 5 should reference the architecture doc Section 4 as the canonical protocol, not redefine it. The backend plan should focus on how Go code implements the protocol (dispatch logic, error handling, marshaling), not the wire format. Having two independent protocol specs is the root cause of most remaining inconsistencies.

9. **[FRONTEND] [SHOULD] Remove application-level heartbeat.** Delete `heartbeat` from `OutboundMessage`, `InboundMessage`, and the `case "heartbeat"` handler in `handleMessage`. Protocol-level pings handle liveness. Also remove the "server pings every 5 seconds" note from Section 5.7 -- the server sends protocol-level pings, which the browser handles automatically.

10. **[BACKEND] [SHOULD] Fix heartbeat interval inconsistency.** Section 4.5 says 30 seconds, Section 6.5 says 5 seconds. Pick one. The architecture doc's hardcoded defaults list says 5 seconds, which is correct for presence detection. Update Section 4.5 to 5 seconds.

11. **[ARCHITECTURE] [SHOULD] Fix send buffer size.** Section 6.4 says 64. Backend plan says 32. Architecture doc's own hardcoded defaults list says 32. Update Section 6.4 to 32.

12. **[ALL] [SHOULD] Unify `participant_joined` payload.** Backend plan omits `status` field. Frontend plan and architecture doc include it. Add `status` to backend plan's `participant_joined`.

13. **[ALL] [SHOULD] Unify `error` event payload.** Backend plan Section 4.4 omits the `code` field. Section 13.2 includes it. Ensure all docs include `code` in the error event.

14. **[ALL] [SHOULD] Agree on frontend directory structure.** Frontend plan uses `components/HomePage/`, `ws.ts`, `utils/`. Architecture doc uses `pages/home.tsx`, `ws/client.ts`, `lib/`. Pick one and update the other. Recommendation: use the frontend plan's structure (it is the frontend-specific doc and is more detailed).

### Simplification Opportunities

15. **[FRONTEND] [COULD] Eliminate `utils/stats.ts`.** If the server always includes stats in `votes_revealed` and in the `room_state` snapshot during reveal phase, the frontend does not need its own stats calculation. It just displays what the server sends. This eliminates code duplication and the risk of client/server disagreement.

16. **[ALL] [COULD] Add `roomName` to the `join` message.** This avoids the fragile slug-to-name reverse-engineering on the server side. The first joiner provides the display name. All subsequent joiners receive it from `room_state`. Adds one field to one message.

17. **[FRONTEND] [COULD] Unify CSS approach.** Frontend plan uses per-component CSS files. Architecture doc uses a single `components.css`. At ~500 lines total, either works. Pick one.

18. **[ALL] [COULD] Align reconnection backoff caps.** Frontend plan caps at 10s. Backend plan (describing client behavior) caps at 30s. The frontend plan is the authoritative source for client behavior. Remove the conflicting description from the backend plan Section 4.6, or make it a brief reference to the frontend plan.

---

## Summary

The v2.0 documents represent a major improvement. The architecture is clean, the component count is reasonable, the dependency count is minimal, and the deployment story is simple. 22 of 25 iteration 1 items were fully addressed.

The remaining issues fall into two categories:

**1. Protocol specification drift (items 1-7, 10-13).** The fundamental problem is that three documents each define the WebSocket protocol independently, and they have drifted. The fix is straightforward: make the architecture doc the single source of truth (it already claims to be), have the backend plan reference it rather than redefine it, and do a reconciliation pass on all message examples. This is mostly find-and-replace work, not design work.

**2. One real bug (item 5).** The frontend generates 8-character room ID suffixes while the backend expects 12. This would cause room URL mismatches in production.

No architectural changes are needed. The fundamental design is sound. The remaining work is reconciliation, not redesign.

**Estimated effort to close all items: 1-2 hours of document editing.** No code design decisions remain open.
