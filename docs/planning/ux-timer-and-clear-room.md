# UX Specification: Room Timer & Clear Room Relocation

## Overview

Two changes to the Room page:

1. **Move "Clear Room" from the action area to the Header** -- it is a rare, destructive action that does not belong alongside the core voting flow.
2. **Add a shared countdown timer** to the center area, helping teams timebox discussion before voting.

---

## 1. User Stories

### Clear Room (relocated)

| ID | Story |
|----|-------|
| CR-1 | As a participant, I can clear the room from a button in the header so that the voting area stays focused on the current round. |
| CR-2 | As a participant, I see a confirmation dialog before the room is actually cleared, preventing accidental data loss. |

### Room Timer

| ID | Story |
|----|-------|
| T-1 | As a participant, I can see a shared countdown timer in the center area showing the currently configured duration. |
| T-2 | As a participant, I can adjust the timer duration in 30-second increments (30 s -- 10 min) and the change is broadcast to everyone in the room. |
| T-3 | As a participant, I can start the timer. It counts down to zero for all participants simultaneously. |
| T-4 | As a participant, when the timer reaches zero I see a visual alarm indicator (bell icon) instead of "00:00". |
| T-5 | As a participant, I can reset the timer after it expires (or while it is running). It returns to the room's configured duration in idle state. |
| T-6 | As a participant, when someone presses "New Round" the timer automatically resets to the configured duration in idle state. |
| T-7 | As an observer, I can see the timer and its state but I can also start, reset, and adjust it (observers facilitate). |
| T-8 | As a reconnecting participant, I see the current timer state (running with correct remaining time, expired, or idle) immediately upon reconnecting. |

---

## 2. UX Flows

### 2.1 Clear Room from Header

1. User clicks the "Clear Room" icon button in the header (positioned after "Copy Link").
2. A `ConfirmDialog` appears: title "Clear Room?", message "This will remove all participants."
3. **Confirm**: sends `clear_room` message, dialog closes.
4. **Cancel**: dialog closes, no action.

Flow is identical to today, only the trigger location changes.

### 2.2 Setting Timer Duration

1. Timer displays the current duration (default "0:30") in idle state.
2. User clicks the **up arrow** above or to the right of the display. Duration increases by 30 s (e.g., 0:30 -> 1:00). Broadcast to all.
3. User clicks the **down arrow**. Duration decreases by 30 s. Minimum 0:30 -- the down arrow is disabled / visually dimmed at minimum. Maximum 10:00 -- the up arrow is disabled at maximum.
4. All participants see the new duration immediately.

### 2.3 Starting the Timer

1. User clicks the **Start** (play icon) button next to the timer display.
2. Timer transitions from idle to running state. The display counts down in real time.
3. The Start button is replaced by a **Reset** (stop/reset icon) button.
4. Duration adjustment arrows are hidden while the timer is running to avoid confusion.

### 2.4 Timer Running and Reaching Zero

1. Timer counts down: "0:30" -> "0:29" -> ... -> "0:01" -> "0:00".
2. At "0:00", the timer transitions to expired state:
   - The time display is replaced by a bell/alarm icon.
   - The timer area gets a subtle attention color (use `--color-status-idle` / amber tint).
3. The Reset button remains visible.

### 2.5 Resetting the Timer

1. User clicks **Reset** (available during running or expired states).
2. Timer returns to idle state showing the room's configured duration.
3. The Reset button is replaced by the Start button. Duration arrows reappear.

### 2.6 New Round Interaction with Timer

1. User clicks "New Round".
2. Round resets as usual (votes cleared, phase -> voting).
3. Timer automatically resets to idle state with the room's configured duration, regardless of whether it was running or expired.

---

## 3. UI Layout

### 3.1 Header Changes

Current header layout (left to right):

```
[om] [Room Name by Creator]          [Voting] [UserName] [edit] [Copy Link]
```

New layout -- add a small danger-styled icon button after Copy Link:

```
[om] [Room Name by Creator]          [Voting] [UserName] [edit] [Copy Link] [Clear Room icon]
```

**Clear Room button details:**
- Icon-only button (trash can SVG, 14x14), no text label.
- Styled like `header__edit-btn` (ghost button) but uses `--color-status-disconnected` (red) on hover.
- `title="Clear Room"` for accessibility.
- Same padding as the edit-name button (`--space-xs`).

### 3.2 Timer Placement

The timer occupies the space where Clear Room used to be, inside `.room__actions` but as a distinct visual group. Layout:

