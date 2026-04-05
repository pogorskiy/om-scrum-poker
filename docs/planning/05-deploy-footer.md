# Feature: Deploy Timestamp Footer

## Overview

Add a footer to the frontend displaying the build/deploy timestamp in UTC, acting as a lightweight version indicator. The footer must be unobtrusive, present on all pages, and consistent with the existing design system.

## Architecture

### Build-Time Timestamp Injection

The deploy timestamp is injected at **build time** through two parallel mechanisms:

1. **Frontend (Vite `define`):** Vite injects `__BUILD_TIMESTAMP__` as a compile-time constant into the JS bundle via `vite.config.ts`. This is the value displayed in the footer.

2. **Backend (Go `ldflags`):** The Makefile passes `-ldflags "-X main.buildTime=..."` during `go build`. This value is exposed via the existing `/health` endpoint for operational monitoring.

Both mechanisms use the same UTC ISO 8601 format: `2024-01-15T14:30:00Z`.

In development mode, the value defaults to `"dev"` (Go) or the current dev-server start time (Vite).

### Frontend Component: Footer

**Location:** `web/src/components/Footer/Footer.tsx` + `Footer.css`

**Design:**
- Renders at the bottom of the page (after all content, pushed down by flexbox)
- Text: `Deployed: Jan 15, 2024, 14:30 UTC` (human-readable format)
- Style: small font (11px), muted color (`var(--color-text)` at 40% opacity), centered
- Semantic HTML: `<footer>` element
- No interactive elements, purely informational
- `<time>` element with `datetime` attribute for machine-readability

**Integration:** Added to `app.tsx` after the route content, outside page components, so it appears on both HomePage and RoomPage.

**CSS approach:**
- Uses existing design tokens (spacing, colors, fonts)
- `margin-top: auto` on the footer to push it to the bottom (parent `#app` is already `display: flex; flex-direction: column; min-height: 100vh`)
- BEM naming: `.footer`, `.footer__text`
- Responsive: same on all viewports

### Backend Changes

**cmd/server/main.go:**
- Add package-level `var buildTime string = "dev"`
- Pass `buildTime` to server config

**internal/server/handler.go:**
- Add `BuildTime string` field to `Config` struct
- Add `build_time` field to `HealthResponse`
- Include in `/health` JSON output

**Makefile:**
- Update `build-backend` target to pass `-ldflags "-X main.buildTime=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')"`

### File Changes Summary

| File | Change |
|------|--------|
| `web/vite.config.ts` | Add `define: { __BUILD_TIMESTAMP__: ... }` |
| `web/src/components/Footer/Footer.tsx` | New component |
| `web/src/components/Footer/Footer.css` | New styles |
| `web/src/components/Footer/Footer.test.tsx` | New tests |
| `web/src/app.tsx` | Import and render `<Footer />` |
| `web/src/vite-env.d.ts` | Declare `__BUILD_TIMESTAMP__` global |
| `cmd/server/main.go` | Add `buildTime` var, pass to config |
| `internal/server/handler.go` | Add `BuildTime` to Config and HealthResponse |
| `internal/server/handler_test.go` | Test build_time in /health response |
| `Makefile` | Add ldflags to build-backend |

### Design Mockup (ASCII)

```
┌─────────────────────────────────────────┐
│  om          Room Name        User ✏ 📋 │  ← Header
├─────────────────────────────────────────┤
│                                         │
│          [Page Content]                 │
│                                         │
│                                         │
├─────────────────────────────────────────┤
│       Deployed: Jan 15, 2024, 14:30 UTC │  ← Footer (muted, small)
└─────────────────────────────────────────┘
```

### Test Plan

**Frontend tests (Footer.test.tsx):**
1. Renders footer element with correct semantic HTML (`<footer>`, `<time>`)
2. Displays formatted timestamp from `__BUILD_TIMESTAMP__`
3. Shows "Development build" when timestamp is not a valid ISO date
4. `<time>` element has correct `datetime` attribute
5. Footer has correct CSS class names

**Backend tests (handler_test.go):**
1. `/health` endpoint includes `build_time` field
2. `build_time` reflects configured value
3. `build_time` defaults to "dev" when not set

### Global Type Declaration

```typescript
// web/src/vite-env.d.ts
declare const __BUILD_TIMESTAMP__: string;
```
