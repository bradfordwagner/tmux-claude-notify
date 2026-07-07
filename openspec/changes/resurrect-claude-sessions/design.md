## Context

After a system reboot or tmux-resurrect session restore, tmux panes are reset to bare shells. Any `claude` foreground session that was running is gone. This plugin already maintains `sessions.jsonl` — an index of every known claude session keyed by session ID, with `project_path`, `pane_id`, and `tmux_session` fields. The watcher's `listClaudePanes()` uses `pane_current_command` to detect active claude processes. These two sources give us everything needed to implement resurrect without reaching into proc internals.

## Goals / Non-Goals

**Goals:**
- `resurrect save`: snapshot current claude panes (positional key + session ID + project path) to a JSON sidecar
- `resurrect restore`: replay `claude --resume <session-id>` into matching live panes by positional key
- No new persistent background process; both subcommands are one-shot

**Non-Goals:**
- Automatic hook wiring into tmux-resurrect plugins (user calls the commands manually or via their own hooks)
- Restoring tmux windows/panes themselves — tmux-resurrect already does that
- Handling sessions with no `--resume`-able ID (new sessions with empty `session_id` are skipped)

## Decisions

### D1: Source of session ID and project path during save

**Decision**: Read `sessions.jsonl` (the plugin's own index) rather than re-deriving from proc or transcript files.

**Rationale**: `sessions.jsonl` is already kept in sync by the watcher and is the authoritative source. Re-reading `/proc/<pid>/environ` for every pane is brittle and redundant. The sessions index also stores the `project_path` directly, avoiding path-recovery work at save time.

**Alternative considered**: Walk `~/.claude/projects/` at save time and match by `pane_current_path`. Rejected: requires re-encoding the path and finding the latest transcript, adding error surface with no benefit.

### D2: Positional key format

**Decision**: Use `(tmux_session, window_index, pane_index)` as the positional key — identical to the approach in bradfordwagner/go.bin's ks resurrect.

**Rationale**: tmux-resurrect restores panes at the same session/window/pane indices, so positional keys remain stable across save/restore cycles. Pane IDs (`%42`) are ephemeral and change after restart.

**Alternative considered**: Keying by window name. Rejected: window names can be duplicate or empty; index is fragile.

### D3: Sidecar file format and location

**Decision**: JSON file at `~/.local/share/tmux-claude-notify/resurrect.json` (same data dir as `notifications.jsonl` and `sessions.jsonl`). Overridable via `@claude-notify-resurrect-file` TPM option.

**Rationale**: Consistent with existing plugin storage conventions. JSON (not JSONL) because the whole file is read/written atomically.

### D4: Restore command for active panes

**Decision**: If a pane in the sidecar is currently running `claude*` (active), skip it silently — don't send `--resume` into a live session.

**Rationale**: Restore should be idempotent and non-destructive. Sending keys into an active session is confusing and potentially data-loss-y.

### D5: Package location

**Decision**: New `internal/resurrect/` package with `Save()` and `Restore()` funcs. New `resurrect` subcommand under `cmd/claude-notify/main.go`.

**Rationale**: Mirrors the structure used in the ks CLI for the same feature, and keeps the binary the single entry point (no new bash scripts).

## Risks / Trade-offs

- [Stale sidecar after long periods] → Restore silently skips panes not present in the live tmux state; stale entries are harmless.
- [sessions.jsonl out of sync if dashboard never opened] → The Stop hook doesn't write to sessions.jsonl directly; if the dashboard was never opened, session IDs may be missing. Mitigation: save falls back to listing active `claude*` panes and extracting session ID from the latest transcript filename.
- [User calls restore twice] → Idempotent: second restore hits active panes and skips them (D4).
- [Project path missing] → If `project_path` is empty in sessions.jsonl, the `RecoverPath()` BFS walk is attempted. If that also fails, the entry is skipped with a log line.
