# Technical Planning Session: Room Timer & Clear Room Relocation

Multi-role planning session with 5 iterations. Roles: Frontend Lead, Backend Lead, System Architect, Devil's Advocate, Product Owner / Chief Architect.

---

## Iteration 1: Initial Proposals

### Frontend Lead

**Feature 1: Move Clear Room to Header**

- Remove the Clear Room button and `showClearConfirm` state from `RoomPage.tsx`.
- Add a trash icon button to `Header.tsx` after the Copy Link button.
- Move `showClearConfirm` useState and the `ConfirmDialog` into `Header.tsx`.
- Add `.header__clear-btn` CSS class styled like `.header__edit-btn` but red on hover.
- No protocol changes needed -- still sends `clear_room`.

**Feature 2: Timer Component**

- New component `web/src/components/Timer/Timer.tsx` + `Timer.css`.
- New signal `timerState` in `state.ts` with shape `{ duration: number, state: 'idle'|'running'|'expired', startedAt: number|null, remaining: number }`.
- Timer component uses `useEffect` with `setInterval(1000)` for local countdown when running.
- Three new `ClientMessage` variants: `timer_set_duration`, `timer_start`, `timer_reset`.
- One new `ServerMessage` variant: `timer_updated`.
- Handle `timer_updated` in `ws.ts` switch. Also update timer from `room_state` and `round_reset`.
- Timer placed inside `.room__actions` as a left-aligned group.

### Backend Lead

**Feature 1: Clear Room** -- No backend changes needed.

**Feature 2: Timer**

- Add `Timer` struct to `domain/room.go`:
  ```go
  type Timer struct {
      Duration  int       // seconds, 30-600
      State     string    // "idle", "running", "expired"
      StartedAt time.Time // zero value when idle
  }
  ```
- Add Timer field to `Room` struct, initialized in `NewRoom` with `{Duration: 30, State: "idle"}`.
- Add methods on Room (all require lock held):
  - `SetTimerDuration(d int) error` -- validate 30-600, only allowed in idle state.
  - `StartTimer() error` -- set state to running, record StartedAt.
  - `ResetTimer()` -- set state to idle, zero StartedAt.
  - `TimerSnapshot() TimerState` -- compute remaining = max(0, Duration - elapsed). If running and remaining <= 0, set state to expired.
- `TimerState` struct for serialization: `Duration int`, `State string`, `StartedAt *int64` (unix ms, nil when idle), `Remaining int`.
- Three new handlers in `ws.go`: `handleTimerSetDuration`, `handleTimerStart`, `handleTimerReset`.
- Each handler locks room, mutates timer, calls `TimerSnapshot()`, broadcasts `timer_updated`.
- Modify `handleNewRound` and `handleClearRoom` to call `room.ResetTimer()` and broadcast `timer_updated`.
- Add `Timer TimerState` field to `RoomStatePayload` in `events.go`.
- Fill timer in `BuildRoomState`.

### System Architect

**Concerns:**
1. The `TimerSnapshot()` method has a side effect: it transitions `running` to `expired` when remaining <= 0. This is intentional -- it's a lazy evaluation pattern. But the state mutation happens during a read-like operation. This is acceptable since it's called with lock held and the transition is idempotent.
2. `startedAt` should be sent as unix milliseconds (int64) for JavaScript compatibility. Go's `time.Time` zero value should map to `null` in JSON.
3. The `room_state` payload must include timer so reconnecting clients get the full picture.
4. `round_reset` event does NOT need to carry timer data -- the server sends a separate `timer_updated` message right after. Two messages in sequence is simpler than modifying the round_reset payload.
5. `clear_room` already triggers re-join. The re-join gets a fresh `room_state` with reset timer. So no extra `timer_updated` is needed after clear -- the room_state covers it.

### Devil's Advocate

**Challenges:**
1. Do we really need `timer_set_duration` as a separate event? Could we combine start + duration into one event? **No** -- the UX clearly separates adjusting duration (idle state) from starting. Keep them separate.
2. Do we need a `TimerState` struct separate from `Timer`? We have `Timer` (domain, has `time.Time`) and `TimerState` (serialization, has `*int64` for startedAt). **Yes** -- separation is clean. The domain shouldn't care about JSON wire format.
3. Should we validate that only idle timers can have duration changed? **Yes** -- prevents confusing behavior. The arrows are hidden in the UI during running/expired, but the server should enforce it too.
4. The `remaining` field is redundant when state is `idle` (it equals `duration`). But including it simplifies the client -- it doesn't need special-case logic. **Keep it.**

