# Fix [55]: Data Race on Room.LastActivity

## Problem

In `collectGarbage()` (room_manager.go:53-66), `room.LastActivity` is read while holding
only `RoomManager.mu`. But `LastActivity` is written in `Room` methods (Join, Leave,
CastVote, Reveal, NewRound, ClearRoom, UpdateName, UpdatePresence) which require
`room.mu` to be held. This is a formal data race under the Go memory model.

## Approach: `atomic.Int64` for LastActivity

Replace `LastActivity time.Time` with a private `lastActivity atomic.Int64` storing
`time.UnixNano()`. Provide accessor methods:

- `TouchActivity()` — stores `time.Now().UnixNano()` atomically
- `GetLastActivity() time.Time` — returns `time.Unix(0, nanos)` from atomic load
- `LastActivityUnixNano() int64` — raw atomic load for GC comparison

### Why atomic over room.Lock() in GC?

1. **No lock ordering risk.** Taking `room.mu` inside `rm.mu` works today, but it's a
   fragile ordering constraint. Atomics eliminate this concern entirely.
2. **Better performance.** GC iterates all rooms — taking N locks is unnecessary overhead.
3. **Simpler mental model.** LastActivity is a single timestamp; atomic is the natural
   primitive for a single value read/written from multiple goroutines.

## Files to Change

1. **internal/domain/room.go**
   - Replace `LastActivity time.Time` with `lastActivity atomic.Int64`
   - Add `TouchActivity()`, `GetLastActivity()`, `LastActivityUnixNano()` methods
   - Update `NewRoom()` to call `TouchActivity()` instead of setting field
   - Update all methods that write `r.LastActivity = time.Now()` to call `r.TouchActivity()`

2. **internal/server/room_manager.go**
   - Update `collectGarbage()` line 60: use `room.GetLastActivity()` instead of `room.LastActivity`

3. **internal/server/room_manager.go (BuildRoomState)** — no change needed (doesn't use LastActivity)

4. **Tests:**
   - `internal/domain/room_test.go` — test `TouchActivity`/`GetLastActivity` methods
   - `internal/domain/room_race_test.go` — concurrent access test that passes `go test -race`
   - `internal/server/room_manager_test.go` — verify GC works correctly with atomic LastActivity

## Constraints

- Keep `CreatedAt time.Time` as-is (set once in constructor, never mutated)
- Do not change any public API signatures beyond replacing the field access pattern
- All existing tests must pass
- `go test -race ./...` must pass clean
