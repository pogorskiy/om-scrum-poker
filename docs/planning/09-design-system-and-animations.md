# Architecture: Design System Overhaul & Vote Reveal Animation

**Date:** 2026-04-05
**Issues:** [58], [42], [35], [60] from review/05-product.md

---

## 1. Overview

This document covers four related frontend improvements:

1. **[58] Color palette alignment** — Switch primary from Blue (#3b82f6) to Indigo (#6366f1) per UX spec
2. **[42] Extract hardcoded hex colors** — Replace all hardcoded hover colors with CSS custom properties
3. **[35] Dark theme support** — Add `prefers-color-scheme: dark` media query with dark token overrides
4. **[60] Card flip animation** — Add 3D flip animation when votes are revealed

---

## 2. Task [58] + [42]: Color Palette & Token Extraction

### Current State

`web/src/tokens.css` defines CSS custom properties for the design system. However:
- Primary color is `#3b82f6` (Blue) instead of `#6366f1` (Indigo) per UX spec
- Several files use hardcoded hex values instead of tokens:
  - `RoomPage.css:42` — `#d1d5db` (hover for secondary button)
  - `ConfirmDialog.css:53` — `#d1d5db` (hover for cancel button)
  - `ConfirmDialog.css:62` — `#dc2626` (hover for confirm/danger button)
  - `EditNameModal.css:62` — `#d1d5db` (hover for cancel button)
  - `ConnectionBanner.css` — Multiple hardcoded colors (`#fef3c7`, `#92400e`, `#fee2e2`, `#991b1b`, `#7f1d1d`)

### Changes Required

**tokens.css** — Update color values and add new tokens:

```css
:root {
  /* Primary: Indigo per UX spec (was Blue #3b82f6) */
  --color-primary: #6366f1;
  --color-primary-hover: #4f46e5;
  --color-card-selected: #e0e7ff;        /* Indigo-100 */
  --color-card-border-selected: #6366f1; /* Indigo-500 */

  /* New tokens for hover states */
  --color-border-hover: #d1d5db;
  --color-danger: #ef4444;
  --color-danger-hover: #dc2626;

  /* Connection banner tokens */
  --color-warning-bg: #fef3c7;
  --color-warning-text: #92400e;
  --color-danger-bg: #fee2e2;
  --color-danger-text: #991b1b;
  --color-danger-text-dark: #7f1d1d;
}
```

**Affected CSS files** — Replace hardcoded values with token references.

---

## 3. Task [35]: Dark Theme

### Approach

Add a `@media (prefers-color-scheme: dark)` block in `tokens.css` that overrides all color tokens.
This is the minimal-impact approach — all components already use CSS custom properties, so
only the token definitions need dark variants.

### Dark Theme Palette

```css
@media (prefers-color-scheme: dark) {
  :root {
    --color-bg: #0f0f0f;
    --color-surface: #1a1a1a;
    --color-text: #e5e5e5;
    --color-text-secondary: #a1a1aa;
    --color-primary: #818cf8;          /* Indigo-400 (lighter for dark bg) */
    --color-primary-hover: #6366f1;    /* Indigo-500 */
    --color-border: #2e2e2e;
    --color-border-hover: #404040;
    --color-status-active: #4ade80;
    --color-status-idle: #facc15;
    --color-status-disconnected: #f87171;
    --color-card-selected: #312e81;    /* Indigo-900 */
    --color-card-border-selected: #818cf8;
    --color-consensus: #052e16;        /* Green-950 */
    --color-overlay: rgba(0, 0, 0, 0.6);
    --color-danger: #f87171;
    --color-danger-hover: #ef4444;
    --color-danger-bg: #450a0a;
    --color-danger-text: #fca5a5;
    --color-danger-text-dark: #f87171;
    --color-warning-bg: #451a03;
    --color-warning-text: #fcd34d;

    --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.3);
    --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.4);
    --shadow-card: 0 2px 8px rgba(0, 0, 0, 0.5);
  }
}
```

### Key Decisions

- Use system preference (`prefers-color-scheme`) — no manual toggle needed for MVP
- Lighter primary in dark mode (`#818cf8` Indigo-400) for proper contrast
- Shadows become more opaque in dark mode for visibility
- All status colors shift to `-400` variants (lighter) for contrast on dark surfaces

---

## 4. Task [60]: Card Flip Animation

### Concept

When votes are revealed, each participant card performs a 3D card flip animation
(Y-axis rotation) to reveal the vote value. This creates the "reveal moment" that
makes planning poker engaging.

### Implementation

**CSS approach** — 3D transform with `rotateY(180deg)`:

```
Before reveal:  Front face (checkmark/empty) visible
During flip:    Card rotates 180° around Y axis (0.6s)
After reveal:   Back face (vote value) visible
```

**ParticipantCard changes:**

1. Add a `participant-card__vote-flipper` wrapper div with `transform-style: preserve-3d`
2. Two faces inside: `--front` (checkmark) and `--back` (vote value)
3. When `revealed && hasVoted`, add class `participant-card__vote-flipper--revealed`
4. Stagger animation with CSS `animation-delay` based on card index (via `--flip-delay` custom property)

**CSS Animation:**

```css
.participant-card__vote-flipper {
  position: relative;
  width: 32px;
  height: 32px;
  transform-style: preserve-3d;
  transition: transform 0.6s cubic-bezier(0.4, 0, 0.2, 1);
}

.participant-card__vote-flipper--revealed {
  transform: rotateY(180deg);
}

.participant-card__vote-face {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  backface-visibility: hidden;
}

.participant-card__vote-face--back {
  transform: rotateY(180deg);
}
```

**Staggering:** Pass participant index as a CSS custom property `style="--flip-delay: ${index * 0.08}s"`
and use `transition-delay: var(--flip-delay)` for cascading reveal effect.

**Accessibility:** Respect `prefers-reduced-motion`:

```css
@media (prefers-reduced-motion: reduce) {
  .participant-card__vote-flipper {
    transition: none;
  }
}
```

### Component Changes

**ParticipantCard.tsx** — Needs to accept an `index` prop for stagger delay.
**ParticipantList.tsx** — Pass index to each ParticipantCard.

---

## 5. Test Strategy

### Frontend Unit Tests

- **tokens.css** — Snapshot test verifying all CSS custom properties exist in both light and dark
- **ParticipantCard** — Test that:
  - Flip container renders with correct structure (front/back faces)
  - `--revealed` class is applied when `isRevealed` and participant has voted
  - `--flip-delay` CSS custom property is set from index prop
  - Vote value appears on back face, checkmark on front face
  - Non-voters don't get flip animation
- **Color token usage** — Grep-based test ensuring no hardcoded hex colors in CSS files (except tokens.css)

### Visual Regression (manual)

- Light theme: verify Indigo primary throughout
- Dark theme: verify all components render correctly with dark tokens
- Card flip: verify staggered animation on reveal

---

## 6. Files Modified

| File | Changes |
|------|---------|
| `web/src/tokens.css` | New tokens, Indigo palette, dark theme media query |
| `web/src/components/RoomPage/RoomPage.css` | Replace hardcoded `#d1d5db` |
| `web/src/components/ConfirmDialog/ConfirmDialog.css` | Replace hardcoded `#d1d5db`, `#dc2626` |
| `web/src/components/EditNameModal/EditNameModal.css` | Replace hardcoded `#d1d5db` |
| `web/src/components/ConnectionBanner/ConnectionBanner.css` | Replace hardcoded colors with tokens |
| `web/src/components/ParticipantCard/ParticipantCard.tsx` | Add flip animation structure, index prop |
| `web/src/components/ParticipantCard/ParticipantCard.css` | Add flip animation CSS |
| `web/src/components/ParticipantList/ParticipantList.tsx` | Pass index to ParticipantCard |
