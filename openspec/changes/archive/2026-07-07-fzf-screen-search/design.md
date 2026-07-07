## Context

The dashboard TUI (`internal/ui/model.go`) uses a bubbletea model with a hand-built cursor and two views (Notifications, Sessions). Entry lists can grow long as sessions accumulate. Currently the only filtering in Sessions is the `f` (active-pane) toggle. There is no text search in either view.

`github.com/charmbracelet/bubbles/textinput` is already in the dependency tree (bubbles v1.0.0). `github.com/sahilm/fuzzy` is a small, zero-dependency fuzzy-matching library used by bubbletea ecosystem projects.

## Goals / Non-Goals

**Goals:**
- `/` enters a search mode that fuzzy-filters the currently visible row list in real time.
- The same mechanism works identically in Notifications view and Sessions view (both levels).
- All existing action keys (enter, p, w, h, v) operate on the filtered list.
- Search is scoped to the current view and cleared on drill navigation.

**Non-Goals:**
- Regex or exact-match mode — fuzzy only.
- Persistent filter across dashboard restarts.
- Searching within pane content or transcript text.
- A separate search pane or popup (search is inline, bottom of header area).

## Decisions

### Use `sahilm/fuzzy` for matching (not `strings.Contains`, not fzf subprocess)

`sahilm/fuzzy` scores and ranks matches exactly like fzf (character-level subsequence matching). `strings.Contains` would feel wrong to users expecting fzf-style behavior. Spawning an external `fzf` process inside a bubbletea TUI is complex, slow on each keystroke, and breaks the popup's terminal ownership. `sahilm/fuzzy` is 1 file, no transitive deps, MIT license.

**Alternative considered:** `github.com/ktr0731/go-fuzzyfinder` — takes over the terminal entirely; incompatible with bubbletea rendering.

### Store search state in the model (not a derived filtered slice)

Four fields live on the model struct:
- `searchMode bool` — whether search is active at all
- `searchFocus bool` — whether the textinput currently has keyboard focus (`true`) vs the table (`false`)
- `searchQuery string` — the current filter string
- `searchInput textinput.Model` — the bubbletea textinput component

Filtered slices are computed inline in `renderNotificationsView` / `renderSessionsView` / `filteredDrill` from the canonical `m.entries` / `m.sessionItems`. This avoids maintaining a parallel data structure and ensures the filtered view is always consistent with the source of truth.

### Two focus sub-states: input-focused and table-focused

When search mode is active, `Tab` toggles keyboard focus between the textinput and the table — it does not clear the filter or switch views. This allows users to type a query, then press Tab to use all normal action keys on the filtered list without leaving search mode. A second `Tab` returns focus to the input to refine the query.

- **Input focused** (`searchFocus=true`): alphanumeric keystrokes go to `textinput.Update`. The textinput shows a blinking cursor. Action keys `enter`/`space` still fire (they fall through to the table handler). `esc` exits search entirely.
- **Table focused** (`searchFocus=false`): the full normal key switch is active — `j/k`, `p`, `w`, `h`, `v`, `q`, `s`, `f`, `enter`. The textinput is blurred (no cursor) but still displays the current query. `esc` exits search entirely.

### Filter field: what columns are matched

- Notifications view: window name + path + session name concatenated as the match target per row.
- Sessions view L1: project path (key) as match target.
- Sessions view L2: session ID + window name as match target.

Rationale: these are the columns the user can read and type against. Status and age are not searched.

### `esc` exits search entirely from either focus sub-state

`esc` always calls `clearSearch()` when `searchMode` is true, regardless of whether the input or table has focus. This matches vim semantics: esc is always "get me out of this mode." The two-tap esc pattern (first esc exits search, second esc back-navigates or quits) is preserved.

### Clear filter on drill navigation; Tab when not searching clears implicitly

Carrying a filter across drill-in/out produces confusing state. Filter is reset on:
- Drill-in (enter at Sessions L1)
- Drill-out (esc at Sessions L2 when not in search mode)

View switch via Tab only happens when `searchMode=false` — so the next view always starts clean. There is no explicit "clear on Tab" needed.

## Risks / Trade-offs

- [New dep `sahilm/fuzzy`] → Small and stable (100+ stars, no transitive deps). Risk is low; if removed later, a `strings.Contains` fallback is trivial to substitute.
- [Fuzzy ranking changes cursor position] → After each keystroke the filtered list shrinks/reorders. Cursor is clamped to `max(0, len(filtered)-1)` on every update. The user may need to re-navigate after typing but this is expected fzf behavior.
- [Two focus sub-states add UX complexity] → The textinput cursor (blinking vs absent) provides clear visual feedback. Tab is a familiar toggle in terminal UIs (like form fields). The tradeoff is a richer but slightly more modal interaction.