### Product Owner / Chief Architect

**Feedback:**
- Good start. The lazy expiry detection in `TimerSnapshot()` is elegant -- no background goroutines per room.
- Concern: what if no messages are sent to a room for a long time while timer is running? The timer will appear running on the server but actually expired. **Resolution:** `TimerSnapshot()` handles this -- whenever anyone queries the state (reconnect, any broadcast), it auto-transitions to expired. This is fine.
- For iteration 2: nail down exact message formats and the `room_state` timer field structure.
- The clear_room flow after the move needs careful testing -- make sure the ConfirmDialog still works correctly in the Header context.

---

## Iteration 2: Protocol and Data Structures

### Frontend Lead

**Refined timer signal:**

```typescript
export interface TimerState {
  duration: number;    // seconds
  state: 'idle' | 'running' | 'expired';
  startedAt: number | null;  // unix ms
  remaining: number;   // seconds
}
```

Add to `state.ts`:
```typescript
export const timerState = signal<TimerState>({ duration: 30, state: 'idle', startedAt: null, remaining: 30 });
```

**Extend RoomState interface:**
```typescript
export interface RoomState {
  // ... existing fields ...
  timer: TimerState;
}
```

**New ClientMessage variants:**
```typescript
| { type: 'timer_set_duration'; payload: { duration: number } }
| { type: 'timer_start'; payload: Record<string, never> }
| { type: 'timer_reset'; payload: Record<string, never> }
```

**New ServerMessage variant:**
```typescript
| { type: 'timer_updated'; payload: TimerState }
```

**Handler in ws.ts:**
- On `timer_updated`: set `timerState.value = msg.payload`.
- On `room_state`: set `timerState.value = msg.payload.timer`.
- On `round_reset`: reset timer to `{ duration: timerState.value.duration, state: 'idle', startedAt: null, remaining: timerState.value.duration }`. But actually, the server will send `timer_updated` separately, so the client should just wait for that. However, there's a brief moment between round_reset and timer_updated. To avoid a flash, we can optimistically reset the timer display on round_reset. Actually, both messages arrive in the same WebSocket frame sequence -- they'll be processed back-to-back. No flash. Just let the `timer_updated` handler do its thing.

### Backend Lead

**Exact Go types:**

```go
// domain/room.go
type TimerState string
const (
    TimerIdle    TimerState = "idle"
    TimerRunning TimerState = "running"
    TimerExpired TimerState = "expired"
)

const (
    MinTimerDuration = 30
    MaxTimerDuration = 600
    DefaultTimerDuration = 30
)

type Timer struct {
    Duration  int
    State     TimerState
    StartedAt time.Time
}
```

```go
// server/events.go
type TimerSetDurationPayload struct {
    Duration int `json:"duration"`
}

type TimerStatePayload struct {
    Duration  int    `json:"duration"`
    State     string `json:"state"`
    StartedAt *int64 `json:"startedAt"` // unix ms, null when idle
    Remaining int    `json:"remaining"` // seconds
}
```

**Room methods:**

- `SetTimerDuration(d int) error`: returns error if state != idle or d out of range.
- `StartTimer() error`: returns error if state != idle. Sets State=running, StartedAt=time.Now().
- `ResetTimer()`: unconditionally sets State=idle, StartedAt=zero, preserves Duration.
- `GetTimerSnapshot() TimerStatePayload`: computes remaining, auto-transitions to expired if needed. Returns serialization-ready struct.

**Question: should `GetTimerSnapshot()` return a domain type or a server type?**

It should return a domain-level snapshot struct, and the server layer maps it to the payload. But that adds a redundant struct. Since the domain package already has no dependency on `encoding/json` tags... actually it does have VoteValue etc. Let me keep it simple: add a `TimerSnapshot` struct in domain that the server converts to `TimerStatePayload`. Or just let `GetTimerSnapshot` return `(duration int, state TimerState, startedAt time.Time, remaining int)` as multiple return values and let the server build the payload. That's cleaner.

**Revised approach:** Add a `TimerInfo` method to Room:
```go
func (r *Room) TimerInfo() (duration int, state TimerState, startedAt time.Time, remaining int)
```

The server builds `TimerStatePayload` from these values. This keeps domain free of JSON concerns.

### System Architect

**Protocol summary (final):**

