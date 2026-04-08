# Decision Log

## 2026-04-08 — Zalgo text and emoji truncation in usernames

**Problem:** Usernames with zalgo text (excessive combining marks) rendered incorrectly and broke layout; name truncation at 30 bytes could split multi-byte UTF-8 characters (emoji, Cyrillic)
**Solution:** Limit combining marks (Mn, Me) to 3 per base character in `sanitizeName()`; switch truncation from `name[:30]` (bytes) to `[]rune(name)[:30]` (characters)
**Agents involved:** backend
**Key decisions:** Chose to allow up to 3 combining marks (not strip all) to preserve normal diacritics like é, ñ for international names
**Tests added:** `TestUpdateName_EmojiTruncation`, `TestSanitizeName_ZalgoText` in `internal/domain/room_test.go`
**Status:** ✅ Resolved

## 2026-04-08 — Remove "by creator" from room header

**Problem:** Room header showed "by {creatorName}" next to room name — not informative, clutters UI
**Solution:** Removed `createdBy` rendering from Header component and associated CSS
**Agents involved:** frontend
**Key decisions:** Full removal of JSX and CSS rather than hiding via CSS, since the data adds no user value
**Tests added:** None needed — removed UI-only element, existing tests pass
**Status:** ✅ Resolved

## 2026-04-08 — Document timer and role WebSocket events

**Problem:** 6 WebSocket events (timer, role) and several payload fields (createdBy, timer, role) missing from spec that claims to be exhaustive
**Solution:** Added all 6 events to canonical table 4.3, full payload examples in 4.4/4.5, updated room_state and participant_joined payloads, updated CLAUDE.md
**Agents involved:** documentation only
**Key decisions:** Also documented the client-side timer expiration caveat (server doesn't track expiration, clock skew possible)
**Tests added:** None — documentation-only change
**Status:** ✅ Resolved

## 2026-04-08 — Unbounded message queue on WebSocket disconnect

**Problem:** messageQueue in ws.ts grew without limits during disconnect, and flushed stale messages in bulk on reconnect risking rate limit violation
**Solution:** Cap queue at 20 messages, clear queue on reconnect (room_state restores state), remove flushQueue function
**Agents involved:** frontend
**Key decisions:** Chose to discard all queued messages rather than selective filtering — room_state from server is authoritative
**Tests added:** `message queue > discards queued messages on reconnect`, `message queue > caps queue size to prevent memory leaks` in `web/src/ws.test.ts`
**Status:** ✅ Resolved

## 2026-04-08 — GC exclusive lock contention on RoomManager

**Problem:** collectGarbage() held exclusive Lock while iterating all rooms, blocking all operations
**Solution:** Two-phase GC: collect candidates under RLock, delete under Lock with re-verification
**Agents involved:** backend
**Key decisions:** Re-check conditions before delete to handle state changes between phases
**Tests added:** `TestCollectGarbage_RechecksBeforeDelete` in `internal/server/room_manager_test.go`
**Status:** ✅ Resolved

## 2026-04-08 — Race condition on disconnect: broadcast after UnregisterClient

**Problem:** UnregisterClient removed client before setting status to "disconnected", allowing stale "active" in room_state
**Solution:** Reorder: set status + broadcast before UnregisterClient
**Agents involved:** backend
**Key decisions:** Simple reorder preferred over new atomic method — KISS principle
**Tests added:** None new — existing TestPresenceLifecycle covers the flow, reorder is a logic-ordering fix
**Status:** ✅ Resolved

## 2026-04-08 — Missing HTTP security headers

**Problem:** No X-Content-Type-Options, X-Frame-Options, Referrer-Policy on HTTP responses
**Solution:** Added securityHeaders middleware wrapping mux in NewServer
**Agents involved:** backend
**Key decisions:** CORS headers not added — SPA is same-origin, WS already checks origin separately
**Tests added:** `TestSecurityHeaders` in `internal/server/handler_test.go`
**Status:** ✅ Resolved
