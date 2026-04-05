# Issue #72: Modal Focus Management & Scroll Blocking

## Problem Statement

All three modal components (NameEntryModal, EditNameModal, ConfirmDialog) lack:
- Focus trapping (Tab key escapes modal)
- Background scroll blocking
- ARIA attributes (`role="dialog"`, `aria-modal`, `aria-labelledby`)
- ConfirmDialog: no Escape key handler

This violates WCAG 2.1 AA (2.4.3 Focus Order, 4.1.2 Name/Role/Value).

## Solution: Native `<dialog>` Element

Use the native HTML `<dialog>` element with `.showModal()` API. This provides **for free**:

| Feature | Custom div overlay | Native `<dialog>.showModal()` |
|---|---|---|
| Focus trap | Manual implementation | Built-in |
| Scroll blocking | Manual `overflow:hidden` on body | Built-in |
| Escape to close | Manual keydown listener | Built-in (fires `cancel` event) |
| Backdrop | Custom overlay div + z-index | `::backdrop` pseudo-element |
| `role="dialog"` | Must add manually | Implicit |
| `aria-modal="true"` | Must add manually | Implicit |
| Browser support | N/A | All modern browsers (97%+ globally) |

### Why native `<dialog>` over a focus-trap library?
- Zero dependencies (KISS principle)
- Browser handles edge cases (shadow DOM, iframes)
- Correct top-layer stacking (no z-index wars)
- Less code to maintain

## Architecture

### New Shared Component: `Modal`

Location: `web/src/components/Modal/Modal.tsx`

```tsx
interface ModalProps {
  open: boolean;
  onClose?: () => void;       // called on Escape / backdrop click
  dismissable?: boolean;       // default true; false = no Escape, no backdrop click
  ariaLabelledBy?: string;
  ariaDescribedBy?: string;
  children: ComponentChildren;
  class?: string;
}
```

**Responsibilities:**
1. Renders a `<dialog>` element
2. Calls `.showModal()` when `open` becomes true, `.close()` when false
3. Handles `cancel` event (Escape key) — calls `onClose` if dismissable
4. Handles backdrop click via `::backdrop` click detection
5. Restores focus to previously focused element on close
6. Passes through `aria-labelledby` and `aria-describedby`

### Refactored Components

#### NameEntryModal
- Wraps content in `<Modal open={true} dismissable={false}>`
- Non-dismissable (user MUST enter name)
- `aria-labelledby` points to label "What should we call you?"
- Keeps `autoFocus` on input

#### EditNameModal
- Wraps content in `<Modal open={true} onClose={onClose}>`
- Dismissable via Escape and backdrop click
- Removes manual keydown listener (dialog handles Escape)
- Removes manual overlay div
- `aria-labelledby` points to "Change your name" label

#### ConfirmDialog
- Wraps content in `<Modal open={true} onClose={onCancel}>`
- Dismissable via Escape and backdrop click (both trigger cancel)
- Removes manual overlay div
- `aria-labelledby` points to title, `aria-describedby` points to message
- Auto-focuses Cancel button (safe default for destructive confirm)

### CSS Changes

- Remove `.confirm-overlay`, `.edit-name-modal__overlay`, `.name-modal-overlay` classes
- Add `Modal.css` with `dialog::backdrop` styling using `var(--color-overlay)`
- Keep existing dialog content styles (`.confirm-dialog`, `.edit-name-modal`, `.name-modal`)
- Add animation via `dialog[open]` and `::backdrop` selectors

### File Changes Summary

| File | Action |
|---|---|
| `web/src/components/Modal/Modal.tsx` | **NEW** — shared dialog wrapper |
| `web/src/components/Modal/Modal.css` | **NEW** — dialog & backdrop styles |
| `web/src/components/ConfirmDialog/ConfirmDialog.tsx` | MODIFY — use Modal, add aria attrs |
| `web/src/components/ConfirmDialog/ConfirmDialog.css` | MODIFY — remove overlay styles |
| `web/src/components/EditNameModal/EditNameModal.tsx` | MODIFY — use Modal, remove manual Escape handler |
| `web/src/components/EditNameModal/EditNameModal.css` | MODIFY — remove overlay styles |
| `web/src/components/NameEntryModal/NameEntryModal.tsx` | MODIFY — use Modal |
| `web/src/components/NameEntryModal/NameEntryModal.css` | MODIFY — remove overlay styles |
| `web/src/components/Modal/Modal.test.tsx` | **NEW** — comprehensive tests |
| `web/src/components/ConfirmDialog/ConfirmDialog.test.tsx` | **NEW** — tests |
| `web/src/components/EditNameModal/EditNameModal.test.tsx` | **NEW** — tests |
| `web/src/components/NameEntryModal/NameEntryModal.test.tsx` | **NEW** — tests |

## Testing Strategy

### Modal (shared component)
- Opens dialog when `open=true`, closes when `open=false`
- Calls `onClose` on Escape key when dismissable
- Does NOT call `onClose` on Escape when `dismissable=false`
- Calls `onClose` on backdrop click when dismissable
- Sets `aria-labelledby` and `aria-describedby`
- Restores focus to previously focused element on close

### ConfirmDialog
- Renders title and message
- Calls `onConfirm` on Confirm click
- Calls `onCancel` on Cancel click
- Calls `onCancel` on Escape
- Has correct aria-labelledby/describedby

### EditNameModal
- Pre-fills current username
- Auto-focuses input
- Submits on form submit, calls onClose
- Closes on Escape
- Closes on backdrop click

### NameEntryModal
- Cannot be dismissed (no Escape, no backdrop close)
- Submits name on form submit
- Auto-focuses input
- Button disabled when input empty

## Important Notes for Implementation

- `jsdom` does NOT implement `HTMLDialogElement.showModal()` — tests must mock it
- Use `useRef` + `useEffect` to manage dialog open/close lifecycle
- The `cancel` event fires on Escape — prevent default and call `onClose` for controlled behavior
- Backdrop click detection: check if click target is the dialog element itself (not children)