Client -> Server:
| Event | Payload | Notes |
|-------|---------|-------|
| `timer_set_duration` | `{"duration": 60}` | Only in idle state. 30-600. |
| `timer_start` | `{}` | Only in idle state. |
| `timer_reset` | `{}` | Any state -> idle. |

Server -> Client:
| Event | Payload | Notes |
|-------|---------|-------|
| `timer_updated` | `{"duration":60,"state":"idle","startedAt":null,"remaining":60}` | Broadcast on any timer change. |

`room_state` gains `"timer"` field with same shape as `timer_updated` payload.

**Integration points confirmed:**
- `handleNewRound`: after `room.NewRound()`, call `room.ResetTimer()`, broadcast `timer_updated`.
- `handleClearRoom`: `ClearRoom()` should reset timer internally. On re-join via `room_state`, client gets fresh timer.
- `BuildRoomState`: include `TimerStatePayload` from `room.TimerInfo()`.

### Devil's Advocate

**Challenge: should `ClearRoom()` reset timer internally or should the handler do it?**

The handler does it for NewRound, but ClearRoom is different -- it wipes everything. The `ClearRoom()` method should reset the timer to defaults (30s, idle) since it resets the entire room state. This is more robust than relying on the handler.

**Challenge: `TimerInfo()` returning 4 values is unwieldy.**

Fair point. Use a simple struct:
```go
type TimerSnapshot struct {
    Duration  int
    State     TimerState
    StartedAt time.Time
    Remaining int
}
```

This is a domain struct with no JSON tags. The server maps it. Clean enough.

### Product Owner / Chief Architect

**Decisions:**
1. `ClearRoom()` resets timer to defaults internally. Approved.
2. `TimerSnapshot` struct in domain. Approved.
3. `TimerInfo()` method returns `TimerSnapshot`. Approved.
4. Server sends separate `timer_updated` after `round_reset`. Client handles both messages. Approved.
5. No timer data in `round_reset` or `room_cleared` payloads. Timer state comes via `timer_updated` or `room_state`. Approved.

Move to iteration 3 to finalize component structure.

---

## Iteration 3: Component Structure and State Management

### Frontend Lead

**Timer component design:**

```
web/src/components/Timer/
  Timer.tsx
  Timer.css
```

Props: none (reads from `timerState` signal directly).

Internal state:
- `displayRemaining` via `useState<number>` -- local countdown value.
- `useEffect` that runs `setInterval(1000)` when `timerState.value.state === 'running'`.
- On each tick: decrement `displayRemaining`. When it hits 0, stop interval (display expired state).
- When `timerState` signal changes (new server message), sync `displayRemaining` to `timerState.value.remaining`.

**Timer rendering logic:**

```
if state === 'idle':
  [down-arrow] MM:SS [up-arrow] [Start btn]
  
if state === 'running':
  MM:SS [Reset btn]
  (no arrows)

if state === 'expired':
  [bell-icon] [Reset btn]
  (no arrows)
```

**Actions:**
- `handleDurationChange(delta: number)`: sends `timer_set_duration` with `timerState.value.duration + delta`.
- `handleStart()`: sends `timer_start`.
- `handleReset()`: sends `timer_reset`.

**Format helper:** `formatTime(seconds: number): string` -- returns "M:SS" format.

**RoomPage changes:**
- Remove Clear Room button and its state/handlers.
- Import Timer component.
- Place `<Timer />` inside `.room__actions` before the primary action button.
- Restructure `.room__actions` CSS to flex with space-between.

**Header changes:**
- Add trash icon button after Copy Link.
- Add `showClearConfirm` state and ConfirmDialog.
- Import `ConfirmDialog` and `send`.

### Backend Lead

**Handler implementations (ws.go):**

All three handlers follow the same pattern: validate session, get room, lock, mutate, get snapshot, unlock, broadcast.

```go
func handleTimerSetDuration(client *Client, manager *RoomManager, payload json.RawMessage) {
    // validate joined
    // unmarshal TimerSetDurationPayload
    // room.Lock()
    // room.SetTimerDuration(p.Duration)
    // snapshot := room.TimerInfo()
    // room.Unlock()
    // broadcast timer_updated with snapshot
}
```

Same pattern for `handleTimerStart` and `handleTimerReset` (no payload needed for these two).

**Dispatch additions:**
```go
case "timer_set_duration":
    handleTimerSetDuration(client, manager, env.Payload)
case "timer_start":
    handleTimerStart(client, manager)
case "timer_reset":
    handleTimerReset(client, manager)
```

