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
| `MAX_CONNS_PER_IP` | `100`    | Max concurrent WebSocket connections per IP |
| `MAX_TOTAL_CONNS` | `1000`    | Max total concurrent WebSocket connections |

## UX Rules

- **Participant list must NOT be re-sorted by status.** Order is by join time (as returned by the server). Sorting by active/idle/disconnected causes names to jump around and confuses users. This is intentional — do not add status-based sorting.

## Conventions

- Go code: standard `gofmt` formatting, no linter config needed
- Frontend: TypeScript strict mode, PascalCase component dirs, BEM CSS naming
- Code comments in English, user-facing strings in English
- KISS principle: minimal dependencies, minimal abstractions

---

## Task Workflow

When given a problem to solve, follow this process strictly. Do not cut corners — thoroughness over speed.

### 1. Validate the Problem
- Read the relevant source code to confirm the problem still exists
- Check `docs/planning/decision-log.md` — maybe it was already resolved
- If the problem is not applicable or already fixed, **report back and move to the next problem**
- Do not start implementation until the problem is confirmed in code

### 2. Plan the Solution
- Analyze the root cause thoroughly
- Propose at least two approaches with explicit pros and cons
- Choose the best approach and document **why** before writing any code
- If the change is non-trivial, write a brief design note in `docs/planning/`

### 3. Branch Strategy
- Create a feature branch: `feat/<short-description>` or `fix/<short-description>`
- All agent work happens on this branch, never directly on `main`

### 4. Implement with Agent Teams
Spawn separate agents for backend and frontend work. They work in parallel.

**Backend Agent (Task):**
- Scope: `cmd/`, `internal/`, `web/embed.go`, `go.mod`
- Must write comprehensive unit tests for all new/changed code
- Must run `go test ./...` and confirm all tests pass
- Must follow Go stdlib conventions, `gofmt` formatting

**Frontend Agent (Task):**
- Scope: `web/src/`, `web/package.json`, `web/vite.config.ts`
- Must write comprehensive unit tests for all new/changed components
- Must run `cd web && npm test` and confirm all tests pass
- Must follow TypeScript strict mode, PascalCase components, BEM CSS

Each agent must produce a summary of changes when done.

### 5. Multi-Agent Review Process
This is mandatory. Do not skip any step.

**Step A — Cross-review between agents:**
- The backend agent reviews the frontend agent's changes for API contract consistency
- The frontend agent reviews the backend agent's changes for WebSocket protocol compatibility
- Each agent lists any concerns or inconsistencies found

**Step B — Lead Architect review (you, the orchestrator):**
- Review ALL changes holistically as a senior architect
- Check for:
  - Consistency between frontend and backend contracts
  - Edge cases and error handling
  - Test coverage completeness
  - No regressions in existing functionality
  - Adherence to UX Rules (e.g., no participant list re-sorting)
  - Code style and conventions compliance
- Run the full build: `make build`
- Run all tests: `go test ./...` and `cd web && npm test`

**Step C — Accept or reject:**
- If review passes → proceed to commit
- If issues found → revert the problematic changes, provide specific feedback, and **re-run the responsible agent with the review notes included**
- Repeat until the review passes

### 6. Commit
- Commit messages in English, conventional commits format
- Format: `type(scope): short description`
- Examples:
  - `fix(ws): handle timer state loss on reconnect`
  - `feat(web): add vote retraction button`
  - `test(server): add room GC edge case coverage`
  - `docs(planning): add ADR for timer sync approach`
- Make atomic commits — one logical change per commit
- Do NOT bundle unrelated changes in a single commit

### 7. Architecture Artifacts
After implementation, create or update documentation so that **future agents can work effectively**:

- **Architecture Decision Records** in `docs/planning/`:
  - What was decided and why
  - What alternatives were considered
  - What trade-offs were accepted
- **Inline code comments** for any non-obvious logic, concurrency patterns, or protocol details
- **Update this CLAUDE.md** if the change affects:
  - Project structure
  - WebSocket protocol (add new events)
  - Environment variables
  - Build commands
  - UX rules or conventions

### 8. Update Decision Log
Append to `docs/planning/decision-log.md`:

```markdown
## [DATE] — Short title of the problem

**Problem:** One-line description of the issue
**Solution:** What was implemented
**Agents involved:** backend / frontend / both
**Key decisions:** Any non-obvious choices made
**Tests added:** List of new test files or test functions
**Status:** ✅ Resolved
```

This log prevents re-investigating already-solved problems.

---

## Agent Roles Reference

| Role | Scope | Tests | Review responsibility |
|------|-------|-------|-----------------------|
| Backend Agent | `cmd/`, `internal/`, `*.go` | `go test ./...` | Reviews frontend for API consistency |
| Frontend Agent | `web/src/`, `web/*.ts` | `npm test` | Reviews backend for WS protocol consistency |
| Lead Architect (orchestrator) | Everything | Full build + all tests | Final approval, revert authority |