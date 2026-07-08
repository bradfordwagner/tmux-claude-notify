## Context

The dashboard is the only visibility surface for pending notifications. Users must actively open it to see how many sessions are waiting. Two low-friction paths are missing:

1. **Ambient count** — a `status-right` badge that stays visible without any keypress
2. **Direct jump** — a single keybinding that navigates to the oldest waiting pane and clears it, bypassing the dashboard entirely

Both features share a common need: querying the oldest/count of uncleared notifications from the JSONL store.

## Goals / Non-Goals

**Goals:**
- `claude-notify status` exits 0 and prints a formatted count string (e.g. `⚡ 2`) usable in `status-right` via `#(...)`
- `@claude-notify-statusline` TPM option: when set, appends a `status-right` segment at TPM load time
- `@claude-notify-jump-key` TPM option (default `C-M-P`): binds a key that jumps to the oldest uncleared pane and clears it
- `store.OldestUncleared()` helper returns the uncleared entry with the smallest `ts`
- README updated with both new options

**Non-Goals:**
- Daemon or background polling — `status-right` polling via `#(...)` is handled by tmux itself
- Fancy status segment formatting (color, separators) — plain text output; user wraps it in their own powerline segments
- Multiple jump keybindings for different priority orderings

## Decisions

### `status` subcommand output format

Output: `⚡ N` where N is the count of uncleared entries. Prints nothing (empty string) when N = 0, so the segment disappears naturally without special tmux option gymnastics.

Alternative considered: always print, even `⚡ 0`. Rejected — an always-visible zero adds noise; empty output removes the segment cleanly.

### TPM-level statusline wiring

`tmux-claude-notify.tmux` reads `@claude-notify-statusline`. If non-empty, it appends the value to `status-right` using:
```bash
tmux set-option -ga status-right " #($PLUGIN_DIR/bin/claude-notify status)"
```

The `@claude-notify-statusline` option acts as a simple on/off gate — any non-empty value enables it. The actual segment format (spacing, delimiters) is user-controlled by setting the option to whatever prefix/wrapper they want (e.g. `[` or just leaving it blank for plain output). Simplest approach: setting `@claude-notify-statusline 1` appends `#(claude-notify status)` with a leading space.

Alternative considered: always-on status-right modification. Rejected — it would break users who manage `status-right` themselves and don't want the badge.

### Quick-jump key binding

`@claude-notify-jump-key` (default `C-M-P`) runs `claude-notify jump` via `tmux bind-key`. The `jump` subcommand:
1. Calls `store.OldestUncleared()` — returns nil if nothing waiting
2. If nil: exits silently (nothing to do)
3. If found: calls `runClear(paneID)` — the same helper used by the dashboard enter handler (SelectPane + SelectWindow + ClearPane + ClearWindowStyle if no siblings remain)

Re-using `runClear` keeps clear semantics identical between dashboard and quick-jump paths.

### `OldestUncleared` vs `NewestUncleared`

Oldest-first (smallest `ts`) prioritizes sessions that have been waiting longest. This matches "most urgent" intuition for interactive use. The dashboard already shows newest-first for scanning; quick-jump addresses the opposite use case.

### Conflict with existing `C-M-p` binding

`C-M-p` is already the dashboard binding. Default for jump is `C-M-P` (uppercase P, i.e. `Shift`). In tmux, `C-M-P` is a distinct binding from `C-M-p`. No conflict.

## Risks / Trade-offs

- **`#(...)` polling cost**: tmux re-runs `claude-notify status` on every status-right refresh (default every 15s). The `status` subcommand only reads the JSONL file — O(N lines) file read with no tmux calls — so this is negligible.
- **JSONL read on jump**: `jump` reads the JSONL to find the oldest entry, then issues tmux commands. If the store is large, the read is still fast (linear scan, small file in practice). No concern.
- **Entry-point thin constraint**: The existing spec (`tpm-entry-point`) has a scenario "No hook registration at TPM load time" that says the entry point SHALL NOT modify `status-right`. The `@claude-notify-statusline` wiring is opt-in and gated on the option being set — a deliberate user choice, not unconditional modification. The delta spec relaxes this scenario for the conditional statusline case.

## Migration Plan

No migration needed. Both features are additive and opt-in via TPM options. Existing users see no change unless they set `@claude-notify-statusline` or `@claude-notify-jump-key`.