**Modification to handleNewRound:**
After `room.NewRound()`, before `room.Unlock()`:
```go
room.ResetTimer()
timerSnapshot := room.TimerInfo()
```
After unlock, broadcast both `round_reset` and `timer_updated`.

**Modification to ClearRoom domain method:**
```go
func (r *Room) ClearRoom() {
    r.Participants = make(map[string]*Participant)
    r.Phase = PhaseVoting
    r.Timer = Timer{Duration: DefaultTimerDuration, State: TimerIdle}
    r.TouchActivity()
}
```

**Modification to BuildRoomState:**
```go
func (rm *RoomManager) BuildRoomState(room *domain.Room) RoomStatePayload {
    // ... existing code ...
    snapshot := room.TimerInfo()
    state.Timer = mapTimerSnapshot(snapshot)
    return state
}
```

### System Architect

**Sequence diagrams (key flows):**

**Timer start flow:**
1. Client sends `timer_start`
2. Server: lock room -> `StartTimer()` -> `TimerInfo()` -> unlock -> broadcast `timer_updated`
3. All clients receive `timer_updated` with `state: "running"`, `remaining: N`, `startedAt: <ms>`
4. Each client starts local countdown interval

**New Round flow:**
1. Client sends `new_round`
2. Server: lock room -> `NewRound()` -> `ResetTimer()` -> `TimerInfo()` -> unlock -> broadcast `round_reset` then `timer_updated`
3. Clients process `round_reset` (clear votes), then `timer_updated` (reset timer display)

**Reconnect flow:**
1. Client sends `join` (re-join)
2. Server: `BuildRoomState` includes timer snapshot
3. Client receives `room_state` with timer field
4. Client sets `timerState` signal from `room_state.timer`
5. If timer is running, client starts local countdown from `remaining`

All flows are consistent. No race conditions because room mutex serializes all operations.

### Devil's Advocate

**Challenge: Do we need `startedAt` in the wire format?**

The client only needs `remaining` and `state` to render correctly. `startedAt` is extra data. However, it could be useful for debugging and for clients that want to compute drift. It's one field. Keep it -- negligible cost.

**Challenge: The Timer component reads from a signal but also has local state (`displayRemaining`). Is this a code smell?**

No. The signal is the server-authoritative state. The local state is for smooth 1-second countdown rendering. This is the standard pattern for real-time countdowns. The signal acts as a sync point.

### Product Owner / Chief Architect

**Approved.** Component structure is clean. One concern: make sure the `useEffect` cleanup properly clears the interval on unmount and when `timerState` changes. This is standard Preact/React cleanup -- the Frontend Lead knows this.

Move to iteration 4: error handling, tests, edge cases.

---

## Iteration 4: Error Handling, Tests, Edge Cases

### Frontend Lead

**Error handling:**
- `timer_set_duration` with invalid value: server sends `error` event, client shows toast. No client-side pre-validation needed beyond disabling arrows at min/max.
- Arrows disabled: down arrow disabled when `duration <= 30`, up arrow disabled when `duration >= 600`.
- Start button: no guard needed -- server rejects if already running.
- Timer display during disconnect: timer freezes (interval keeps running locally but reconnect will sync). On reconnect, `room_state` re-syncs everything.

**Tests (frontend):**
- Timer component unit test: renders idle state with correct format.
- Timer component: clicking start sends correct message.
- Timer component: arrows disabled at boundaries.
- Timer component: local countdown decrements correctly (mock setInterval).
- ws.ts: `timer_updated` handler updates `timerState` signal.
- ws.ts: `room_state` handler extracts timer field.
- Header: clear room button renders, click opens ConfirmDialog, confirm sends `clear_room`.
- RoomPage: no longer renders Clear Room button.

### Backend Lead

**Error cases in handlers:**
- `handleTimerSetDuration`: not joined -> error. Invalid duration (< 30 or > 600 or not multiple of 30) -> error. Timer not idle -> error.
- `handleTimerStart`: not joined -> error. Timer not idle -> error.
- `handleTimerReset`: not joined -> error. Timer already idle -> no-op (no error, no broadcast -- already in desired state).

**Wait -- should duration be constrained to multiples of 30?** The UX adjusts in 30s increments, but should the server enforce multiples of 30? The server should validate range (30-600) but not enforce multiples. The client sends exact values (30, 60, 90, ...) but if somehow a non-multiple arrives, it's still valid. KISS -- just validate range.

**Tests (backend):**

