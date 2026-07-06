## Context

The dashboard TUI (`internal/ui/model.go`) renders a list of active notifications. Each row currently uses an unlabeled `fmt.Sprintf` format string with fixed widths, showing WindowName, Session, Pane ID, Status, and Age. The rendering has two structural problems: (1) the pane ID column is meaningless to users, and (2) a bug causes ANSI color escape codes to be included in the `%-9s` width calculation, drifting column alignment. The change adds column headers, reorders columns, replaces pane ID with a live path fetch, and adds status icons.

## Goals / Non-Goals

**Goals:**
- Column headers + separator so users know what they're looking at
- Status leads each row (icon + colored text), padded correctly
- Path column: pane's current working directory, trimmed to last 2 components, `$HOME → ~`
- Drop raw pane ID from the visible table
- Fix the ANSI-inflated width alignment bug

**Non-Goals:**
- Changes to the store schema, hook wiring, or notification delivery
- Scrolling, sorting, or filtering UI
- Persisting the path in the JSONL store (fetched live at load time)

## Decisions

### D1: Fetch path live in `loadEntries()`, not stored in `store.Record`

Path is live state (the user can `cd` after the notification was created). Fetching at load time in `loadEntries()` is correct. `loadEntries()` is only called on data-change events (not per-frame), so the extra `tmux display-message` calls per pane are infrequent.

Alternative considered: store `pane_current_path` in the JSONL record at notification time. Rejected — stale path from when the notification was created is less useful than the current directory.

### D2: Add `Path string` to `entry`, not `store.Record`

`entry` is the TUI-local view model. Adding path there avoids polluting the persistent store schema with UI-derived state and keeps the store format stable.

### D3: Fix alignment by padding plain text before colorizing

```go
// Before (bug): ANSI codes included in %-9s width
badge := style.Render(status) // includes ANSI
line := fmt.Sprintf("%-9s", badge) // wrong width

// After (fix): pad first, color second
padded := fmt.Sprintf("%-9s", status) // plain text, correct width
badge := style.Render(padded)         // ANSI wraps padded string
```

Alternative: use `lipgloss.Width()` to compute padding manually. Rejected — the simple pad-then-color approach is cleaner and doesn't require lipgloss width introspection.

### D4: New helper `tmuxclient.PanePath(paneID string) (string, error)`

Encapsulates `tmux display-message -p -t <pane> '#{pane_current_path}'`. Path truncation (last 2 components, home substitution) happens in `loadEntries()` in the UI layer, not in the tmux client — keeps the client generic.

## Risks / Trade-offs

- **Tmux call per pane on load**: `loadEntries()` now calls `tmux display-message` for each live entry. At typical scale (1–5 claude sessions) this is negligible. → Mitigation: errors are silently ignored, returning empty string; no user-visible impact if tmux call fails.
- **Emoji rendering width**: Terminal emoji width varies (single vs double-width). Icons ⏳, ⚙, 💤 are tested in the target WSL2 + tmux environment. → Mitigation: use a fixed-width icon slot; if emoji renders wide, the padding accommodates it.
- **Column overflow on narrow terminals**: Fixed-width columns may wrap or truncate on very narrow terminals. → Mitigation: no change from current behavior; this is a pre-existing issue not in scope here.
