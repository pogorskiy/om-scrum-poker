# Issue [55] — WebSocket Origin Validation

## Problem

`internal/server/ws.go:33-35` uses `InsecureSkipVerify: true` in `websocket.AcceptOptions`,
which completely disables Origin header validation. Any website can open a WebSocket
connection to the server, enabling Cross-Site WebSocket Hijacking.

## Solution Design

### Backend Changes (`internal/server/`)

1. **Add `AllowedOrigins` to `Config`** (`handler.go`):
   - New field: `AllowedOrigins []string`
   - Parsed from `ALLOWED_ORIGINS` environment variable (comma-separated)
   - Special value `"*"` means allow all origins (backward compat / dev mode)
   - Empty means use default same-origin check (Origin host == Host header)

2. **Pass `AllowedOrigins` to `HandleWebSocket`** (`handler.go`, `ws.go`):
   - Update `HandleWebSocket` signature to accept `allowedOrigins []string`
   - Update `NewServer` to pass `config.AllowedOrigins`

3. **Configure `websocket.AcceptOptions`** (`ws.go`):
   - If `allowedOrigins` contains `"*"`: use `InsecureSkipVerify: true`
   - If `allowedOrigins` is non-empty: use `OriginPatterns` field
   - If `allowedOrigins` is empty: omit both fields (library enforces same-origin by default)

   The `nhooyr.io/websocket` library's default behavior (when neither `InsecureSkipVerify`
   nor `OriginPatterns` is set) checks that the Origin header's host matches the Host header.
   This is exactly the secure default we want.

4. **Update `main.go`**:
   - Parse `ALLOWED_ORIGINS` env var, split by comma, trim whitespace
   - Pass to `Config.AllowedOrigins`

### Frontend Changes (`web/`)

None required. The browser automatically sends the `Origin` header on WebSocket connections.
The `ws.ts` file uses `window.location.host` to build the WebSocket URL, which means
in production the Origin will match the Host (same-origin).

**Development note**: When using Vite dev server (`:5173`) proxying to Go (`:8080`),
the `http-proxy` forwards the browser's `Origin: http://localhost:5173` to the backend.
Since Host will be `localhost:8080`, same-origin check would fail. Developers should set
`ALLOWED_ORIGINS=http://localhost:5173` when running the Go server in dev mode, or use
`ALLOWED_ORIGINS=*` for convenience.

### Environment Variables Update

| Variable          | Default | Description                                    |
|-------------------|---------|------------------------------------------------|
| `ALLOWED_ORIGINS` | (empty) | Comma-separated allowed origins. `*` = allow all. Empty = same-origin only. |

### Test Plan

1. **Unit tests** (`ws_test.go` or new `origin_test.go`):
   - Test `buildAcceptOptions` with empty origins (same-origin default)
   - Test `buildAcceptOptions` with `"*"` (insecure skip)
   - Test `buildAcceptOptions` with specific origins list
   - Test `parseAllowedOrigins` helper function

2. **Integration-style tests** (`handler_test.go`):
   - HTTP test with valid Origin header -> upgrade succeeds
   - HTTP test with invalid Origin header -> upgrade rejected (403)
   - Test with `ALLOWED_ORIGINS=*` -> any origin accepted

### Files to Modify

- `cmd/server/main.go` — parse env var
- `internal/server/handler.go` — Config struct, pass origins
- `internal/server/ws.go` — accept options logic
- `internal/server/ws_test.go` — new origin validation tests
- `CLAUDE.md` — document new env var
