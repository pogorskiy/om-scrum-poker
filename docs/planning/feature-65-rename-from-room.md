# Feature [65]: In-Room Name Change

## Problem
Users cannot change their display name from within a room. The only way is to clear localStorage manually. The backend already supports `update_name`, and the frontend handles `name_updated` events, but there is no UI trigger.

## Architecture Decision

### Backend Status: COMPLETE
The backend already fully supports name changes:
- `ws.go:321-355` — `handleUpdateName` parses payload, calls `room.UpdateName`, broadcasts `name_updated`
- `domain/room.go:184-198` — `UpdateName` validates name (non-empty, max 30 chars), updates participant
- `events.go:27-29` — `UpdateNamePayload` struct
- `events.go:75-78` — `NameUpdatedPayload` struct
- `state.ts:58` — `ClientMessage` already includes `update_name` type
- `ws.ts:209-219` — Frontend already handles incoming `name_updated` events

**Backend work:** Add comprehensive unit tests for the `handleUpdateName` dispatch path.

### Frontend Changes Required

#### 1. Header Component (`web/src/components/Header/Header.tsx`)
- Display current user's name in the header (right side, before "Copy Link")
- Add a pencil/edit icon button next to the name
- Clicking the edit button opens the `EditNameModal`

#### 2. New Component: `EditNameModal` (`web/src/components/EditNameModal/`)
- Reuse modal styling patterns from `NameEntryModal` and `ConfirmDialog`
- Pre-fill input with current name from `userName` signal
- On submit:
  1. Call `setUserName(newName)` to update localStorage and signal
  2. Call `send({ type: 'update_name', payload: { userName: newName } })` to notify server
- Validate: non-empty, max 30 chars, trimmed
- Cancel button closes without changes

#### 3. State Integration
- Import `userName` signal (already exists in `state.ts`)
- Import `send` from `ws.ts` (already exists)
- Import `setUserName` from `state.ts` (already exists)
- No new signals needed

### Wire Protocol (existing, no changes)
```
Client → Server: { "type": "update_name", "payload": { "userName": "New Name" } }
Server → All:    { "type": "name_updated", "payload": { "sessionId": "...", "userName": "New Name" } }
```

### CSS Conventions
- BEM naming: `edit-name-modal`, `edit-name-modal__input`, etc.
- Use existing CSS variables: `--color-*`, `--space-*`, `--radius-*`, `--shadow-*`
- Modal overlay z-index: 850 (between NameEntryModal:800 and ConfirmDialog:900)

### File Map
```
EXISTING (no changes):
  internal/server/ws.go          — handleUpdateName (lines 321-355)
  internal/domain/room.go        — UpdateName (lines 184-198)
  internal/server/events.go      — UpdateNamePayload, NameUpdatedPayload
  web/src/state.ts               — userName signal, setUserName, ClientMessage type
  web/src/ws.ts                  — name_updated handler (lines 209-219), send()

MODIFY:
  web/src/components/Header/Header.tsx  — add user name display + edit button
  web/src/components/Header/Header.css  — styles for name display + edit button

CREATE:
  web/src/components/EditNameModal/EditNameModal.tsx  — modal component
  web/src/components/EditNameModal/EditNameModal.css  — modal styles

TEST (create/extend):
  internal/server/ws_test.go     — tests for handleUpdateName dispatch path
  internal/domain/room_test.go   — verify UpdateName tests exist (they do)
```
