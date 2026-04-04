# om-scrum-poker: Frontend Technical Plan

**Version:** 3.1
**Date:** 2026-04-04
**Status:** Approved (aligned with Architecture Doc v3.0)
**Depends on:** [01-ux-design.md](./01-ux-design.md), [04-architecture.md](./04-architecture.md)

> **Protocol Reference:** See [Architecture Doc v3.0, Section 4](./04-architecture.md#4-websocket-protocol-canonical) for the canonical WebSocket protocol. This frontend plan does NOT redefine the protocol. All event names, payload structures, and the `{ type, payload }` envelope format are authoritative in the architecture doc.

---

## Changelog

### v3.0 -> v3.1

| # | Change | Reason (review iteration 3 ref) |
|---|---|---|
| 1 | **No frontend changes required.** Phase string `"reveal"` and all other values were already correct. Version bump for reconciliation with backend plan fixes. | Review iter 3 final reconciliation. |

### v2.0 -> v3.0

| # | Change | Reason (review iteration 2 ref) |
|---|---|---|
| 1 | **Removed all protocol definitions.** `OutboundMessage`, `InboundMessage` type unions, and all payload examples removed. Frontend plan now references Architecture Doc v3.0, Section 4 as the single source of truth. | Review 3.1: Two independent protocol specs caused all remaining inconsistencies. |
| 2 | **WebSocket URL aligned to `/ws/room/{roomId}`.** | Review 2.2: Canonical path from architecture doc. |
| 3 | **All event type strings aligned to architecture doc canonical names.** `presence_update` changed to `presence`. `heartbeat` removed entirely. | Review 2.4, 2.5: Frontend used non-canonical event names. |
| 4 | **`{ type, payload }` envelope format referenced, not redefined.** | Review 2.1: Envelope format is defined once in architecture doc. |
| 5 | **Room ID: 12-char hex suffix.** `generateRoomUrl` now uses `crypto.getRandomValues(new Uint8Array(6))` for 12 hex chars. `extractRoomName` regex updated to `/-[a-f0-9]{12}$/`. | Review 2.7: Frontend generated 8-char suffixes; backend expected 12. |
| 6 | **Removed application-level heartbeat.** Deleted `heartbeat` from outbound/inbound types, removed `case "heartbeat"` handler, removed Section 5.7. Protocol-level pings only. | Review 2.5, 3.2: Backend and architecture doc removed app-level heartbeat. Browser handles protocol pings automatically. |
| 7 | **`room_state` and `votes_revealed` payloads aligned to architecture doc.** Field names: `roomId`, `roomName`, `phase`, `participants`, `result`. Stats are flat in `votes_revealed`, not nested under `stats`. | Review 2.8, 2.9: Three docs had different structures. |
| 8 | **Removed `utils/stats.ts`.** Server always includes stats in `votes_revealed` and in `room_state` during reveal phase. Frontend displays what server sends. | Review 3.7: Redundant computation, risk of client/server disagreement. |
| 9 | **`join` message includes `roomName`.** First joiner provides display name; eliminates slug-to-name parsing on server. | Review 3.8: Reverse-engineering display name from slug is lossy. |
| 10 | **Presence event name: `presence`** (not `presence_update`). `usePresence` hook sends `{ type: "presence", payload: { status } }`. | Review 2.4: Backend and architecture doc use `presence`. |
| 11 | **Statistics rendering reads server-provided stats directly.** No local computation. Displays `average`, `median`, `uncertainCount`, `totalVoters`, `hasConsensus`, `spread` from `votes_revealed` payload. | Review 2.9: Architecture doc unified the payload structure. |

### v1.0 -> v2.0

| # | Change | Reason (review ref) |
|---|---|---|
| 1 | All file extensions are `.tsx` / `.ts` -- never `.jsx` | [MUST] Consistency with TypeScript choice (review 1.9) |
| 2 | WebSocket message format aligned to `{ type, payload }` envelope | [MUST] Match backend plan envelope (review 1.3) |
| 3 | WebSocket URL changed to `/ws/room/:id` | [MUST] Canonical path agreed with backend (review 1.4) |
| 4 | All event names aligned with backend plan canonical protocol | [MUST] Eliminate naming mismatches (review 1.3) |
| 5 | Components reduced from 16 to 10 by inlining trivial wrappers | [SHOULD] CopyLinkButton, ReconnectionBanner, ActionButtons, SettingsDropdown inlined (review DA 1.2) |
| 6 | Three state files + storage merged into one `state.ts` | [SHOULD] 11 signals fit in one file (review DA 1.3) |
| 7 | CSS approach confirmed as plain CSS with custom properties (not CSS Modules) | [SHOULD] Simpler for project size (review 1.2) |
| 8 | Added error handling for WebSocket `error` events | [SHOULD] Missing `error` case in message handler (review 4.1) |
| 9 | Added name change protocol end-to-end specification | [SHOULD] Was underspecified (review 4.2) |
| 10 | Added accessibility implementation section | [COULD] Map UX doc Appendix C to implementation (review 2, gap 1) |
| 11 | Eliminated `types/` directory -- types co-located with usage | [SHOULD] ~10 types do not justify 3 files (review DA 4.4) |
| 12 | Eliminated `utils/clipboard.ts` -- inlined into Header | [SHOULD] Single-use 10-line helper (review DA 1.5) |
| 13 | Eliminated `hooks/useWebSocket.ts` -- `ws/client.ts` manages lifecycle directly | [SHOULD] Hook was thin wrapper around client module (review DA 1.4) |
| 14 | Eliminated `hooks/useClickOutside.ts` -- settings dropdown inlined | [SHOULD] Used only once (review DA 1.4) |

---

## Table of Contents

1. [Technology Choice](#1-technology-choice)
2. [Project Structure](#2-project-structure)
3. [Component Architecture](#3-component-architecture)
4. [State Management](#4-state-management)
5. [WebSocket Client](#5-websocket-client)
6. [Routing](#6-routing)
7. [Responsive Design Implementation](#7-responsive-design-implementation)
8. [Accessibility](#8-accessibility)
9. [Deployment](#9-deployment)

---

## 1. Technology Choice

### 1.1 UI Library: Preact

**Choice:** [Preact](https://preactjs.com/) (v10.x) with `preact/hooks` and `@preact/signals`.

**Justification:**

| Considered | Size (gzipped) | Verdict |
|---|---|---|
| Vanilla JS/TS | 0 KB | No framework overhead, but managing DOM updates for a real-time collaborative UI (participant list mutations, vote state changes, presence transitions) leads to imperative spaghetti. The complexity is right at the threshold where a reactive library pays for itself. |
| React | ~45 KB | Too heavy for a lightweight tool. We do not need the ecosystem. |
| Preact | ~4 KB | API-compatible with React, tiny footprint, mature, excellent hook support. Signals provide fine-grained reactivity without external state libraries. |
| Solid | ~7 KB | Excellent performance, but smaller ecosystem and less familiar to most developers. |
| Svelte | ~2 KB runtime | Good option, but non-standard component file format (.svelte) and compilation model. |

Preact wins on the balance of simplicity, size, developer familiarity, and ecosystem maturity. The `preact/compat` layer is NOT needed -- we use Preact's native API directly.

### 1.2 Language: TypeScript

**Choice:** TypeScript (strict mode). All files use `.ts` / `.tsx` extensions.

**Justification:** The WebSocket message protocol and room state model benefit enormously from type safety. Types serve as documentation. The cost is near-zero with modern tooling (Vite handles TS natively). Shared types (message payloads, room state) are defined once and used everywhere.

### 1.3 CSS Approach: Plain CSS with CSS Custom Properties

**Choice:** Vanilla CSS using a single design-token file with CSS custom properties, plus component-scoped CSS files with BEM-inspired naming.

**Justification:**

| Considered | Verdict |
|---|---|
| CSS Modules | Good scoping, but adds tooling complexity. With ~10 components and disciplined BEM-like naming, class collisions are not a realistic risk. Plain CSS is simpler to debug and has zero build configuration. |
| Tailwind CSS | Adds a build dependency, a config file, and a purge step. For ~10 components, utility classes are overkill. |
| CSS-in-JS | Runtime cost, additional dependency. Antithetical to KISS. |
| Plain CSS + custom properties | Zero dependencies. Custom properties give us theming and design tokens. One file defines the palette from the UX spec. Component CSS files are co-located. Simple, debuggable, fast. |

We use a BEM-inspired naming convention: `.component-name__element--modifier` (e.g., `.card__value--selected`). This prevents collisions without tooling.

### 1.4 Build Tool: Vite

**Choice:** [Vite](https://vitejs.dev/) with `@preact/preset-vite`.

Near-instant HMR during development. Produces optimized, hashed bundles for production. First-class TypeScript and Preact support. Single plugin: `@preact/preset-vite`. No Webpack, no Babel, no PostCSS.

**Production build target:** ES2020. All modern browsers support this. No polyfills needed.

### 1.5 Dependencies Summary

| Package | Purpose | Size |
|---|---|---|
| `preact` | UI rendering | ~4 KB |
| `@preact/signals` | Reactive state management | ~2 KB |
| `vite` | Build tool (dev only) | -- |
| `@preact/preset-vite` | Vite plugin (dev only) | -- |
| `typescript` | Type checking (dev only) | -- |

**Total runtime dependencies: 2 packages, ~6 KB gzipped.** No router library, no state library, no CSS framework.

**Note on `@preact/signals`:** This is a separate package from `preact` -- it is NOT included in the core `preact` package. It adds ~2 KB gzipped. The trade-off vs. plain hooks (`useState` + props) is marginal at this project size (10 components, max 3 levels deep), but signals simplify state sharing by eliminating prop drilling. Acceptable either way; we choose signals for cleaner code.

---

## 2. Project Structure

```
frontend/
  index.html                    # Entry point, single <div id="app">
  vite.config.ts                # Vite configuration
  tsconfig.json                 # TypeScript configuration

  src/
    main.tsx                    # App bootstrap, mounts to #app
    app.tsx                     # Root component, handles routing
    state.ts                    # All signals + localStorage helpers + types
    ws.ts                       # WebSocket client, handlers, message builders

    components/
      HomePage/
        HomePage.tsx
        home-page.css
      NameEntryModal/
        NameEntryModal.tsx
        name-entry-modal.css
      RoomPage/
        RoomPage.tsx
        room-page.css
      Header/
        Header.tsx
        header.css
      ParticipantList/
        ParticipantList.tsx
        participant-list.css
      ParticipantCard/
        ParticipantCard.tsx
        participant-card.css
      CardDeck/
        CardDeck.tsx
        card-deck.css
      Card/
        Card.tsx
        card.css
      Toast/
        Toast.tsx
        toast.css
      ConfirmDialog/
        ConfirmDialog.tsx
        confirm-dialog.css

    hooks/
      usePresence.ts            # Hook that tracks visibility/activity for presence

    utils/
      room-url.ts               # Slug generation, room ID parsing

    styles/
      tokens.css                # CSS custom properties (colors, spacing, radii, shadows)
      reset.css                 # Minimal CSS reset (~30 lines)
      global.css                # Body, typography, base element styles
```

**Conventions:**
- One component per directory. Each directory contains a `.tsx` and a `.css` file.
- No `index.ts` barrel files -- explicit imports improve traceability.
- All `.tsx` / `.ts` extensions, never `.jsx` / `.js`.
- Types are co-located with the code that uses them: domain types (`Participant`, `Vote`, `Phase`) live in `state.ts`; WebSocket message types live in `ws.ts`.
- CSS file names use kebab-case matching the BEM block name (e.g., `card-deck.css` for the `CardDeck` component).

**What was removed from v1.0 and why:**
- `types/` directory (3 files) -- ~10 types co-located in `state.ts` and `ws.ts` instead.
- `state/` directory (4 files) -- merged into single `state.ts`.
- `ws/handlers.ts`, `ws/messages.ts` -- merged into single `ws.ts`.
- `hooks/useWebSocket.ts` -- `ws.ts` manages lifecycle directly, no wrapper hook needed.
- `hooks/useClickOutside.ts` -- only consumer (SettingsDropdown) was inlined.
- `utils/clipboard.ts` -- inlined into Header component.
- `utils/stats.ts` -- removed in v3.0. Server always includes stats in `votes_revealed` and `room_state` during reveal phase. Frontend displays what server sends.
- Components `CopyLinkButton`, `SettingsDropdown`, `ActionButtons`, `ReconnectionBanner`, `Statistics` -- inlined into parent components.

---

## 3. Component Architecture

### 3.1 Component Inventory (10 components)

| Component | Responsibility |
|---|---|
| `App` | Route resolution, renders HomePage or RoomPage, hosts Toast container |
| `HomePage` | Room creation form (input + button) |
| `NameEntryModal` | Captures participant display name, blocks interaction |
| `RoomPage` | Main room experience, orchestrates all sub-components, manages WebSocket |
| `Header` | Top bar: room name, copy-link button (inline), settings menu (inline) |
| `ParticipantList` | Grid of participant cards |
| `ParticipantCard` | Single participant: name, presence dot, vote status |
| `CardDeck` | Row of selectable voting cards |
| `Card` | Single selectable voting card |
| `Toast` | Non-blocking notification pills |
| `ConfirmDialog` | Modal confirmation for destructive actions |

### 3.2 App (Root)

**File:** `src/app.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Route resolution. Renders either `HomePage` or `RoomPage` based on URL path. Hosts `Toast` container globally. |
| State | `currentPath: Signal<string>` -- derived from `window.location.pathname`. |
| Children | `HomePage`, `RoomPage`, `Toast`. |

### 3.3 HomePage

**File:** `src/components/HomePage/HomePage.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Room creation form. Single input + button. |
| Props | None. |
| Local state | `roomName: string` -- controlled input value. `validationError: string \| null`. |
| Behavior | On submit: generate slug from room name, append 12-char hex ID, navigate to `/room/{slug}-{id}`. |
| Validation | Regex: `/^[a-zA-Z0-9\s\-_]+$/`. Max 60 chars. Inline error message below input on invalid characters. |
| Submit | Enter key or button click. Button disabled when input is empty or invalid. |
| Auto-focus | Input receives focus on mount via `ref.current.focus()`. |

### 3.4 NameEntryModal

**File:** `src/components/NameEntryModal/NameEntryModal.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Captures participant display name. Blocks interaction until name is provided. |
| Props | `onSubmit: (name: string) => void`, `initialName?: string` |
| Local state | `name: string` -- controlled input value. |
| Rendering condition | Shown when `userName` signal is empty (first visit). Also shown in edit mode when triggered by "Change my name" (input pre-filled with `initialName`). |
| Behavior | On submit: calls `onSubmit(name.trim())`, which stores name in localStorage and triggers either WebSocket `join` (first time) or `update_name` (edit mode). |
| Overlay | Full-viewport backdrop with `backdrop-filter: blur(4px)`. No close button in join mode. In edit mode, backdrop click or Escape cancels. Traps keyboard focus within modal. |
| Auto-focus | Input receives focus on mount. |
| Submit | Enter key or button click. Button disabled when input is empty after trim. |
| Accessibility | `role="dialog"`, `aria-modal="true"`, `aria-labelledby` pointing to heading. Focus trapped: Tab cycles through input and button(s) only. |

### 3.5 RoomPage

**File:** `src/components/RoomPage/RoomPage.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Main room experience. Orchestrates all room sub-components. Initiates WebSocket connection on mount. Contains reconnection banner, action buttons, and statistics display inline (not separate components). |
| Props | `roomId: string` (extracted from URL). |
| Local state | None directly -- all room state lives in signals (`state.ts`). |
| Lifecycle | `useEffect` on mount: check for `userName` in localStorage. If missing, show `NameEntryModal`. If present, call `wsConnect(roomId, sessionId, userName)`. On unmount: call `wsDisconnect()`. |
| Children | `Header`, `NameEntryModal` (conditional), `ParticipantList`, `CardDeck`, `ConfirmDialog` (conditional). |
| Room-not-found | If the server responds with an `error` event with code `room_not_found` on join, render a simple message with a link to `/`. |

**Inline elements (previously separate components):**

- **Reconnection banner:** A conditional `<div>` at the top of the room page. Shows "Reconnecting..." (yellow) when `connectionStatus === "reconnecting"` or "Connection lost" + Retry button (red) when `connectionStatus === "failed"`. Hidden when connected. `position: sticky; top: 0`.
- **Action buttons:** "Show Votes" button (voting phase, full width, shows `votedCount of activeCount voted`), "New Round" button (secondary style), "Clear Room" button (destructive style, opens ConfirmDialog). All rendered inline in the room page layout.
- **Statistics:** After reveal, displays average, median, vote count, uncertain count, and consensus/spread status. Reads values directly from the server-provided `result` in the `votes_revealed` or `room_state` payload -- no client-side computation. Rendered inline below the participant list during reveal phase.

### 3.6 Header

**File:** `src/components/Header/Header.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Top bar showing room name, copy-link button, and settings menu. |
| Props | `roomName: string`, `onChangeName: () => void`, `onLeaveRoom: () => void` |
| Room name | Truncated with CSS `text-overflow: ellipsis` if exceeding available width. |
| Copy link (inline) | A `<button>` that copies `window.location.href` to clipboard via `navigator.clipboard.writeText()` with a fallback to `document.execCommand('copy')`. Toggles label between "Copy Link" and "Copied!" for 1.5s. Triggers a toast on success. `aria-label="Copy room link to clipboard"`. |
| Settings menu (inline) | Gear icon button. On click, toggles a dropdown with two items: "Change my name" (calls `onChangeName`, which opens `NameEntryModal` in edit mode) and "Leave room" (calls `onLeaveRoom`, which disconnects WebSocket and navigates to `/`). Dropdown dismisses on click outside (inline `useEffect` with `addEventListener`/`removeEventListener`) or Escape key. |

### 3.7 ParticipantList

**File:** `src/components/ParticipantList/ParticipantList.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Renders the grid of participant cards. |
| Props | None (reads from room state signals). |
| Derived state | `participants: Participant[]` from `participants` signal. |
| Layout | CSS flexbox with `flex-wrap: wrap`. Gap: 12px. Horizontally centered. |
| Section title | "Participants" during voting phase, "Results" during reveal phase. |
| Accessibility | `role="list"` on the container. |

### 3.8 ParticipantCard

**File:** `src/components/ParticipantCard/ParticipantCard.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Displays one participant: name, presence indicator, and vote status. |
| Props | `participant: Participant`, `phase: Phase`, `isCurrentUser: boolean` |
| Visual states (voting phase) | **Not voted:** Empty card outline or "..." placeholder. **Voted:** Face-down card with subtle checkmark. |
| Visual states (reveal phase) | **Voted (numeric):** Card face-up showing value. **Voted "?":** Shows "?" as-is. **Did not vote:** Shows "--". |
| Presence dot | 10px circle, positioned top-left. Color: green (active), yellow (idle), red (disconnected). CSS transition 300ms for color changes. |
| Disconnected state | Entire card at `opacity: 0.6`. |
| Name truncation | CSS `text-overflow: ellipsis` at ~10 characters. Full name via `title` attribute. |
| Current user indicator | If `isCurrentUser`, show a subtle pencil icon next to the name. Clicking it triggers name change flow. |
| Dimensions | `min-width: 80px`, `max-width: 120px`. |
| Accessibility | `role="listitem"`. Presence state conveyed via `aria-label` (e.g., "Ana, active, has voted"). |

### 3.9 CardDeck

**File:** `src/components/CardDeck/CardDeck.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Renders the row of selectable voting cards. |
| Props | None (reads `selectedCard` and `phase` from signals). |
| Card values | `['?', '0', '0.5', '1', '2', '3', '5', '8', '13', '20', '40', '100']` -- defined as a constant. |
| Behavior | Maps over values, renders a `Card` for each. Passes `isSelected`, `isDisabled`, and `onSelect` to each. |
| Layout | Flexbox with `flex-wrap: wrap`. On mobile, wraps into 2 rows of 6. On desktop, single row. Gap: 8px. |
| Disabled state | During reveal phase, all cards are disabled (reduced opacity, no pointer events). |
| Label | "Your card:" above the deck during voting phase. "Your vote: {value}" during reveal phase. |
| Sticky positioning | On mobile, `position: sticky; bottom: 0` with a solid background to keep the deck always visible. |
| Accessibility | `role="radiogroup"`, `aria-label="Select your vote"`. |

### 3.10 Card

**File:** `src/components/Card/Card.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | A single selectable voting card. |
| Props | `value: string`, `isSelected: boolean`, `isDisabled: boolean`, `onSelect: (value: string) => void` |
| Behavior | On click: if disabled, do nothing. If already selected, call `onSelect('')` (deselect = un-vote). Otherwise, call `onSelect(value)`. |
| Visual states | **Default:** White background, thin gray border. **Hover:** `translateY(-2px)`, shadow increase. **Selected:** Primary color background, white text, slight scale-up (1.05). **Disabled:** `opacity: 0.4`, `cursor: not-allowed`. |
| Dimensions | Desktop: 56px x 72px. Mobile: 44px x 56px (via CSS media query). `border-radius: 8px`. |
| Accessibility | `role="radio"`, `aria-checked={isSelected}`, `aria-disabled={isDisabled}`, `aria-label="Vote {value}"`. Keyboard: Enter/Space to select. |
| Animation | `transition: transform 150ms ease, box-shadow 150ms ease, background-color 150ms ease`. Respects `prefers-reduced-motion: reduce`. |

### 3.11 Toast

**File:** `src/components/Toast/Toast.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Displays non-blocking notification pills at the top of the viewport. |
| Props | None (reads from `toasts` signal in `state.ts`). |
| Rendering | Maps over the `toasts` array. Each toast is a pill-shaped element with text. Positioned `fixed`, top-center. Newest on top. Max 3 visible. |
| Animation | Fade in over 200ms. Fade out over 300ms. CSS `opacity` transition plus `transform: translateY` for slide-in. |
| Auto-dismiss | Each toast auto-removes after 2000ms. |
| API | `addToast(message: string)` exported from `state.ts`. |
| Accessibility | `role="status"`, `aria-live="polite"` on the container so screen readers announce toasts. |

**Toast messages used in the app:**

| Trigger | Message |
|---|---|
| Link copied | "Link copied!" |
| Participant joined | "{name} joined" |
| Participant left | "{name} left" |
| Room cleared | "Room cleared" |
| New round started | "New round started" |
| Reconnected after drop | "Reconnected" |
| WebSocket error | "{error message}" |

### 3.12 ConfirmDialog

**File:** `src/components/ConfirmDialog/ConfirmDialog.tsx`

| Aspect | Detail |
|---|---|
| Responsibility | Modal confirmation for destructive actions (Clear Room). |
| Props | `title: string`, `body: string`, `confirmLabel: string`, `onConfirm: () => void`, `onCancel: () => void` |
| Overlay | Dimmed backdrop. Clicking backdrop or pressing Escape triggers `onCancel`. |
| Buttons | Cancel (secondary, left) and Confirm (destructive/red, right). |
| Focus trap | On mount, focus moves to the Cancel button. Tab cycles between Cancel and Confirm only. |
| Accessibility | `role="alertdialog"`, `aria-modal="true"`, `aria-labelledby` pointing to title, `aria-describedby` pointing to body. |

---

## 4. State Management

### 4.1 Approach: Preact Signals in a Single File

We use `@preact/signals` for all shared state. Signals provide fine-grained reactivity -- when a signal value changes, only the components that read that specific signal re-render. No context providers, no reducers, no boilerplate.

All signals, types, computed values, helper functions, and localStorage accessors live in a single `state.ts` file. With ~11 signals and ~10 types, this is perfectly readable in one file.

### 4.2 `state.ts`

```typescript
import { signal, computed } from "@preact/signals";

// ---------- Types ----------

export type Phase = "voting" | "reveal";
export type PresenceStatus = "active" | "idle" | "disconnected";
export type ConnectionStatus = "connecting" | "connected" | "reconnecting" | "failed";

export interface Participant {
  sessionId: string;
  userName: string;
  status: PresenceStatus;
  hasVoted: boolean;       // visible during voting phase (no value exposed)
  vote: string | null;     // populated after reveal, null during voting
}

// Result structure -- matches server's votes_revealed / room_state.result payload.
// See Architecture Doc v3.0, Section 4.5 for canonical field definitions.
export interface RoundResult {
  votes: VoteEntry[];
  average: number | null;
  median: number | null;
  uncertainCount: number;
  totalVoters: number;
  hasConsensus: boolean;
  spread: [number, number] | null;
}

export interface VoteEntry {
  sessionId: string;
  userName: string;
  value: string;
}

export interface ToastItem {
  id: number;
  message: string;
}

// ---------- Room State ----------

export const roomId = signal<string>("");
export const roomName = signal<string>("");
export const phase = signal<Phase>("voting");
export const participants = signal<Participant[]>([]);
export const roundResult = signal<RoundResult | null>(null);

// Derived
export const votedCount = computed(() =>
  participants.value.filter(p => p.hasVoted).length
);
export const activeParticipantCount = computed(() =>
  participants.value.filter(p => p.status !== "disconnected").length
);

// ---------- Connection State ----------

export const connectionStatus = signal<ConnectionStatus>("connecting");
export const reconnectAttempts = signal<number>(0);

// ---------- UI State ----------

export const selectedCard = signal<string>("");
export const toasts = signal<ToastItem[]>([]);
export const confirmDialog = signal<{
  title: string;
  body: string;
  confirmLabel: string;
  onConfirm: () => void;
} | null>(null);
export const showNameModal = signal<boolean>(false);
export const nameModalMode = signal<"join" | "edit">("join");

// Toast helper
let nextToastId = 0;
export function addToast(message: string) {
  const id = nextToastId++;
  toasts.value = [{ id, message }, ...toasts.value].slice(0, 3);
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id);
  }, 2000);
}

// ---------- localStorage ----------

const USER_NAME_KEY = "userName";
const SESSION_ID_KEY = "sessionId";

export function getUserName(): string | null {
  return localStorage.getItem(USER_NAME_KEY);
}

export function setUserName(name: string): void {
  localStorage.setItem(USER_NAME_KEY, name);
}

export function getSessionId(): string {
  let id = localStorage.getItem(SESSION_ID_KEY);
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem(SESSION_ID_KEY, id);
  }
  return id;
}
```

**Key changes from v2.0:**
- `Participant.presence` renamed to `Participant.status` to match server payload field name.
- `Participant.vote` is now `string | null` (not optional `string | undefined`) to match server JSON (`null` vs absent).
- Added `RoundResult` and `VoteEntry` interfaces matching the server's canonical payload structure (Architecture Doc v3.0, Section 4.5).
- `revealedVotes` signal replaced with `roundResult` signal that holds the full result object from the server.
- Removed `Vote` interface (replaced by `VoteEntry` matching server structure).

**Update pattern:** WebSocket message handlers in `ws.ts` directly mutate signals. Because signals are reactive, any component reading a signal will automatically re-render. No dispatch, no action creators, no reducers.

---

## 5. WebSocket Client

> **Protocol Reference:** See [Architecture Doc v3.0, Section 4](./04-architecture.md#4-websocket-protocol-canonical) for the canonical WebSocket protocol -- all event names, payload structures, envelope format, error codes, and heartbeat behavior. This section describes the frontend *implementation* of that protocol, not the protocol itself.

### 5.1 Module: `ws.ts`

A single file containing the WebSocket client, outbound message builders, and inbound message handlers. Exports three functions:

```typescript
export function wsConnect(roomId: string, sessionId: string, userName: string): void;
export function wsSend(msg: { type: string; payload: Record<string, unknown> }): void;
export function wsDisconnect(): void;
```

Message types in `ws.ts` are derived from the Architecture Doc v3.0, Section 4.3 canonical event table. The TypeScript discriminated union for inbound messages is defined in `ws.ts` and must stay in sync with the architecture doc.

### 5.2 Connection Lifecycle

```
wsConnect() called
    |
    v
Create WebSocket: ws(s)://{host}/ws/room/{roomId}
    |
    v
onopen -->  Send "join" message: { type: "join", payload: { sessionId, userName, roomName } }
            Set connectionStatus = "connected"
            Reset reconnectAttempts = 0
    |
    v
onmessage --> Parse JSON, dispatch to handleMessage()
    |
    v
onclose / onerror --> Set connectionStatus = "reconnecting"
                      Begin reconnection loop
```

The `join` message includes `roomName` (the human-readable display name). The first joiner's `roomName` is used as the room's display name; subsequent joiners' `roomName` is ignored by the server.

### 5.3 Reconnection Strategy

**Algorithm:** Exponential backoff with jitter.

```typescript
function getReconnectDelay(attempt: number): number {
  const base = Math.min(500 * Math.pow(2, attempt), 10000); // 500ms, 1s, 2s, 4s, 8s, 10s cap
  const jitter = Math.random() * base * 0.3; // up to 30% jitter
  return base + jitter;
}
```

**Sequence:** 500ms, 1s, 2s, 4s, 8s, 10s, 10s, 10s, ...

**Timeout:** After 30 seconds of failed attempts (wall-clock time, not attempt count), set `connectionStatus = "failed"`. The reconnection banner in RoomPage then shows a manual "Retry" button.

**Manual retry:** Resets the attempt counter and restarts the reconnection loop.

**On successful reconnect:** The client sends the `join` message again with the same `sessionId`. The server restores the participant to the room with their previous vote if the round is still active. A "Reconnected" toast is shown.

### 5.4 Message Handling

A single dispatcher function that pattern-matches on `type` and updates signals. All event names and payload structures match [Architecture Doc v3.0, Section 4.3-4.5](./04-architecture.md#43-canonical-event-table).

```typescript
export function handleMessage(data: string): void {
  const msg = JSON.parse(data);

  switch (msg.type) {
    case "room_state":
      roomId.value = msg.payload.roomId;
      roomName.value = msg.payload.roomName;
      phase.value = msg.payload.phase;
      participants.value = msg.payload.participants;
      roundResult.value = msg.payload.result ?? null;
      break;

    case "participant_joined":
      participants.value = [...participants.value, {
        sessionId: msg.payload.sessionId,
        userName: msg.payload.userName,
        status: msg.payload.status,
        hasVoted: false,
        vote: null,
      }];
      addToast(`${msg.payload.userName} joined`);
      break;

    case "participant_left":
      const leftName = participants.value.find(
        p => p.sessionId === msg.payload.sessionId
      )?.userName;
      participants.value = participants.value.filter(
        p => p.sessionId !== msg.payload.sessionId
      );
      if (leftName) addToast(`${leftName} left`);
      break;

    case "vote_cast":
      participants.value = participants.value.map(p =>
        p.sessionId === msg.payload.sessionId ? { ...p, hasVoted: true } : p
      );
      break;

    case "vote_retracted":
      participants.value = participants.value.map(p =>
        p.sessionId === msg.payload.sessionId ? { ...p, hasVoted: false } : p
      );
      break;

    case "votes_revealed":
      phase.value = "reveal";
      roundResult.value = {
        votes: msg.payload.votes,
        average: msg.payload.average,
        median: msg.payload.median,
        uncertainCount: msg.payload.uncertainCount,
        totalVoters: msg.payload.totalVoters,
        hasConsensus: msg.payload.hasConsensus,
        spread: msg.payload.spread,
      };
      // Update participant vote values from the votes array
      const voteMap = new Map(
        msg.payload.votes.map((v: VoteEntry) => [v.sessionId, v.value])
      );
      participants.value = participants.value.map(p => ({
        ...p,
        vote: voteMap.get(p.sessionId) ?? null,
      }));
      break;

    case "round_reset":
      phase.value = "voting";
      selectedCard.value = "";
      roundResult.value = null;
      participants.value = participants.value.map(p => ({
        ...p, hasVoted: false, vote: null,
      }));
      addToast("New round started");
      break;

    case "room_cleared":
      participants.value = [];
      phase.value = "voting";
      selectedCard.value = "";
      roundResult.value = null;
      addToast("Room cleared");
      break;

    case "name_updated":
      participants.value = participants.value.map(p =>
        p.sessionId === msg.payload.sessionId
          ? { ...p, userName: msg.payload.userName }
          : p
      );
      break;

    case "presence_changed":
      participants.value = participants.value.map(p =>
        p.sessionId === msg.payload.sessionId
          ? { ...p, status: msg.payload.status }
          : p
      );
      break;

    case "error":
      addToast(msg.payload.message);
      // If error code is room_not_found, the RoomPage will
      // read the error and render a not-found message
      break;
  }
}
```

**No `heartbeat` case.** The server uses protocol-level WebSocket pings only (see Architecture Doc v3.0, Section 4.6). The browser handles protocol-level pong responses automatically -- no application code needed.

### 5.5 Name Change Protocol (End-to-End)

The full round-trip for changing a participant's display name:

1. User clicks pencil icon on their ParticipantCard, or "Change my name" in the Header settings menu.
2. `NameEntryModal` opens in edit mode, pre-filled with the current name.
3. User edits the name and submits.
4. Frontend updates localStorage via `setUserName(newName)`.
5. Frontend sends: `{ type: "update_name", payload: { userName: "New Name" } }`.
6. Server validates: non-empty after trim, 1-30 characters.
7. Server updates `Participant.UserName` in the room.
8. Server broadcasts to ALL participants (including sender): `{ type: "name_updated", payload: { sessionId: "...", userName: "New Name" } }`.
9. All clients update the participant list via the `name_updated` handler (see Section 5.4).
10. `NameEntryModal` closes.

**Error case:** If the server rejects the name (validation failure), it sends `{ type: "error", payload: { code: "invalid_name", message: "Name must be 1-30 characters" } }`. The toast displays the error. The modal stays open so the user can correct the name.

### 5.6 Presence Tracking (`hooks/usePresence.ts`)

This hook monitors user activity and tab visibility, then sends `presence` messages (canonical event name per Architecture Doc v3.0, Section 4.3):

- Listens to `document.visibilitychange`. On `hidden`: send `{ type: "presence", payload: { status: "idle" } }`. On `visible`: send `{ type: "presence", payload: { status: "active" } }`.
- Listens to `mousemove`, `keydown`, `touchstart` (debounced to every 30 seconds). Resets an inactivity timer. If no activity for 2 minutes: send `idle`.
- On unmount: cleans up all listeners.

### 5.7 Error Handling

WebSocket errors are handled at two levels:

1. **Connection errors** (`onerror` / `onclose`): Trigger the reconnection loop. The `connectionStatus` signal updates to `"reconnecting"`, and the reconnection banner appears in RoomPage.

2. **Application errors** (server sends `{ type: "error", payload: { code, message } }`): The `handleMessage` function shows `message` as a toast. Specific error codes may trigger additional behavior (see Architecture Doc v3.0, Section 4.5 for the full error code list):
   - `room_not_found`: RoomPage renders a not-found state with a link to `/`.
   - `invalid_name`: NameEntryModal stays open (user can correct the name).
   - All others: toast only.

---

## 6. Routing

### 6.1 Approach: Minimal Custom Router

With only two routes (`/` and `/room/:id`), a router library is unnecessary. We use a simple path-matching function in the `App` component.

### 6.2 Implementation

```typescript
// src/app.tsx
import { signal } from "@preact/signals";

const path = signal(window.location.pathname);

// Listen for popstate (browser back/forward)
window.addEventListener("popstate", () => {
  path.value = window.location.pathname;
});

// Programmatic navigation
export function navigate(to: string): void {
  window.history.pushState(null, "", to);
  path.value = to;
}

// Route matching
type Route =
  | { page: "home" }
  | { page: "room"; roomId: string };

function parseRoute(pathname: string): Route {
  const roomMatch = pathname.match(/^\/room\/(.+)$/);
  if (roomMatch) {
    return { page: "room", roomId: roomMatch[1] };
  }
  return { page: "home" };
}

// In App component render:
function App() {
  const route = parseRoute(path.value);
  return (
    <>
      {route.page === "home" && <HomePage />}
      {route.page === "room" && <RoomPage roomId={route.roomId} />}
      <ToastContainer />
    </>
  );
}
```

### 6.3 Navigation Flows

| Action | Method |
|---|---|
| Create room | `navigate("/room/{slug}-{id}")` after generating the room URL. |
| Leave room | `navigate("/")`. |
| Browser back from room | `popstate` fires, `path` signal updates, App renders `HomePage`. |
| Direct URL entry / shared link | Initial `path` signal reads `window.location.pathname`. App renders `RoomPage`. |
| Room not found | `RoomPage` renders inline error message with a link that calls `navigate("/")`. |

### 6.4 Room URL Generation and Parsing

The room ID is the full path segment after `/room/` (e.g., `sprint-42-a3f1c9b2d4e6`). The slug portion is human-readable but the server uses the entire string (including the 12-char hex suffix) as the room identifier.

```typescript
// utils/room-url.ts

export function generateRoomUrl(roomName: string): string {
  const slug = roomName
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .slice(0, 40);
  // 12 hex chars from 6 random bytes (48 bits entropy)
  // See Architecture Doc v3.0, Section 5.1 for room ID format spec
  const bytes = crypto.getRandomValues(new Uint8Array(6));
  const hex = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('');
  return `/room/${slug}-${hex}`;
}

export function extractRoomName(roomId: string): string {
  // Remove the 12-char hex suffix for display purposes (used only before
  // WebSocket connects; once connected, use roomName from room_state).
  return roomId.replace(/-[a-f0-9]{12}$/, "").replace(/-/g, " ");
}
```

**Note:** `extractRoomName` is only needed as a temporary display name before the WebSocket connection is established. Once connected, the authoritative room name comes from the server's `room_state` payload (`roomName` field).

---

## 7. Responsive Design Implementation

### 7.1 Breakpoints

Defined as media queries (CSS cannot use custom properties in media queries):

```css
/* Mobile: < 640px (base styles) */
/* Tablet: 640px - 1024px */
@media (min-width: 640px) { /* tablet+ */ }
/* Desktop: > 1024px */
@media (min-width: 1024px) { /* desktop+ */ }
```

Mobile-first approach: base styles target mobile, then progressively enhance.

### 7.2 Layout Strategy by Breakpoint

**Mobile (< 640px):**
- Single column layout.
- Header: room name truncated aggressively, icons only for Copy Link and Settings.
- Participant cards: 2-column CSS grid, `grid-template-columns: repeat(2, 1fr)`.
- Card deck: `flex-wrap: wrap`, 2 rows of 6 cards. Card size: 44px x 56px.
- Card deck area: `position: sticky; bottom: 0; background: var(--color-bg)` with a top border.
- Action buttons: stacked below 320px, same row above.
- Show Votes button: full width.

**Tablet (640px - 1024px):**
- Participant cards: flexbox with wrapping, auto-sized.
- Card deck: single row, may still wrap if viewport is narrow.
- Card size transitions to desktop size (56px x 72px).

**Desktop (> 1024px):**
- Content max-width: 960px, centered.
- Participant cards: flexbox with wrapping. Comfortable for up to 12 participants without scrolling.
- Card deck: single row, no wrapping needed.

### 7.3 Scrolling Behavior

```css
.room-page {
  display: flex;
  flex-direction: column;
  height: 100dvh; /* dynamic viewport height -- avoids mobile address-bar issue */
}

.room-page__participants {
  flex: 1;
  overflow-y: auto;
}

.room-page__bottom {
  flex-shrink: 0; /* card deck + actions never shrink */
}
```

### 7.4 Touch Targets

All interactive elements have a minimum touch target of 44px x 44px. For visually smaller elements, padding extends the tappable area:

```css
.header__settings-button {
  width: 24px;
  height: 24px;
  padding: 10px; /* total tap area: 44px x 44px */
}
```

---

## 8. Accessibility

### 8.1 Overview

This section maps the UX spec's accessibility requirements (Appendix C) to concrete implementation details. The goal is WCAG 2.1 AA compliance for the core flows.

### 8.2 Focus Management

| Context | Behavior |
|---|---|
| NameEntryModal opens | Focus moves to the name input. Tab cycles through input and button(s) only. |
| ConfirmDialog opens | Focus moves to the Cancel button (safe default). Tab cycles between Cancel and Confirm. |
| Modal closes (any) | Focus returns to the element that triggered the modal. Store a ref to the trigger element before opening. |
| Settings dropdown opens | Focus moves to the first menu item. Arrow keys navigate items. Escape closes and returns focus to the gear button. |
| Room page mount | No auto-focus (user may be reading the participant list). |

### 8.3 ARIA Attributes

| Element | ARIA |
|---|---|
| Card | `role="radio"`, `aria-checked`, `aria-disabled`, `aria-label="Vote {value}"` |
| CardDeck | `role="radiogroup"`, `aria-label="Select your vote"` |
| ParticipantCard | `role="listitem"`, `aria-label="{name}, {presence}, {vote status}"` |
| ParticipantList | `role="list"` |
| NameEntryModal | `role="dialog"`, `aria-modal="true"`, `aria-labelledby` |
| ConfirmDialog | `role="alertdialog"`, `aria-modal="true"`, `aria-labelledby`, `aria-describedby` |
| Toast container | `role="status"`, `aria-live="polite"` |
| Copy link button | `aria-label="Copy room link to clipboard"` (changes to "Link copied" after copy) |
| Reconnection banner | `role="alert"` (so screen readers announce connection issues immediately) |

### 8.4 Keyboard Navigation

- All interactive elements are focusable and operable via keyboard (Enter/Space to activate buttons, Enter to submit forms).
- Cards in the CardDeck support arrow-key navigation within the radiogroup.
- Escape closes any open modal or dropdown.
- Tab order follows visual layout (no `tabindex` hacks needed if DOM order matches).

### 8.5 Screen Reader Announcements

State changes that should be announced to screen readers:

| Event | Mechanism |
|---|---|
| Participant joins/leaves | Toast with `aria-live="polite"` |
| Votes revealed | Toast + the results section appearing naturally in DOM order |
| New round started | Toast with `aria-live="polite"` |
| Connection lost/restored | Reconnection banner with `role="alert"` (assertive) |
| Error from server | Toast with `aria-live="polite"` |

### 8.6 Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

Applied globally in `styles/global.css`. Affects card hover/select animations, toast slide-in, presence dot color transitions.

### 8.7 Color Contrast

All text colors meet WCAG AA contrast ratio (4.5:1 for normal text, 3:1 for large text) against their backgrounds. The design token palette from the UX spec was chosen with this in mind:
- `--color-text` (#18181b) on `--color-bg` (#fafafa): ratio ~18:1.
- `--color-text-secondary` (#71717a) on `--color-surface` (#ffffff): ratio ~5.3:1.
- White text on `--color-primary` (#6366f1): ratio ~4.6:1.

---

## 9. Deployment

### 9.1 Strategy: Frontend Embedded in Backend Binary

The frontend is built into static files (`index.html`, `assets/*.js`, `assets/*.css`) and embedded into the backend binary at compile time via Go's `embed.FS`. The backend serves these files directly.

**Rationale:**
- Single binary deployment. No separate web server, no CDN, no CORS.
- Docker image is a single `FROM scratch` or `FROM alpine` with one binary.
- Development uses Vite dev server with a proxy to the backend for WebSocket and API requests.

### 9.2 Build Pipeline

```
frontend/
  npm run build
    --> vite build
    --> output to frontend/dist/
      index.html
      assets/
        main-{hash}.js     (~10-15 KB gzipped, estimated)
        main-{hash}.css     (~3-5 KB gzipped, estimated)
```

### 9.3 Vite Configuration

```typescript
// vite.config.ts
import { defineConfig } from "vite";
import preact from "@preact/preset-vite";

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: "dist",
    assetsDir: "assets",
    sourcemap: false,
    target: "es2020",
  },
  server: {
    port: 5173, // Vite default
    proxy: {
      "/ws": {
        target: "http://localhost:8080",
        ws: true,
      },
      "/api": {
        target: "http://localhost:8080",
      },
    },
  },
});
```

### 9.4 Development Workflow

1. Start backend on port 8080.
2. Start `npm run dev` (Vite) on port 5173.
3. Open `http://localhost:5173`. Vite serves the frontend with HMR. WebSocket and API requests are proxied to the backend.
4. For production: `npm run build`, then the backend serves everything from a single port.

### 9.5 WebSocket URL Resolution

The WebSocket client determines the server URL at runtime:

```typescript
function getWsUrl(roomId: string): string {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/ws/room/${roomId}`;
}
```

No hardcoded URLs. Works in development (proxied through Vite), staging, and production. Works behind reverse proxies that support WebSocket upgrade.

---

## Appendix A: CSS Design Tokens

Derived directly from the UX spec's color palette (Appendix B):

```css
/* styles/tokens.css */
:root {
  /* Colors */
  --color-primary: #6366f1;
  --color-primary-hover: #4f46e5;
  --color-bg: #fafafa;
  --color-surface: #ffffff;
  --color-text: #18181b;
  --color-text-secondary: #71717a;
  --color-border: #e4e4e7;
  --color-success: #22c55e;
  --color-warning: #eab308;
  --color-destructive: #ef4444;

  /* Spacing */
  --space-xs: 4px;
  --space-sm: 8px;
  --space-md: 12px;
  --space-lg: 16px;
  --space-xl: 24px;
  --space-2xl: 32px;

  /* Radii */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-full: 9999px;

  /* Shadows */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.07);
  --shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.1);

  /* Typography */
  --font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  --font-size-sm: 0.875rem;
  --font-size-base: 1rem;
  --font-size-lg: 1.25rem;
  --font-size-xl: 1.5rem;

  /* Card dimensions */
  --card-width: 56px;
  --card-height: 72px;
  --card-width-mobile: 44px;
  --card-height-mobile: 56px;

  /* Transitions */
  --transition-fast: 150ms ease;
  --transition-normal: 300ms ease;
}
```

## Appendix B: Key Technical Decisions Log

| Decision | Chosen | Reason |
|---|---|---|
| Framework | Preact + Signals | Smallest reactive framework with React-like API. Signals eliminate prop drilling. |
| State management | Preact Signals (single `state.ts`) | No context providers, no prop drilling. 11 signals in one file. |
| CSS | Plain CSS + custom properties + BEM naming | Zero runtime cost, zero build dependencies, debuggable in DevTools. |
| Router | 20-line custom implementation | Two routes do not justify a library. |
| Build | Vite | Industry standard, near-zero config for Preact + TS. |
| Deployment | Static files embedded in backend | Single binary, single port, no CORS. |
| Stats computation | Server-side only | Server includes stats in `votes_revealed` and `room_state` during reveal. No `utils/stats.ts`. Eliminates risk of client/server disagreement. |
| Protocol definition | Architecture Doc v3.0, Section 4 | Single source of truth. Frontend plan references, does not redefine. |