Domain tests (`domain/room_test.go`):
- `TestSetTimerDuration`: valid range, out of range, non-idle state.
- `TestStartTimer`: from idle, from running (error), from expired (error).
- `TestResetTimer`: from each state.
- `TestTimerInfo`: idle returns duration=remaining, running computes remaining, auto-expires when remaining <= 0.
- `TestNewRound_ResetsTimer`: new round resets timer to idle.
- `TestClearRoom_ResetsTimer`: clear room resets timer to defaults.

Handler tests (`server/ws_test.go` or `server/handlers_test.go`):
- Integration test: send `timer_set_duration`, verify broadcast contains correct payload.
- Integration test: send `timer_start`, verify broadcast has running state.
- Integration test: send `timer_reset`, verify broadcast has idle state.
- Integration test: send `new_round`, verify both `round_reset` and `timer_updated` are broadcast.

### System Architect

**Edge case: timer_set_duration with duration not a multiple of 30.**

Decision: server accepts any integer 30-600. The client always sends multiples of 30 via the UI arrows, but the server is lenient. This follows Postel's Law.

**Edge case: two clients send timer_start simultaneously.**

Server processes them sequentially (room mutex). First succeeds, second gets error "timer already running". Both clients receive the `timer_updated` broadcast from the first. The second client's error toast is harmless but slightly noisy. Alternative: make `StartTimer()` return success even if already running (idempotent). The broadcast would be redundant but harmless.

Decision: make `StartTimer()` a no-op if already running (no error). Same for `ResetTimer()` if already idle. This eliminates spurious error toasts in concurrent scenarios.

**Edge case: `timer_updated` arrives before `room_state` on initial join.**

This cannot happen. The server sends `room_state` first (directly to the joining client), then any subsequent `timer_updated` broadcasts. The client won't have a Timer component rendered until `roomState` is set.

### Devil's Advocate

**Challenge: should timer_start and timer_reset be idempotent or return errors?**

Idempotent is better for concurrent UX. No user expects an error toast for pressing Start when another user already pressed it 100ms earlier. Make them idempotent: start when already running -> no-op (no broadcast). Reset when already idle -> no-op (no broadcast).

**Challenge: do we need to broadcast timer_updated for no-ops?**

No. If the state didn't change, don't broadcast. This avoids unnecessary network traffic and re-renders.

### Product Owner / Chief Architect

**Decisions:**
1. `StartTimer()` is idempotent -- no error if already running, no broadcast. Approved.
2. `ResetTimer()` is idempotent -- no error if already idle, no broadcast. Approved.
3. Server validates duration range 30-600 but does NOT enforce multiples of 30. Approved.
4. Test plan covers domain methods and handler integration. Approved.

Final iteration: implementation order, docs.

---

## Iteration 5: Implementation Order and Documentation

### Frontend Lead

**Implementation order (frontend):**
1. Move Clear Room to Header (small, isolated change -- can be tested independently).
2. Add `TimerState` type and `timerState` signal to `state.ts`.
3. Add `timer_updated` handler to `ws.ts` and extend `room_state` handler.
4. Add `ClientMessage` variants for timer events.
5. Build `Timer` component (TSX + CSS).
6. Integrate Timer into RoomPage layout.
7. Test all flows.

### Backend Lead

**Implementation order (backend):**
1. Add `Timer` struct and constants to `domain/room.go`.
2. Add timer methods to Room: `SetTimerDuration`, `StartTimer`, `ResetTimer`, `TimerInfo`.
3. Initialize timer in `NewRoom` and reset in `ClearRoom`, `NewRound`.
4. Write domain tests.
5. Add payload types to `events.go`.
6. Add handlers to `ws.go` and dispatch cases.
7. Modify `BuildRoomState` and `handleNewRound`.
8. Write handler tests.

### System Architect

**Recommended overall order:**
1. Backend domain layer (Timer struct, methods, tests).
2. Backend protocol layer (events, handlers, BuildRoomState).
3. Frontend state and ws handler changes.
4. Frontend Timer component.
5. Frontend Header changes (Clear Room move).
6. Integration testing (manual: two browser tabs).
7. Documentation updates.

Steps 1-2 can be one commit. Steps 3-5 can be another. Step 5 is independent and could go first as a separate small commit.

### Devil's Advocate

The plan is tight. No unnecessary abstractions. The only thing I'd flag: the Timer CSS should not introduce new CSS custom properties. Use existing design system variables. Confirmed in the UX spec -- it references existing `--color-primary`, `--color-status-idle`, `--color-text`, etc.

