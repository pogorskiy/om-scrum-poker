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

## 2026-04-08 — Race condition on disconnect: broadcast after UnregisterClient

**Problem:** UnregisterClient removed client before setting status to "disconnected", allowing stale "active" in room_state
**Solution:** Reorder: set status + broadcast before UnregisterClient
**Agents involved:** backend
**Key decisions:** Simple reorder preferred over new atomic method — KISS principle
**Tests added:** None new — existing TestPresenceLifecycle covers the flow, reorder is a logic-ordering fix
**Status:** ✅ Resolved
