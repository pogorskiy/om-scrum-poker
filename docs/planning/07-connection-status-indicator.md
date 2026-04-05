# Architecture: Connection Status Indicator [#70]

## Problem

When the WebSocket connection is lost, the user has no persistent visual feedback:
- Toast "Connection lost. Click to retry." appears after 30s timeout, disappears in 2.3s
- Toast is not clickable (`retry()` exported but unused in UI)
- User can attempt to vote while disconnected (votes queue silently)
- No indication during reconnection attempts

## Solution Overview

### Frontend Changes

#### 1. New signal: `reconnectInfo` in `state.ts`

```typescript
export interface ReconnectInfo {
  attempt: number;
  maxReached: boolean; // true when RECONNECT_TIMEOUT exceeded
}

export const reconnectInfo = signal<ReconnectInfo>({ attempt: 0, maxReached: false });
```

#### 2. Updated `ws.ts` reconnect logic

- Remove the 30s hard timeout that stops reconnection
- Instead, after RECONNECT_TIMEOUT (30s), switch to slow polling (every 10s) indefinitely
- Update `reconnectInfo` signal on each attempt
- Keep `retry()` export — it resets attempt counter and reconnects immediately

**Key changes:**
- `scheduleReconnect()`: instead of giving up after 30s, continue with longer intervals
- On each reconnect attempt, update `reconnectInfo.value = { attempt: N, maxReached: bool }`
- On successful connect, reset `reconnectInfo.value = { attempt: 0, maxReached: false }`
- Remove the toast "Connection lost. Click to retry." from `scheduleReconnect()` — the banner handles this now

#### 3. New component: `ConnectionBanner`

Location: `web/src/components/ConnectionBanner/ConnectionBanner.tsx`

**States:**
| connectionStatus | reconnectInfo.maxReached | Display |
|-----------------|------------------------|---------|
| `connected` | — | Hidden |
| `connecting` | `false` | "Reconnecting..." + spinner + attempt count |
| `disconnected` | `false` | "Reconnecting..." + spinner + attempt count |
| `disconnected` | `true` | "Connection lost" + Retry button |

**Design:**
- Full-width banner below header, above content
- Yellow/amber background for reconnecting state
- Red background for connection lost state
- Retry button calls `retry()` from `ws.ts`
- CSS animation: slide-down on appear, slide-up on disappear
- BEM naming: `.connection-banner`, `.connection-banner--reconnecting`, `.connection-banner--lost`

#### 4. Integration in `RoomPage.tsx`

- Import `ConnectionBanner`
- Place it after `<Header />` and before `<ParticipantList />`
- No changes to existing connection/disconnection logic

#### 5. Disable voting UI when disconnected

- In `CardDeck`, check `connectionStatus` — if not `connected`, add visual dimming and `pointer-events: none`
- This prevents silent vote queuing that confuses users

### Backend Changes

#### 1. Health check endpoint

Add `GET /api/health` returning `{"status":"ok","rooms":<count>,"clients":<count>}`.

Useful for monitoring and load balancer health checks. Minimal implementation in `handler.go`.

#### 2. Enhanced test coverage for disconnect scenarios

Add tests in `ws_test.go` verifying:
- Client disconnect marks participant as `disconnected`
- Reconnecting client (same sessionID) restores `active` status
- Ping timeout triggers proper cleanup

## File Change Summary

| File | Change |
|------|--------|
| `web/src/state.ts` | Add `ReconnectInfo` interface, `reconnectInfo` signal |
| `web/src/ws.ts` | Remove hard timeout, add slow polling, update reconnectInfo signal |
| `web/src/components/ConnectionBanner/ConnectionBanner.tsx` | New component |
| `web/src/components/ConnectionBanner/ConnectionBanner.css` | New styles |
| `web/src/components/RoomPage/RoomPage.tsx` | Import and render ConnectionBanner |
| `web/src/components/CardDeck/CardDeck.tsx` | Dim when disconnected |
| `web/src/components/CardDeck/CardDeck.css` | Add disabled state styles |
| `internal/server/handler.go` | Add `/api/health` endpoint |
| `internal/server/handler_test.go` | Test health endpoint |
| `internal/server/ws_test.go` | Add disconnect/reconnect scenario tests |

## Testing Strategy

### Frontend unit tests (vitest + @testing-library/preact)
- `ConnectionBanner.test.tsx`: renders correct state for each connectionStatus/reconnectInfo combination
- `ws.test.ts`: reconnect logic — verifies attempt counting, slow polling after timeout, retry reset

### Backend unit tests (Go testing)
- Health endpoint returns correct JSON
- Disconnect/reconnect lifecycle tests