### Product Owner / Chief Architect

**Final approval.** The plan is complete. Proceed to implementation plan.

---

## IMPLEMENTATION PLAN

### 1. File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/domain/room.go` | Modify | Add Timer struct, constants, methods, integrate into NewRoom/ClearRoom/NewRound |
| `internal/domain/room_test.go` | Modify | Add timer-related tests |
| `internal/server/events.go` | Modify | Add TimerSetDurationPayload, TimerStatePayload; add Timer field to RoomStatePayload |
| `internal/server/ws.go` | Modify | Add 3 dispatch cases, 3 handlers, modify handleNewRound, helper to build timer payload |
| `web/src/state.ts` | Modify | Add TimerState interface, timerState signal, extend RoomState, extend ClientMessage/ServerMessage |
| `web/src/ws.ts` | Modify | Handle timer_updated, extract timer from room_state, reset on round_reset |
| `web/src/components/Timer/Timer.tsx` | Create | New timer component |
| `web/src/components/Timer/Timer.css` | Create | Timer styles |
| `web/src/components/Header/Header.tsx` | Modify | Add Clear Room button + ConfirmDialog |
| `web/src/components/Header/Header.css` | Modify | Add .header__clear-btn styles |
| `web/src/components/RoomPage/RoomPage.tsx` | Modify | Remove Clear Room button, add Timer, restructure actions layout |
| `web/src/components/RoomPage/RoomPage.css` | Modify | Update .room__actions layout for timer + action button |

### 2. New Go Types and Methods

#### `internal/domain/room.go`

```go
// Timer state constants
type TimerState string
const (
    TimerIdle    TimerState = "idle"
    TimerRunning TimerState = "running"
    TimerExpired TimerState = "expired"
)

const (
    MinTimerDuration     = 30
    MaxTimerDuration     = 600
    DefaultTimerDuration = 30
)

// Timer holds the countdown timer state for a room.
type Timer struct {
    Duration  int        // seconds, 30-600
    State     TimerState
    StartedAt time.Time  // zero when idle/expired
}

// TimerSnapshot is a point-in-time view of timer state.
type TimerSnapshot struct {
    Duration  int
    State     TimerState
    StartedAt time.Time
    Remaining int // seconds
}
```

**New field on Room struct:**
```go
Timer Timer
```

**Initialized in NewRoom:**
```go
Timer: Timer{Duration: DefaultTimerDuration, State: TimerIdle},
```

**New methods on Room (all require lock held):**

```go
// SetTimerDuration sets the timer duration. Only allowed in idle state.
func (r *Room) SetTimerDuration(d int) error

// StartTimer begins the countdown. Only allowed in idle state. No-op if already running.
func (r *Room) StartTimer() bool // returns true if state changed

// ResetTimer returns timer to idle with current duration. No-op if already idle.
func (r *Room) ResetTimer() bool // returns true if state changed

// TimerInfo returns a point-in-time snapshot. Auto-transitions running->expired if time is up.
func (r *Room) TimerInfo() TimerSnapshot
```

**Modified methods:**
- `NewRound()`: add `r.ResetTimer()` call.
- `ClearRoom()`: reset Timer to `Timer{Duration: DefaultTimerDuration, State: TimerIdle}`.

#### `internal/server/events.go`

```go
// Client -> Server
type TimerSetDurationPayload struct {
    Duration int `json:"duration"`
}

// Server -> Client (also embedded in RoomStatePayload)
type TimerStatePayload struct {
    Duration  int    `json:"duration"`
    State     string `json:"state"`
    StartedAt *int64 `json:"startedAt"` // unix ms, null when idle
    Remaining int    `json:"remaining"` // seconds
}
```

**Modified struct:**
```go
type RoomStatePayload struct {
    RoomID       string              `json:"roomId"`
    RoomName     string              `json:"roomName"`
    CreatedBy    string              `json:"createdBy"`
    Phase        domain.Phase        `json:"phase"`
    Participants []ParticipantInfo   `json:"participants"`
    Result       *domain.RoundResult `json:"result"`
    Timer        TimerStatePayload   `json:"timer"`
}
```

#### `internal/server/ws.go`

**New helper:**
```go
func buildTimerPayload(s domain.TimerSnapshot) TimerStatePayload
```

Converts `domain.TimerSnapshot` to `TimerStatePayload`. Maps `time.Time` to `*int64` (unix ms) or nil if zero.

