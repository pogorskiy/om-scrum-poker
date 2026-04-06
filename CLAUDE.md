# om-scrum-poker

Lightweight real-time scrum poker service. Single Go binary with embedded Preact frontend.

## Build Commands

```bash
# Full production build (frontend + backend)
make build

# Frontend only
cd web && npm run build

# Backend only (requires web/dist to exist)
go build ./cmd/server

# Run tests
go test ./...

# Docker
docker build -t om-scrum-poker .
docker run -p 8080:8080 om-scrum-poker
```

## Development

```bash
# Backend (serves on :8080, uses web/dist from disk if available)
ALLOWED_ORIGINS=http://localhost:5173 go run ./cmd/server

# Frontend dev server (Vite on :5173, proxies API/WS to :8080)
cd web && npm run dev
```

## Project Structure

```
cmd/server/         Entry point (main.go)
internal/server/    HTTP handlers, WebSocket, room manager, rate limiter
web/                Preact frontend (TypeScript + Vite)
web/embed.go        go:embed directive for web/dist
web/dist/           Built frontend assets (gitignored, kept via .gitkeep)
docs/planning/      Architecture and planning documents
```

## Architecture

- **Backend:** Go stdlib + nhooyr.io/websocket. Two packages: `server` (all logic) and `main` (entry point).
- **Frontend:** Preact + @preact/signals + TypeScript. Vite bundler. No router library.
- **Static embedding:** `web/embed.go` embeds `web/dist/` via `//go:embed`. The server tries embedded FS first, then disk-based `web/dist/`, then shows a placeholder.
- **State:** All room state is in-memory. No database. Rooms are garbage-collected after inactivity.
- **Auth:** None. Display name stored in localStorage.

## WebSocket Protocol

Endpoint: `GET /ws/{roomId}`

All messages use envelope format: `{ "type": "<event>", "payload": { ... } }`.

Key events (client -> server): `join`, `vote`, `reveal`, `new_round`, `clear_room`, `update_name`, `presence`, `update_role`, `leave`, `timer_set_duration`, `timer_start`, `timer_reset`.
Key events (server -> client): `room_state`, `participant_joined`, `participant_left`, `vote_cast`, `vote_retracted`, `votes_revealed`, `round_reset`, `room_cleared`, `presence_changed`, `name_updated`, `role_updated`, `timer_updated`, `error`.

Full protocol spec: [docs/planning/04-architecture.md](docs/planning/04-architecture.md) Section 4.

## Environment Variables

| Variable      | Default   | Description                      |
|---------------|-----------|----------------------------------|
| `HOST`        | `0.0.0.0` | Listen address                   |
| `PORT`        | `8080`    | Listen port                      |
| `TRUST_PROXY` | `false`   | Trust X-Forwarded-For for rate limiting |
| `ALLOWED_ORIGINS` | (empty)   | Comma-separated allowed WebSocket origins. `*` = allow all. Empty = same-origin only. |

## Conventions

- Go code: standard `gofmt` formatting, no linter config needed
- Frontend: TypeScript strict mode, PascalCase component dirs, BEM CSS naming
- Code comments in English, user-facing strings in English
- KISS principle: minimal dependencies, minimal abstractions
