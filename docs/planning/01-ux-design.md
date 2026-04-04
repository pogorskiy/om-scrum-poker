# om-scrum-poker: UX Design & UI Requirements

**Version:** 1.0
**Date:** 2026-04-04
**Status:** Draft

---

## Table of Contents

1. [Design Principles](#1-design-principles)
2. [User Flow Diagrams](#2-user-flow-diagrams)
3. [Screen-by-Screen UX Specification](#3-screen-by-screen-ux-specification)
4. [UI Component Requirements](#4-ui-component-requirements)
5. [Card Design](#5-card-design)
6. [Presence Indicators](#6-presence-indicators)
7. [Action Buttons](#7-action-buttons)
8. [Responsive Design Notes](#8-responsive-design-notes)
9. [Edge Cases & Micro-interactions](#9-edge-cases--micro-interactions)
10. [Scrum Poker Best Practices Review](#10-scrum-poker-best-practices-review)

---

## 1. Design Principles

These principles guide every decision in the document. When in doubt, refer back here.

- **Zero friction.** A user should go from link click to voting in under 5 seconds (name entry excluded on first visit).
- **No accounts, no auth.** Identity is a name stored in `localStorage`. That is it.
- **Clarity over cleverness.** Every state must be instantly readable. No ambiguity about who voted, what was voted, or what phase we are in.
- **Speed of ceremony.** Planning poker already feels slow to engineers. The tool must never be the bottleneck. Clicks, not workflows.
- **Mobile-first layout.** Half the team will be on phones during standup or remote calls.

---

## 2. User Flow Diagrams

### 2.1 Room Creation Flow

```
[Homepage]
    |
    v
User enters room name (free text input)
    |
    v
User clicks "Create Room"
    |
    v
System generates URL: /room/{slug}-{partial-guid}
  - slug = kebab-case of room name (e.g. "Sprint 42" -> "sprint-42")
  - partial-guid = first 8 chars of a UUID v4 (e.g. "a3f1c9b2")
  - example: /room/sprint-42-a3f1c9b2
    |
    v
System checks localStorage for "userName"
    |
    +-- Name found --> Join room directly, land on Room Page
    |
    +-- Name NOT found --> Show Name Entry Modal --> then join room
    |
    v
[Room Page] -- user sees empty room, can copy link to share
```

### 2.2 Joining via Shared Link

```
[User clicks shared link]
    |
    v
System loads /room/{slug}-{partial-guid}
    |
    v
System checks localStorage for "userName"
    |
    +-- Name found --> Join room via WebSocket, land on Room Page
    |
    +-- Name NOT found --> Show Name Entry Modal
    |                           |
    |                           v
    |                      User enters name, clicks "Join"
    |                           |
    |                           v
    |                      Name saved to localStorage
    |                           |
    +---------------------------+
    |
    v
[Room Page] -- user sees other participants, current vote state
```

### 2.3 Voting Flow

```
[Room Page - Voting Phase]
    |
    v
Each participant sees card deck at bottom of screen
    |
    v
User taps a card --> card highlights as selected
    |                  Other participants see "voted" indicator (face-down card)
    |                  User can change selection freely (tap different card)
    |
    v
Any participant clicks "Show Votes"
    |
    v
[Room Page - Reveal Phase]
    |
    v
All cards flip face-up simultaneously
Statistics displayed: Average, Median, and agreement indicator
    |
    v
Discussion happens (outside the tool)
    |
    v
Any participant clicks "New Round" (was "Delete Estimates")
    |
    v
All votes cleared, deck re-enabled --> back to Voting Phase
```

### 2.4 Clear Room Flow

```
[Room Page]
    |
    v
Any participant clicks "Clear Room"
    |
    v
Confirmation dialog: "Remove all participants? This cannot be undone."
    |
    +-- Cancel --> return to Room Page (no change)
    |
    +-- Confirm --> All users removed, room reset to empty state
                    Each connected client sees a "Room was cleared" notice
                    and is returned to the Name Entry state
```

---

## 3. Screen-by-Screen UX Specification

### 3.1 Home Page (Room Creation)

**Purpose:** Single action -- create a room.

**Layout:**

```
+--------------------------------------------------+
|  [om logo/wordmark]           om-scrum-poker      |
+--------------------------------------------------+
|                                                    |
|              [om-scrum-poker]                      |
|          Simple. Self-hosted. No signup.            |
|                                                    |
|     +------------------------------------+         |
|     |  Room name                         |         |
|     +------------------------------------+         |
|     [        Create Room                 ]         |
|                                                    |
|                                                    |
+--------------------------------------------------+
```

**Specifications:**

| Element | Detail |
|---|---|
| Page title | "om-scrum-poker" |
| Tagline | "Simple. Self-hosted. No signup." |
| Input field | Placeholder: "e.g. Sprint 42 Planning". Max 60 characters. Auto-focused on page load. |
| Create button | Primary action. Disabled until input is non-empty (after trimming whitespace). |
| Submit on Enter | Yes -- pressing Enter in the input field triggers room creation. |
| Validation | Alphanumeric, spaces, hyphens, underscores only. Inline error for invalid characters. |
| Empty state | No rooms list, no history. Pure single-purpose page. |

### 3.2 Name Entry Modal

**Purpose:** Capture participant display name. Shown only if `localStorage` has no stored name.

**Presentation:** Overlay modal with backdrop blur. Cannot be dismissed without entering a name (no close button, no clicking outside to dismiss).

**Layout:**

```
+------------------------------------------+
|                                            |
|          What should we call you?          |
|                                            |
|     +--------------------------------+     |
|     |  Your name                     |     |
|     +--------------------------------+     |
|     [           Join               ]       |
|                                            |
+------------------------------------------+
```

**Specifications:**

| Element | Detail |
|---|---|
| Title | "What should we call you?" |
| Input field | Placeholder: "Your name". Max 30 characters. Auto-focused. |
| Join button | Disabled until input has 1+ non-whitespace characters. |
| Submit on Enter | Yes. |
| Persistence | Name saved to `localStorage` key `userName`. |
| Changing name later | Small "pencil" icon next to the user's own name on the Room Page allows editing. Changing name broadcasts update to all participants. |

### 3.3 Room Page (Main Experience)

This is where all the action happens. It has two visual phases: **Voting** and **Reveal**.

**Layout -- Voting Phase:**

```
+----------------------------------------------------------+
| [om]  Room: Sprint 42 Planning          [Copy Link] [⚙]  |
+----------------------------------------------------------+
|                                                            |
|  Participants                                              |
|  +--------+  +--------+  +--------+  +--------+           |
|  | 🟢 Ana |  | 🟢 Bob |  | 🟡 Cat |  | 🔴 Dan |           |
|  |  [✓]   |  |  [ ]   |  |  [✓]   |  |  [-]   |           |
|  +--------+  +--------+  +--------+  +--------+           |
|                                                            |
|  ✓ = has voted (card face down)                            |
|  [ ] = has not voted yet                                   |
|  [-] = disconnected, no vote                               |
|                                                            |
+----------------------------------------------------------+
|                                                            |
|  [         Show Votes (2 of 4 voted)         ]             |
|                                                            |
+----------------------------------------------------------+
|                                                            |
|  Your card:                                                |
|  [?] [0] [0.5] [1] [2] [3] [5] [8] [13] [20] [40] [100]  |
|                                                            |
+----------------------------------------------------------+
|  [New Round]                            [Clear Room]       |
+----------------------------------------------------------+
```

**Layout -- Reveal Phase:**

```
+----------------------------------------------------------+
| [om]  Room: Sprint 42 Planning          [Copy Link] [⚙]  |
+----------------------------------------------------------+
|                                                            |
|  Results                                                   |
|  +--------+  +--------+  +--------+  +--------+           |
|  | 🟢 Ana |  | 🟢 Bob |  | 🟡 Cat |  | 🔴 Dan |           |
|  |   5    |  |   8    |  |   5    |  |   ?    |           |
|  +--------+  +--------+  +--------+  +--------+           |
|                                                            |
|  Average: 6.0    Median: 5    Votes: 3 of 4               |
|  [consensus bar visualization]                             |
|                                                            |
+----------------------------------------------------------+
|                                                            |
|  Your vote: 5                                              |
|  (Voting is locked until a new round starts)               |
|                                                            |
+----------------------------------------------------------+
|  [New Round]                            [Clear Room]       |
+----------------------------------------------------------+
```

**Room Page Specifications:**

| Element | Detail |
|---|---|
| Header | Room name (truncated with ellipsis if too long), Copy Link button, Settings gear icon. |
| Copy Link | Copies the full room URL to clipboard. Shows brief "Copied!" toast (1.5s, then fades). |
| Settings (gear) | Opens a small dropdown: "Change my name" and "Leave room" options only. |
| Participant list | Horizontal wrap layout (cards flow into rows as needed). Each card shows presence dot, name, and vote status. |
| Show Votes button | Centered, prominent. Shows count of votes cast (e.g. "Show Votes (3 of 5 voted)"). Enabled even if not all have voted -- the facilitator decides when to reveal. |
| Card deck | Horizontal scrollable row, fixed to bottom area. Cards are tappable. |
| New Round button | Bottom-left. Secondary style. No confirmation needed -- this action is frequent and reversible by simply voting again. |
| Clear Room button | Bottom-right. Destructive style (red text or outline). Requires confirmation dialog. |

---

## 4. UI Component Requirements

### 4.1 Room Name Input (Home Page)

| State | Appearance |
|---|---|
| Empty | Light border, placeholder text visible, button disabled (muted). |
| Focused | Border color changes to primary. |
| Filled | Placeholder hidden, button becomes active (primary color). |
| Validation error | Red border, error text below: "Only letters, numbers, spaces, and hyphens allowed." |

### 4.2 Participant Card

A compact card representing one person in the room.

| State | Visual |
|---|---|
| Not voted (voting phase) | Name + presence dot. Empty card outline below name (or a subtle "..." placeholder). |
| Voted (voting phase) | Name + presence dot. Face-down card icon below name (filled card shape, back pattern). Subtle checkmark. |
| Revealed (reveal phase) | Name + presence dot. Card face-up showing the voted value in large text. |
| Did not vote (reveal phase) | Name + presence dot. Card shows "--" or is absent. |
| Disconnected | Entire card slightly dimmed (reduced opacity ~60%). Presence dot is red. |

**Dimensions:**
- Min width: 80px. Max width: 120px.
- Name truncates with ellipsis after ~10 characters. Full name shown on hover/tap tooltip.

### 4.3 Action Buttons

| Button | Type | Color/Style |
|---|---|---|
| Create Room | Primary | Solid fill, primary brand color. |
| Join | Primary | Same as Create Room. |
| Show Votes | Primary, large | Solid fill, prominent. Full width of middle section. |
| New Round | Secondary | Outlined or ghost style. Neutral color. |
| Clear Room | Destructive | Red outlined or ghost. Red text. |
| Copy Link | Icon + text | Small, subtle. Clipboard icon. |

### 4.4 Toast Notifications

Used for non-blocking confirmations: "Link copied!", "Room cleared", "[Name] joined", "[Name] left".

| Property | Value |
|---|---|
| Position | Top-center of viewport. |
| Duration | 2 seconds, then fade out over 300ms. |
| Style | Pill-shaped, semi-transparent background, white text. |
| Stacking | Newest on top, max 3 visible. |

### 4.5 Confirmation Dialog

Used only for "Clear Room".

| Property | Value |
|---|---|
| Style | Centered modal with backdrop overlay (dimmed background). |
| Title | "Clear Room?" |
| Body | "This will remove all participants from the room. Everyone will need to rejoin." |
| Buttons | "Cancel" (secondary, left) and "Clear Room" (destructive/red, right). |
| Dismiss | Clicking backdrop or pressing Escape triggers Cancel. |

---

## 5. Card Design

### 5.1 Card Deck (User's Own Cards)

The card deck is the row of selectable values at the bottom of the room page.

**Card values:** `?`, `0`, `0.5`, `1`, `2`, `3`, `5`, `8`, `13`, `20`, `40`, `100`

**Card states:**

| State | Visual Description |
|---|---|
| **Default (unselected)** | White/light background. Thin border (1px, neutral gray). Value centered in medium-weight font. Subtle hover shadow. Cursor: pointer. |
| **Hover** | Slight lift effect (translateY -2px + shadow increase). Border darkens slightly. |
| **Selected** | Primary brand color background. White text. Elevated shadow. Slight scale-up (1.05x). Border matches background. |
| **Disabled (reveal phase)** | Reduced opacity (40%). No hover effect. Cursor: not-allowed. Selected card retains highlight but is also at reduced opacity. |

**Card dimensions:**
- Desktop: ~56px wide x 72px tall (playing-card aspect ratio ~1:1.3).
- Mobile: ~44px wide x 56px tall.
- Border radius: 8px.
- Spacing between cards: 8px.

**Interaction:**
- Tap/click to select. Selecting a different card deselects the previous one (radio behavior).
- Tap the already-selected card to deselect (user un-votes).
- Selection change is broadcast instantly via WebSocket.

### 5.2 Participant's Card (Other People's Votes)

During **voting phase** (face-down):
- A small card shape (~40px x 52px) with a subtle pattern or solid brand-color back.
- No value visible.
- Indicates "this person has voted" without revealing what.

During **reveal phase** (face-up):
- Same small card shape but white background with the value displayed prominently.
- Cards that have high/low outlier values could have a subtle highlight (e.g., red tint for highest, blue tint for lowest) to draw attention to spread. This is optional and can be deferred.

### 5.3 The "?" Card

The "?" card is special. It means "I'm not sure" or "I need more discussion."
- It is treated as a non-numeric vote.
- It is **excluded** from average and median calculations.
- In the results summary it appears as a separate count: e.g., "Average: 5.0 | Median: 5 | 1 uncertain".

---

## 6. Presence Indicators

### 6.1 Presence States

| State | Color | Dot | Meaning |
|---|---|---|---|
| **Active** | Green (#22c55e) | Solid filled circle | User's tab is open and focused, or was focused within the last 30 seconds. |
| **Idle** | Yellow (#eab308) | Solid filled circle | User's tab is open but unfocused or no user interaction for **2 minutes**. |
| **Disconnected** | Red (#ef4444) | Solid filled circle | WebSocket connection lost. User has closed the tab, lost network, or has been unreachable for **10 seconds** after last heartbeat. |

### 6.2 Timing Thresholds

| Transition | Trigger | Delay |
|---|---|---|
| Active --> Idle | `visibilitychange` event (tab hidden) OR no mouse/keyboard/touch activity | 2 minutes of inactivity, or immediately on tab hidden. |
| Idle --> Active | `visibilitychange` (tab visible) OR any user interaction | Immediate. |
| Active/Idle --> Disconnected | WebSocket `close` event or heartbeat timeout | 10 seconds after last heartbeat. The server pings every 5 seconds; if 2 consecutive pings are missed, mark as disconnected. |
| Disconnected --> Active | WebSocket reconnects successfully | Immediate. Participant card reappears in full opacity. |

### 6.3 Visual Details

- Dot size: 10px diameter.
- Position: Top-left corner of the participant card, overlapping the card edge slightly.
- Transition between colors: CSS transition over 300ms for smooth state change.
- No pulsing or animation (distracting during focused estimation work).

### 6.4 Disconnected User Behavior

- Disconnected users remain in the participant list. Their card is dimmed (60% opacity).
- Their vote (if cast before disconnect) is **preserved** and included in reveal.
- If the user reconnects, their previous vote is restored.
- Disconnected users are only removed when "Clear Room" is used or when the server garbage-collects the room (e.g., 24 hours after last activity).

---

## 7. Action Buttons

### 7.1 "Show Votes" Button

| Property | Value |
|---|---|
| Label | "Show Votes (X of Y voted)" where X = votes cast, Y = active + idle participants (excluding disconnected). |
| Position | Centered, between participant area and card deck. Full width of content area (max 400px). |
| Visibility | Voting phase only. Hidden during reveal phase. |
| Enabled | Always enabled, even if 0 votes. The facilitator may want to reveal to prompt discussion. |
| Confirmation | None. This is a frequent action; adding a confirm dialog would be hostile. |
| Keyboard shortcut | `Space` or `Enter` when the button is focused. No global hotkey (would conflict with chat or other tools). |

### 7.2 "New Round" Button

| Property | Value |
|---|---|
| Label | "New Round" |
| Position | Bottom-left of room page. |
| Visibility | Always visible in both phases. |
| Confirmation | None. Clearing votes is low-cost -- people can simply re-vote. |
| Effect | Clears all votes, returns to voting phase, re-enables card deck for all participants. |
| Broadcast | All clients receive the reset event. Brief toast: "New round started." |

### 7.3 "Clear Room" Button

| Property | Value |
|---|---|
| Label | "Clear Room" |
| Position | Bottom-right of room page. |
| Visibility | Always visible. |
| Confirmation | **Yes.** Modal confirmation dialog (see section 4.5). This action is destructive and infrequent. |
| Effect | Removes all participants. Each connected client receives a "room cleared" event and is shown a message with a "Rejoin" button (which re-triggers the name entry flow if needed, or re-adds them if name is in localStorage). |

### 7.4 Button Layout

```
Desktop:
+------------------------------------------------------------+
|  [New Round]                              [Clear Room]      |
+------------------------------------------------------------+

Mobile:
+------------------------------------------------------------+
|  [New Round]                              [Clear Room]      |
+------------------------------------------------------------+
```

Both buttons are on the same row, left/right aligned. On very narrow screens (<320px), they stack vertically, full width.

---

## 8. Responsive Design Notes

### 8.1 Breakpoints

| Breakpoint | Range | Layout adjustments |
|---|---|---|
| Mobile | < 640px | Single column. Cards in deck are smaller (44x56px). Participant cards wrap into 2-column grid. |
| Tablet | 640px - 1024px | Participant cards wrap freely. Card deck may scroll horizontally. |
| Desktop | > 1024px | All content comfortable in single view. No scrolling needed for up to ~12 participants. |

### 8.2 Card Deck on Mobile

- The 12 cards must be accessible without excessive scrolling.
- Layout: wrap into 2 rows of 6, or a horizontally scrollable single row with snap points.
- **Recommended:** 2 rows wrapped. This avoids hidden cards (users may not realize they can scroll).

```
Mobile card deck (2 rows):
[?] [0] [0.5] [1]  [2]  [3]
[5] [8] [13]  [20] [40] [100]
```

### 8.3 Touch Targets

- Minimum touch target: 44x44px (Apple HIG guideline).
- Cards on mobile meet this at 44x56px.
- Buttons must also respect this minimum. Padding accordingly.

### 8.4 Viewport Considerations

- The card deck should always be visible without scrolling. It is the primary interaction element.
- On mobile, use `position: sticky` or `position: fixed` at the bottom to keep the deck always accessible.
- The participant area scrolls if there are many participants; the deck does not.

### 8.5 Landscape Mobile

- Card deck stays at bottom.
- Participant area is given less vertical space.
- No special layout changes needed; the wrapping behavior handles it.

---

## 9. Edge Cases & Micro-interactions

### 9.1 Changing Vote Before Reveal

- **Allowed freely.** User taps a different card; old selection is deselected, new one is selected.
- Other participants see no visible change (they only see the face-down "voted" indicator). No flicker, no notification.
- The server replaces the old vote with the new one atomically.
- If a user deselects their card (taps the selected card again), their status changes back to "not voted" and other participants see the face-down card disappear.

### 9.2 Joining Mid-Vote

- New participant appears in the participant list immediately with "not voted" status.
- They see the current phase:
  - If **voting phase**: they see the deck and can vote. They see who has voted (face-down cards) but not what.
  - If **reveal phase**: they see all revealed votes and statistics. They did not participate in this round. Their card shows "--" (no vote).
- Brief toast for existing participants: "[Name] joined."

### 9.3 Empty Room

- If all participants disconnect or leave, the room remains on the server (no immediate cleanup).
- A user visiting the room link joins as the sole participant.
- The room page shows normally with just one participant card.
- "Show Votes" still works with 1 person (useful for testing, and the tool should not judge how you use it).
- Server-side room expiration: rooms with zero connected users for **24 hours** are garbage-collected.

### 9.4 "?" Votes in Calculations

- "?" is a valid vote that signals uncertainty.
- **Excluded from average and median** calculations.
- If ALL votes are "?", statistics show: "No numeric votes to summarize."
- Display format when some are "?": "Average: 5.0 | Median: 5 | 2 uncertain" (where 2 is the number of "?" votes).
- In the participant card during reveal, "?" is shown as-is.

### 9.5 Reconnection Behavior

- When a WebSocket connection drops, the client immediately begins reconnection attempts.
- Reconnection strategy: exponential backoff starting at 500ms, max 10 seconds, with jitter.
- During reconnection: a subtle banner at the top of the room page: "Reconnecting..." (yellow background).
- On successful reconnect:
  - Client sends its stored `userName` and room ID.
  - Server restores the participant to the room with their previous vote (if the room/round is still active).
  - Banner disappears. Toast: "Reconnected."
  - If the round has changed (new round started while disconnected), the user enters the current phase with no vote.
- If reconnection fails after **30 seconds** of attempts: banner changes to "Connection lost. [Retry]" with a manual retry button.

### 9.6 Duplicate Names

- Allowed. The system does not enforce unique names. Each participant is identified internally by a session ID (e.g., a UUID stored in `localStorage` alongside the name).
- If two participants have the same name, both appear in the list. Users can differentiate by presence status or by knowing their team.
- This keeps the system simple. Names are display-only.

### 9.7 Room Not Found

- If a user visits a room URL that does not exist on the server (e.g., room expired or never created):
  - Show a simple page: "Room not found. It may have expired. [Create a New Room]" linking to the home page.

### 9.8 Very Large Number of Participants

- The design supports up to ~20 participants comfortably. Beyond that, the participant area scrolls.
- No hard limit enforced. Planning poker with more than 15 people is an antipattern anyway, but the tool should not block it.

### 9.9 Browser Back/Forward Navigation

- Pressing Back from the room page goes to the home page (or previous page in browser history).
- The room page uses the URL as the source of truth. Returning to it (via Forward or re-entering the URL) reconnects to the room.
- No "are you sure you want to leave?" prompts. The tool should not be clingy.

---

## 10. Scrum Poker Best Practices Review

Having facilitated hundreds of planning poker sessions, here is an honest assessment of what this tool covers and what might be missing, filtered through the KISS lens.

### 10.1 What We Cover Well

- **Core voting flow.** Select, reveal, discuss, repeat. This is the 90% use case and we nail it.
- **Low barrier to entry.** No signup, no install, share a link. This is the single biggest feature. Tools that require accounts lose half the team at onboarding.
- **Visual clarity of results.** Average + median + spread is enough for most teams.
- **Presence awareness.** Knowing who is actually here prevents "let's wait for Bob" when Bob has been on his phone for 10 minutes.

### 10.2 Considered and Intentionally Excluded

| Feature | Why excluded |
|---|---|
| **Timer / Discussion phase** | Adds complexity for marginal value. Teams that want time-boxing use a separate timer. Embedding one creates pressure that can reduce estimate quality. |
| **Vote confidence** (e.g., "I'm 70% sure") | Interesting but niche. The "?" card already covers the "I don't know" case. Adding confidence to every vote clutters the interface. |
| **Task/story integration** (JIRA, etc.) | Violates the self-hosted/no-auth principle. Also, task management tools have their own poker features. Our value is being independent and simple. |
| **History / Vote log** | Planning poker estimates are consumed immediately ("write it in the ticket"). Storing history adds server complexity with almost no user value. |
| **Custom card decks** | The Fibonacci-like sequence is the industry standard. Custom decks serve <5% of teams and add settings UI complexity. |
| **Facilitator role** | All participants are equal. Any person can reveal or reset. This prevents the "facilitator left and we're stuck" problem. Flat authority is simpler and more resilient. |

### 10.3 Recommended Addition: Consensus Highlight

**Include this.** It is low-cost and high-value.

After reveal, if all numeric votes are identical, show a visual celebration state:

- "Consensus! Everyone voted 5" with a subtle green background on the results area.
- This gives the team a micro-dopamine hit and signals "no discussion needed, move on."

If votes have high spread (e.g., range > 2 Fibonacci steps), show a subtle highlight:

- "High spread (3 to 13) -- discussion recommended" in a neutral info style.
- This nudges the team to talk about the outliers without being prescriptive.

**Implementation:** a simple bar or color strip under the statistics.

```
Consensus:
+----------------------------------------------------------+
| ✓ Consensus: Everyone voted 5                             |
+----------------------------------------------------------+

High spread:
+----------------------------------------------------------+
| ↔ Spread: 3 to 13 — consider discussing                  |
+----------------------------------------------------------+

Normal (no highlight):
+----------------------------------------------------------+
| Average: 6.0 | Median: 5 | Votes: 4                      |
+----------------------------------------------------------+
```

### 10.4 Recommended Addition: "Nudge" Indicator

**Consider this (optional, low priority).**

In the "Show Votes" button, displaying "2 of 5 voted" already applies gentle social pressure. No additional nudge mechanism is needed. The button label is the nudge.

### 10.5 Not Recommended: Anonymity Mode

Some tools offer anonymous voting (names hidden during reveal). This is counterproductive for planning poker -- the entire point is to surface disagreements and have the high/low voters explain their reasoning. Anonymity defeats this purpose.

---

## Appendix A: WebSocket Events

For developer reference, the key real-time events the system must support:

| Event | Direction | Payload |
|---|---|---|
| `join` | Client --> Server | `{ sessionId, userName, roomId }` |
| `participant_joined` | Server --> All | `{ sessionId, userName }` |
| `participant_left` | Server --> All | `{ sessionId }` |
| `vote` | Client --> Server | `{ sessionId, value }` |
| `vote_cast` | Server --> All | `{ sessionId }` (no value -- just "someone voted") |
| `reveal` | Client --> Server | `{ }` |
| `votes_revealed` | Server --> All | `{ votes: [{ sessionId, userName, value }], average, median }` |
| `new_round` | Client --> Server | `{ }` |
| `round_reset` | Server --> All | `{ }` |
| `clear_room` | Client --> Server | `{ }` |
| `room_cleared` | Server --> All | `{ }` |
| `presence_update` | Client --> Server | `{ sessionId, status: "active" | "idle" }` |
| `presence_changed` | Server --> All | `{ sessionId, status }` |
| `heartbeat` | Bidirectional | `{ }` (keep-alive ping/pong) |

---

## Appendix B: Color Palette Suggestion

A minimal palette to keep the design clean:

| Role | Color | Usage |
|---|---|---|
| Primary | #6366f1 (Indigo 500) | Buttons, selected card, links. |
| Primary hover | #4f46e5 (Indigo 600) | Button hover state. |
| Background | #fafafa | Page background. |
| Surface | #ffffff | Cards, modals, inputs. |
| Text primary | #18181b (Zinc 900) | Body text. |
| Text secondary | #71717a (Zinc 500) | Placeholder, labels, secondary info. |
| Border | #e4e4e7 (Zinc 200) | Input borders, card borders. |
| Success/Consensus | #22c55e (Green 500) | Consensus indicator, active presence. |
| Warning/Idle | #eab308 (Yellow 500) | Idle presence. |
| Destructive | #ef4444 (Red 500) | Clear Room button, disconnected presence. |

---

## Appendix C: Accessibility Notes

- All interactive elements must be keyboard-navigable (Tab order, Enter/Space to activate).
- Card selection must be announced by screen readers ("Selected 5 points", "Deselected").
- Presence colors must not be the only indicator -- add a text label or icon for colorblind users (e.g., dot + tooltip with "active/idle/disconnected").
- Minimum contrast ratio: 4.5:1 for text, 3:1 for UI components (WCAG AA).
- The reveal animation should respect `prefers-reduced-motion`.