**New handlers:**
```go
func handleTimerSetDuration(client *Client, manager *RoomManager, payload json.RawMessage)
func handleTimerStart(client *Client, manager *RoomManager)
func handleTimerReset(client *Client, manager *RoomManager)
```

**New dispatch cases:**
```go
case "timer_set_duration":
    handleTimerSetDuration(client, manager, env.Payload)
case "timer_start":
    handleTimerStart(client, manager)
case "timer_reset":
    handleTimerReset(client, manager)
```

**Modified handler: `handleNewRound`**
After `room.NewRound()` and before unlock:
```go
room.ResetTimer()
timerSnapshot := room.TimerInfo()
```
After unlock, broadcast `timer_updated` in addition to `round_reset`.

**Modified function: `BuildRoomState`**
After building participants, add:
```go
snapshot := room.TimerInfo()
state.Timer = buildTimerPayload(snapshot)
```

### 3. New TypeScript Types, Signals, and Components

#### `web/src/state.ts`

**New interface:**
```typescript
export interface TimerState {
  duration: number;      // seconds
  state: 'idle' | 'running' | 'expired';
  startedAt: number | null;  // unix ms
  remaining: number;     // seconds
}
```

**New signal:**
```typescript
export const timerState = signal<TimerState>({
  duration: 30, state: 'idle', startedAt: null, remaining: 30
});
```

**Extend RoomState:**
```typescript
export interface RoomState {
  roomId: string;
  roomName: string;
  createdBy: string;
  phase: 'voting' | 'reveal';
  participants: Participant[];
  result: VoteResult | null;
  timer: TimerState;         // NEW
}
```

**Extend ServerMessage union:**
```typescript
| { type: 'timer_updated'; payload: TimerState }
```

**Extend ClientMessage union:**
```typescript
| { type: 'timer_set_duration'; payload: { duration: number } }
| { type: 'timer_start'; payload: Record<string, never> }
| { type: 'timer_reset'; payload: Record<string, never> }
```

#### `web/src/ws.ts`

**New case in handleMessage switch:**
```typescript
case 'timer_updated': {
  timerState.value = msg.payload;
  break;
}
```

**Modified case `room_state`:**
Add after setting roomState:
```typescript
if (msg.payload.timer) {
  timerState.value = msg.payload.timer;
}
```

#### `web/src/components/Timer/Timer.tsx`

New component. Reads `timerState` signal. Contains:
- `displayRemaining` local state for smooth countdown.
- `useEffect` with `setInterval(1000)` when state is running.
- `useEffect` to sync `displayRemaining` from `timerState.value.remaining` when signal changes.
- `formatTime(seconds)` helper: returns `M:SS` string.
- Renders based on `timerState.value.state`:
  - **idle**: `[down-chevron] M:SS [up-chevron] [Start button]`
  - **running**: `M:SS [Reset button]`
  - **expired**: `[bell icon] [Reset button]`
- Down arrow disabled when `duration <= 30`.
- Up arrow disabled when `duration >= 600`.

#### `web/src/components/Timer/Timer.css`

New stylesheet. Classes:
- `.timer` -- flex container, align-items center, gap.
- `.timer__display` -- monospace font, 18px, font-weight 600.
- `.timer__display--running` -- color: var(--color-primary).
- `.timer__display--expired` -- color: var(--color-status-idle).
- `.timer__arrow-btn` -- small chevron button, ghost style.
- `.timer__arrow-btn:disabled` -- dimmed.
- `.timer__action-btn` -- pill button for start/reset.

### 4. WebSocket Protocol Additions

**Client -> Server:**

```json
{ "type": "timer_set_duration", "payload": { "duration": 60 } }
```
Sets timer duration in seconds. Must be 30-600. Only in idle state.

```json
{ "type": "timer_start", "payload": {} }
```
Starts the countdown. Idempotent -- no-op if already running.

```json
{ "type": "timer_reset", "payload": {} }
```
Resets timer to idle with current duration. Idempotent -- no-op if already idle.

**Server -> Client:**

```json
{
  "type": "timer_updated",
  "payload": {
    "duration": 60,
    "state": "running",
    "startedAt": 1712438400000,
    "remaining": 45
  }
}
```
Broadcast on any timer state change.

**Modified `room_state` payload:**
```json
{
  "type": "room_state",
  "payload": {
    "roomId": "...",
    "roomName": "...",
    "createdBy": "...",
    "phase": "voting",
    "participants": [...],
    "result": null,
    "timer": {
      "duration": 30,
      "state": "idle",
      "startedAt": null,
      "remaining": 30
    }
  }
}
```