```
                    [Participant List]

  [v] 0:30 [^]  [>Start]          [Show Votes (2 of 4 voted)]

                      [Card Deck]
```

During reveal phase:

```
                    [Participant List]
                    [Vote Results]

  [v] 0:30 [^]  [>Start]          [New Round]

                      [Card Deck]
```

**Structural change to `.room__actions`:**

The actions area becomes a flex row with `justify-content: space-between` (or center with a gap). Two groups:

- **Left group**: Timer widget (arrows + display + start/reset button).
- **Right group**: Primary action button (Show Votes / New Round).

On narrow screens (< 480px), the layout stacks vertically: timer on top, action button below, both centered.

### 3.3 Timer Widget Anatomy

```
[ down-arrow ]  MM:SS  [ up-arrow ]  [ start/reset-btn ]
```

- **Down/Up arrows**: Small chevron buttons (similar to the header edit-btn style). Vertically stacked or horizontally placed. Horizontal placement (left for down, right for up) is simpler and works better on mobile.
- **Time display**: Monospace font (`--font-mono`), 18px, font-weight 600. Color: `--color-text` when idle, `--color-primary` when running, `--color-status-idle` (amber) when expired.
- **Start button**: Small pill button with play triangle icon. Styled like `room__action-btn--secondary`.
- **Reset button**: Same position/size as Start, with a reset/stop icon. Same styling.
- **Bell icon**: Replaces the MM:SS text when expired. Same size (18px). Color: `--color-status-idle`.

Total timer widget width: approximately 180-200px. Does not dominate the action area.

### 3.4 Mobile Responsiveness

- **>= 640px**: Timer and action button side by side in the actions row.
- **< 640px**: Timer stacks above the action button, both centered. Timer widget remains a single horizontal row internally (arrows + display + button fit easily at 200px).
- **< 360px**: Timer arrows can shrink; the layout still works because the widget is compact.

The header already wraps naturally via flex. The Clear Room icon button adds only ~30px of width, which is negligible.

---

## 4. Timer States

### 4.1 Idle (default)

- Display: configured duration in `M:SS` format (e.g., "0:30", "2:00").
- Font color: `--color-text`.
- Arrows: visible, enabled (unless at min/max).
- Button: Start (play icon).
- Background: none / transparent.

### 4.2 Running

- Display: remaining time counting down, `M:SS` format.
- Font color: `--color-primary` (indigo) to indicate active state.
- Arrows: hidden (duration locked while running).
- Button: Reset (stop icon).
- Background: none.

### 4.3 Expired

- Display: bell/alarm icon instead of time.
- Icon color: `--color-status-idle` (amber).
- Arrows: hidden.
- Button: Reset (stop icon).
- Background: subtle amber tint (optional -- `--color-warning-bg` at low opacity, or just the icon color is enough).

---

## 5. WebSocket Protocol Additions

### Client -> Server

| Event | Payload | Description |
|-------|---------|-------------|
| `timer_set_duration` | `{ "duration": 60 }` | Set timer duration in seconds. Server validates 30-600 range. |
| `timer_start` | `{}` | Start the countdown. Server records start timestamp. |
| `timer_reset` | `{}` | Reset timer to idle with room's configured duration. |

### Server -> Client

| Event | Payload | Description |
|-------|---------|-------------|
| `timer_updated` | `{ "duration": 60, "state": "idle" \| "running" \| "expired", "startedAt": null \| <unix_ms>, "remaining": 60 }` | Broadcast on any timer change. `remaining` is computed by server at send time. |

**Integration with existing events:**

- `room_state` payload gains a `timer` field: `{ "duration": 60, "state": "idle" | "running" | "expired", "startedAt": null | <unix_ms>, "remaining": 60 }`.
- `round_reset` implicitly resets the timer; the server sends a `timer_updated` alongside (or includes timer state in the reset broadcast).

### Client-Side Timer Rendering

The client does NOT poll the server each second. On receiving a `timer_updated` with `state: "running"`, the client:

1. Notes `remaining` (server-authoritative value at message time).
2. Starts a local `setInterval(1000)` that decrements the displayed value.
3. When local countdown hits 0, transitions display to expired state.
4. If a new `timer_updated` arrives (from reset, or new round), the client syncs to the server value.

This keeps the timer visually smooth without constant WebSocket traffic.

---

## 6. Edge Cases

### 6.1 Duration adjustment while timer is running

**Decision**: Arrows are hidden during running state. Duration cannot be changed while the timer runs. User must reset first, then adjust, then start again. This avoids confusing mid-countdown changes.

