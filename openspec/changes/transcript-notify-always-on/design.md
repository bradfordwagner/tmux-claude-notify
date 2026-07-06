## Context

Currently, `claude-notify notify` is invoked only by the Claude Code `Stop` hook, which fires at the end of a complete response turn. This means:
- No notification during tool-heavy turns until the turn fully completes
- No distinction between "claude is actively working" and "claude is waiting for you"
- Dashboard only knows about state that the Stop hook explicitly wrote to JSONL

Claude Code writes a JSONL transcript for each session at `~/.claude/projects/<encoded-path>/<session-id>.jsonl`. These transcripts contain a stream of events (user messages, assistant messages, tool calls, tool results) that can be used to derive agent state in near-realtime.

The dashboard is currently a short-lived popup. It needs to handle state from both the "open" period (transcript watcher active) and the "closed" period (only Stop hook was writing to JSONL). Reconciliation on open bridges the gap.

## Goals / Non-Goals

**Goals:**
- Embed a transcript watcher inside the dashboard TUI process — no separate daemon
- Watch Claude Code transcript files via fsnotify to derive `running`, `waiting`, `idle`, `stale` states while the dashboard is open
- On dashboard open: read current transcript state and reconcile with JSONL store so stale/incorrect entries are corrected
- Retain the Stop hook as a fallback signal — it writes to JSONL even when the dashboard is closed
- Extend store.Record with a `status` field; existing records default to `waiting`

**Non-Goals:**
- Persistent background daemon or separate subcommand
- Changes to TPM entry point or keybinding mechanism
- Fleet-view / `claude agents` background agent integration
- Supporting non-tmux terminal multiplexers

## Decisions

### D1: Watcher lives in TUI process, started in model.Init()

`ui.newModel()` creates the transcript watcher alongside the existing fsnotify watcher for settings/JSONL files. The watcher runs as a goroutine feeding a bubbletea `Cmd` channel, same pattern as the existing `watchCmd`. On `q`/`esc`/`ctrl+c` the watcher is closed.

**Alternatives considered:**
- Separate daemon process: adds IPC complexity, requires TPM changes, harder to reason about lifecycle
- Background goroutine outside bubbletea: harder to coordinate shutdown

### D2: Reconcile on open — re-derive state from transcripts at startup

On `newModel()`, after loading JSONL entries, the watcher immediately scans active transcript files and calls `store.UpdateStatus` or `store.ClearPane` to correct any entries that changed while the dashboard was closed. This makes the open state authoritative.

**Alternatives considered:**
- Trust JSONL state blindly: stale "waiting" entries would persist indefinitely if the user never opened the dashboard
- Reconcile lazily on first watcher event: adds latency before dashboard shows correct state

### D3: Transcript parsing strategy — tail + heuristic

Read the last N lines of each active transcript file on change events:
- `tool_use` event → `running`
- `assistant` message, then silence for ≥2s → `waiting`
- `user` message (i.e. user just responded) → clear the notification (`ClearPane`)
- No file activity for >5 minutes → `stale`

**Alternatives considered:**
- Full parse from start: expensive for large sessions
- inotify tail: fsnotify already a dep, works cross-platform

### D4: Stop hook remains — writes to JSONL as fallback

When the dashboard is closed, the transcript watcher is not running. The Stop hook still fires `claude-notify notify` which writes to JSONL. When the dashboard next opens, reconciliation corrects any status fields that are stale. This means JSONL is always at least as fresh as the last Stop event even without the dashboard.

### D5: Status field added to store.Record

`Status string` added to `store.Record` (default `"waiting"` for backwards compat). New `store.UpdateStatus(paneID, status string) error` rewrites the most recent uncleared record's status atomically.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| Claude Code transcript format undocumented, may change | Parse defensively; skip unrecognized event types; treat errors as `unknown` → fallback to Stop hook behavior |
| Many old transcript files accumulate | Only watch files modified within past 24 hours |
| Reconciliation on open takes time for many sessions | Bound scan to active sessions (modified in last 24h); run reconciliation asynchronously, don't block TUI render |
| User message in transcript → ClearPane fires | Correct: if the user responded, the session is no longer "waiting for input" |

## Confirmed Transcript Schema (Task 1.1)

Read from real transcripts at `~/.claude/projects/*/*.jsonl`:

```json
{"type":"assistant","timestamp":"2026-07-05T01:54:58.312Z","message":{"role":"assistant","stop_reason":"tool_use","content":[{"type":"thinking"},{"type":"tool_use"}]}}
{"type":"user","timestamp":"...","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"...","content":"...","is_error":false}]}}
{"type":"user","timestamp":"...","message":{"role":"user","content":[{"type":"text","text":"<user typed response>"}]}}
```

State derivation rules (confirmed against opensessions `agent_parsers.rs`):
- `type:"assistant"`, content has `tool_use` → `running`
- `type:"assistant"`, `stop_reason:"end_turn"` → `done` → `waiting` after ≥2s silence
- `type:"user"`, content has `tool_result` → `running` (Claude processing tool output, not real user input)
- `type:"user"`, content is `text` (or any non-`tool_result`) → user responded → **clear notification**

Pane correlation: `tmux list-panes -a -F "#{pane_id} #{pane_current_path} #{pane_current_command}"`, filter `pane_current_command` matches prefix `claude*` (covers `claude`, `claude-code`, future variants), match `pane_current_path` against decoded project dir.

Path decoding: replace all leading/separator `-` with `/`, call `os.Stat` to validate; if stat fails, path is ambiguous (skip).

## Open Questions

- Should `stale` threshold (5 min) be a TPM option (`@claude-notify-stale-minutes`)?
