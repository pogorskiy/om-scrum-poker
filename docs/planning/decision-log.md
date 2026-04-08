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

## 2026-04-08 — GC exclusive lock contention on RoomManager

**Problem:** collectGarbage() held exclusive Lock while iterating all rooms, blocking all operations
**Solution:** Two-phase GC: collect candidates under RLock, delete under Lock with re-verification
**Agents involved:** backend
**Key decisions:** Re-check conditions before delete to handle state changes between phases
**Tests added:** `TestCollectGarbage_RechecksBeforeDelete` in `internal/server/room_manager_test.go`
**Status:** ✅ Resolved