### 5. Integration Points

| Existing Code | Change |
|---------------|--------|
| `handleNewRound` in ws.go | After `room.NewRound()`, timer is auto-reset. Get snapshot, broadcast `timer_updated` after `round_reset`. |
| `Room.ClearRoom()` in room.go | Reset timer to defaults (30s, idle) inside the method. |
| `BuildRoomState` in room_manager.go | Include `TimerStatePayload` from `room.TimerInfo()`. |
| `dispatch` in ws.go | Add 3 new cases. |
| `handleMessage` in ws.ts | Add `timer_updated` case. Extend `room_state` case. |
| `RoomPage.tsx` | Remove Clear Room button. Add `<Timer />`. Restructure `.room__actions`. |
| `Header.tsx` | Add Clear Room icon button + ConfirmDialog. |
| `state.ts` | Add TimerState type, timerState signal, extend RoomState/ClientMessage/ServerMessage. |

### 6. Test Plan

**Backend unit tests (domain/room_test.go):**
- `TestTimer_SetDuration_Valid` -- set 30, 60, 300, 600.
- `TestTimer_SetDuration_OutOfRange` -- set 0, 29, 601. Expect error.
- `TestTimer_SetDuration_NotIdle` -- set while running. Expect error.
- `TestTimer_Start_FromIdle` -- state becomes running, StartedAt set.
- `TestTimer_Start_AlreadyRunning` -- returns false (no-op).
- `TestTimer_Reset_FromRunning` -- state becomes idle, StartedAt zeroed.
- `TestTimer_Reset_AlreadyIdle` -- returns false (no-op).
- `TestTimer_Info_Idle` -- remaining equals duration.
- `TestTimer_Info_Running` -- remaining = duration - elapsed.
- `TestTimer_Info_AutoExpire` -- if elapsed > duration, state transitions to expired, remaining = 0.
- `TestNewRound_ResetsTimer` -- after NewRound, timer is idle.
- `TestClearRoom_ResetsTimerToDefaults` -- after ClearRoom, timer is idle with default duration.

**Backend handler tests (if test infrastructure exists):**
- Send `timer_set_duration` -> verify broadcast payload shape.
- Send `timer_start` -> verify broadcast has running state.
- Send `timer_reset` -> verify broadcast has idle state.
- Send `new_round` -> verify both `round_reset` and `timer_updated` broadcast.
- Send `timer_start` without joining -> verify error.

**Frontend tests:**
- Timer component renders idle state correctly.
- Timer arrows send correct messages and are disabled at boundaries.
- Start/Reset buttons send correct messages.
- Local countdown decrements displayed time.
- `timer_updated` handler updates signal.
- Header Clear Room button opens ConfirmDialog.
- Clear Room confirm sends `clear_room` message.

### 7. Implementation Order

1. **Backend domain** (room.go): Timer struct, constants, methods, modify NewRoom/ClearRoom/NewRound. Write domain tests. (~1 hour)
2. **Backend protocol** (events.go, ws.go, room_manager.go): Payload types, handlers, dispatch, BuildRoomState modification. Write handler tests. (~1 hour)
3. **Frontend state** (state.ts): TimerState type, signal, extend RoomState/ClientMessage/ServerMessage. (~15 min)
4. **Frontend ws handler** (ws.ts): Handle timer_updated, extract timer from room_state. (~15 min)
5. **Frontend Timer component** (Timer.tsx, Timer.css): Build component with local countdown. (~1 hour)
6. **Frontend Header change** (Header.tsx, Header.css): Add Clear Room button + ConfirmDialog. (~30 min)
7. **Frontend RoomPage change** (RoomPage.tsx, RoomPage.css): Remove old Clear Room, add Timer, restructure actions. (~30 min)
8. **Integration testing**: Two browser tabs, test all timer flows + clear room from header. (~30 min)

Total estimate: ~5 hours.

### 8. AI Documentation Plan

After implementation, update/create these docs for future AI agents:

| Document | Action |
|----------|--------|
| `docs/planning/04-architecture.md` Section 4 (WebSocket Protocol) | Add 3 client events and 1 server event. Update room_state payload spec with timer field. |
| `CLAUDE.md` WebSocket Protocol section | Add timer events to the key events lists. |
| `docs/planning/ux-timer-and-clear-room.md` | Add a "Status: Implemented" header. No other changes needed -- it's the source spec. |

No new documentation files needed. Existing docs cover the architecture; they just need the protocol additions.
