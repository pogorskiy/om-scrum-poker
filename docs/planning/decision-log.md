# Decision Log

## 2026-04-08 — Zalgo text and emoji truncation in usernames

**Problem:** Usernames with zalgo text (excessive combining marks) rendered incorrectly and broke layout; name truncation at 30 bytes could split multi-byte UTF-8 characters (emoji, Cyrillic)
**Solution:** Limit combining marks (Mn, Me) to 3 per base character in `sanitizeName()`; switch truncation from `name[:30]` (bytes) to `[]rune(name)[:30]` (characters)
**Agents involved:** backend
**Key decisions:** Chose to allow up to 3 combining marks (not strip all) to preserve normal diacritics like é, ñ for international names
**Tests added:** `TestUpdateName_EmojiTruncation`, `TestSanitizeName_ZalgoText` in `internal/domain/room_test.go`
**Status:** ✅ Resolved