### 6.2 Reconnect during running timer

The `room_state` message includes the timer field with `state: "running"`, `startedAt`, and `remaining` (computed by server at send time). The client starts its local countdown from `remaining`. The user sees the correct time within 1 second of accuracy.

### 6.3 Timer expires during reveal phase

Nothing special. The timer and the voting flow are independent. The bell icon appears. The team can discuss results at their own pace. When they click "New Round", the timer resets.

### 6.4 Observer interactions with timer

Observers CAN start, reset, and adjust the timer. In Scrum, the Scrum Master often observes rather than votes, and they are the ones who facilitate timeboxing. Restricting timer controls to voters only would be counterproductive.

### 6.5 Multiple users pressing start/reset simultaneously

The server is authoritative. All timer operations are serialized through the room mutex. If two users press Start at the same instant, the server processes both but the second is a no-op (timer is already running). The resulting `timer_updated` broadcast is identical for both. No race condition visible to users.

### 6.6 Timer drift across clients

Each client runs its own local interval. Over 10 minutes, drift could be 1-2 seconds. This is acceptable for Scrum Poker. If the server detects expiry (e.g., via a lazy check on next message), it broadcasts `timer_updated` with `state: "expired"`, which re-syncs all clients.

### 6.7 Room with no participants, timer running

If all participants disconnect while the timer runs, the timer state persists in the room's in-memory state. When someone reconnects, they get the current state via `room_state`. The server does not need a background ticker per room -- it computes `remaining` on demand: `remaining = max(0, duration - (now - startedAt))`. If remaining <= 0, state is "expired".

### 6.8 Clear Room interaction with timer

Clear Room removes all participants. On re-join (which happens automatically via the existing `room_cleared` handler), the room state is fresh. The timer should reset to default (30s, idle). The server resets timer state as part of `clear_room` handling.

---

## 7. Scrum Process Fit

### 7.1 Is 30 seconds the right default?

**Yes, for this tool's purpose.** The timer in Scrum Poker is not for the full discussion -- it is for the "think and vote" phase after the item has been presented. 30 seconds is a good nudge: enough time to pick a card, short enough to keep the session moving. Teams that need more time can bump to 1:00 or 2:00.

For reference: Planning Poker best practices suggest 30-60 seconds for individual voting after discussion. The full discussion timebox (typically 2-5 minutes per story) is managed by the Scrum Master separately.

### 7.2 Should the timer integrate with voting flow?

**No automatic integration** (e.g., auto-reveal when timer expires). Reasons:

- Auto-reveal would surprise participants who haven't voted yet.
- Some teams use the timer as a soft nudge, not a hard deadline.
- Keeping timer and voting independent preserves simplicity (KISS).

The only integration is: "New Round" resets the timer. This is natural and expected.

### 7.3 Missing features worth considering (future, not in this iteration)

- **Auto-start on New Round**: Option to automatically start the timer when a new round begins. Useful for fast-paced sessions. Deferred -- adds a setting that complicates the UI.
- **Sound on expiry**: A short beep. Explicitly excluded for now (the spec says visual only), but could be a future toggle.
- **Per-room default duration**: Currently the default is always 30s. If teams consistently use 2 minutes, they have to adjust every session. Low priority -- the adjustment is fast enough.

None of these are needed for v1. The timer is a lightweight aid, not a process enforcement tool.

---

## 8. Implementation Notes

### Backend (Go)

- Add `Timer` struct to `domain.Room`: `Duration int` (seconds), `State string` (idle/running/expired), `StartedAt time.Time`.
- Add methods: `SetTimerDuration(d int)`, `StartTimer()`, `ResetTimer()`, `TimerSnapshot() TimerState` (computes remaining on the fly).
- Handle three new client message types in `ws.go`.
- Include timer in `RoomStatePayload`.
- On `new_round` and `clear_room`, call `ResetTimer()`.

### Frontend (Preact)

- New component: `Timer/Timer.tsx` + `Timer/Timer.css`. Renders inside `.room__actions`.
- New signals in `state.ts`: `timerState` (idle/running/expired), `timerDuration`, `timerRemaining`.
- Handle `timer_updated` in `ws.ts` message handler.
- Local countdown via `setInterval` in the Timer component (managed with `useEffect`).
- Move Clear Room button and its confirm dialog state into `Header.tsx`.

### Estimated Scope

- Backend: ~100-150 lines of new/modified code.
- Frontend: ~150-200 lines (new Timer component + wiring).
- Total: Small feature, fits in one PR.
