## Context

The dashboard TUI (`internal/ui/model.go`) renders each view as an unbounded `strings.Builder` string. The bubbletea runtime prints this string directly into the terminal. When the rendered content is taller than the terminal, lines beyond the bottom of the screen are clipped by the terminal emulator — they are never shown and the user has no way to reach them.

The model already receives `tea.WindowSizeMsg` and stores `m.width`. It does not currently track `m.height`, so there is no way to know the available vertical space.

The `charmbracelet/bubbles` module is already a direct dependency (textinput and timer are imported). The `viewport` sub-package is part of the same module and is available at zero additional dependency cost.

## Goals / Non-Goals

**Goals:**
- All three renderable list surfaces (Notifications, Sessions L1, Sessions L2) scroll when content exceeds terminal height
- The viewport flexes — it expands/contracts to fill exactly the space left after fixed elements render
- The selected cursor row is always kept visible (scroll-follows-cursor)
- A position indicator (`n/total`) appears at the right of the footer hint line when the list overflows
- `m.height` is tracked via `tea.WindowSizeMsg`; the viewport is resized on every `WindowSizeMsg` and whenever fixed-element visibility changes
- `ctrl+d` / `ctrl+u` for half-page scroll (vim-style), moving both viewport and cursor together
- PgUp/PgDn provided for free by `bubbles/viewport`
- Implementation stays entirely within `internal/ui/model.go`; no other packages change

**Non-Goals:**
- Mouse scroll support
- Animated scrolling
- Per-view separate viewport instances (one viewport is shared, content is swapped on view change)

## Decisions

### Decision: Use `bubbles/viewport` with flex height

**Choice**: Embed a single `viewport.Model` in the model. On each `WindowSizeMsg` and on any state change that alters the fixed-row count, call `recalcViewport()` which sets:
```
viewport.Width  = m.width
viewport.Height = max(1, m.height - fixedRows)
viewport.SetContent(renderListContent())
```
The `View()` method renders fixed elements (tab header, search bar, setup status, toast, blank separator) above the viewport, and the footer hint below it. The viewport fills the remaining vertical space.

**Rationale**: `bubbles/viewport` handles scroll state, PgUp/PgDn, and half-page movement out of the box. All rows in this TUI are single-line, so cursor-follows-selection is simple integer arithmetic against `viewport.YOffset`. This is the standard Charm idiom for full-height layouts and aligns with how the rest of the charmbracelet ecosystem expects viewports to be used.

**Alternative considered**: Manual `scrollOffset int` — rejected because `bubbles/viewport` gives PgUp/PgDn for free, is the conventional Charm approach for this exact use case, and doesn't require reimplementing scroll-boundary clamping.

### Decision: Single shared viewport instance, content swapped on view/drill change

**Choice**: One `viewport.Model` in the model struct. When the active view or drill level changes, `recalcViewport()` is called to re-render content and reset scroll position to top.

**Rationale**: Maintaining separate viewports per view (notif, L1, L2) would require tracking which is active and syncing dimensions across all three on every `WindowSizeMsg`. One instance keeps the model simple; the scroll position reset on context change matches the expected UX (you're at the top when you switch views).

### Decision: List content (header + separator + rows) goes inside the viewport

**Choice**: `renderListContent()` returns the column header row, separator, and all data rows as a single string. This string is passed to `viewport.SetContent()`. Fixed elements outside the viewport are: tab header, search bar, setup status, toast, blank separator, and footer hint.

**Rationale**: Putting headers inside the viewport content is the simpler option — it avoids having to conditionally add 2 to `fixedRows` when entries are present vs. absent, which would require resizing the viewport on every data reload. Headers scrolling away on large lists is acceptable and mirrors conventions in many TUI tools (lazygit, k9s).

**Alternative considered**: Headers as fixed elements above the viewport — rejected because it complicates `fixedRows` computation and requires a `recalcViewport()` call every time the entries-present state changes.

### Decision: `fixedRows` computed from live state in `recalcViewport()`

**Choice**:
```
fixedRows = 1  // tab header
           + (1 if searchMode)
           + (setupStatusLineCount if viewNotifications)  // 0 or 2
           + (1 if toast active)
           + 1  // blank separator
           + 1  // footer hint
```
`recalcViewport()` is called from: `WindowSizeMsg` handler, search mode toggle, toast timer events, and settings-changed events.

**Rationale**: The fixed elements are fully enumerable and each contributes a known line count. Computing on demand avoids stale sizing. The `renderSetupStatus` result is 0 or 2 lines (not 1); using `countLines(s)` handles this without hardcoding.

### Decision: Cursor-follows-selection via `viewport.SetYOffset`

**Choice**: After any cursor movement, enforce the scroll invariant:
```go
if cursor < vp.YOffset {
    vp.SetYOffset(cursor)
} else if cursor >= vp.YOffset + vp.Height {
    vp.SetYOffset(cursor - vp.Height + 1)
}
```
Note: `cursor` here is the data-row index; if the header row is included in content, the actual line is `cursor + headerLines` (2 for header + separator).

**Rationale**: Minimum-scroll is the conventional terminal TUI behavior (vim, less, fzf). Centering on every keystroke causes jarring jumps when near the top/bottom.

## Risks / Trade-offs

- **`m.height` is 0 before first `WindowSizeMsg`**: Guard `termHeight()` with a `24` default; viewport is initialized with `New(termWidth(), 24 - fixedRows)` and resized on the first real message.
- **`fixedRows` must stay in sync**: If a new fixed element is added to the header area, `recalcViewport()` must be updated in tandem. This is a maintenance coupling, mitigated by keeping the formula in one function.
- **`viewport.SetContent` on every key press**: With many sessions, re-rendering the full list on every cursor movement could be slow. In practice the list is bounded (hundreds of entries at most) and lipgloss rendering is fast; this is acceptable.
- **`setupStatus` line count**: `renderSetupStatus` returns `""` (0 lines) or a 2-line string. Using `strings.Count(s, "\n")` on the rendered result correctly counts 0 or 2.
