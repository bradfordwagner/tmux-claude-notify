## Context

No code exists yet. This design establishes the architecture before any script is written. The core challenge is bridging two independent systems: Claude Code (which fires a `Stop` hook in the shell running `claude`) and tmux (which manages window state). The hook runs inside the same shell session that launched `claude`, so `$TMUX_PANE` is available — but tmux commands must be issued via the tmux socket, not assumed to be on PATH everywhere.

A secondary challenge is the one-shot clear hook: tmux `pane-focus-in` hooks are global, so we must scope the clear action to the specific pane that was notified and then remove the hook to avoid accumulating stale hooks.

## Goals / Non-Goals

**Goals:**
- Define the exact data flow from `claude` Stop event → tmux window highlight → desktop notification → clear on focus
- Document how window identity (`$TMUX_PANE`) maps to the tmux window-level highlight
- Decide how the one-shot `pane-focus-in` hook is registered and de-registered without leaking
- Establish the TPM entry point contract (what it sets up, what the binary handles)
- Produce `architecture.md` at the repo root containing the canonical architecture diagram; this file is linked from `README.md` and must be kept current with every future change
- Produce `DEVELOPMENT.md` at the repo root listing all development items in implementation order; linked from `README.md` and updated as the project evolves
- Produce `Taskfile.yml` at the repo root with tasks for building, testing, and setting up the development environment

**Non-Goals:**
- Support for non-tmux terminals
- Background / fleet-view `claude agents` sessions
- Multi-session routing (notifications stay within the session where `claude` ran)

## Decisions

### D1: Single Go binary owns the entire hook path — no bash scripts

**Decision**: All logic lives in `bin/claude-notify`. The Claude Code `Stop` hook calls `claude-notify notify`; the `pane-focus-in` clear hook calls `claude-notify clear --pane <id>`. There are no `notify.sh` or `clear.sh` bash scripts.  
**Rationale**: The binary already owns notification log storage. Routing the hook path through it too means one place for all logic, consistent error handling, and no bash/Go split to maintain. The dashboard (default invocation, no verb) also runs from the same binary.  
**Build**: The TPM entry point compiles the binary on first load (or when source is newer) using `go build`. Build failure is surfaced via `tmux display-message` so it is visible at startup.

### D2: Window highlight via `window-status-style` option on the specific window

**Decision**: Use `tmux set-option -t <window> window-status-style fg=#AD8EE6,bold` to highlight, and `set-option -u` to unset (revert to inherit).  
**Rationale**: Setting per-window style is the cleanest tmux primitive — it doesn't require rewriting `status-left/right` or tracking global state. `set-option -u` is a true clear, not a reset to a hardcoded default.  
**Alternative considered**: Appending a marker to `window-status-format`. Rejected: fragile, requires knowing the current format string.

### D3: One-shot pane-focus-in hook calls the binary's clear subcommand

**Decision**: `claude-notify notify` registers the clear hook as `set-hook -t <session> pane-focus-in[<index>] "run-shell 'claude-notify clear --pane <id>'"` using the pane numeric ID as the array index. `claude-notify clear` clears the highlight, updates the log, and calls `unset-hook` to deregister itself.  
**Rationale**: tmux hooks support array-style named indices, allowing us to add and remove a specific hook entry without clobbering unrelated hooks. Using the binary for the clear path keeps all state mutation in one place.  
**Alternative considered**: A global `pane-focus-in` hook that checks a tmux variable. Rejected: harder to deregister cleanly; accumulates stale checks.

### D4: Pane → Window mapping inside `claude-notify notify`

**Decision**: The binary derives the window ID from `$TMUX_PANE` using `tmux display-message -t $TMUX_PANE -p '#{window_id}'`.  
**Rationale**: `$TMUX_PANE` is a pane ID (e.g. `%34`). Window highlighting targets the window, not the pane. One `display-message` call gives us the stable `@N` window ID. If `$TMUX` or `$TMUX_PANE` is unset the binary exits 0 silently — it is not in a tmux session.

### D5: TPM entry point compiles the binary and binds a configurable keybinding

**Decision**: `tmux-claude-notify.tmux` compiles `bin/claude-notify` if the binary is missing or stale, then reads the TPM option `@claude-notify-key` (default: `C-M-p`) and binds it to `run-shell "bin/claude-notify"`. If `go build` fails, the entry point displays an error via `tmux display-message -d 0` and exits non-zero so the failure is visible at tmux startup.  
**Rationale**: Hard-coding `C-M-p` in the plugin forces every user to patch their config if there's a conflict. A TPM option (`set -g @claude-notify-key 'C-M-p'` in `tmux.conf`) is the standard TPM convention for user-overridable bindings and keeps the default sensible. Surfacing build errors at startup prevents silent failures where the keybinding is never registered and the user has no idea why.

### D6: Notification log as append-only JSON Lines file

**Decision**: `notify.sh` appends a JSON record to `~/.local/share/tmux-claude-notify/notifications.jsonl` on each Stop event. The Go binary reads this file for the picker.  
**Rationale**: JSONL is easy to append from bash (`echo '{"ts":...}' >> file`) and easy to parse in Go. Append-only means no locking needed from the shell side. The Go binary handles deduplication and filters to panes that are still alive via `tmux list-panes -a`.  
**Schema per record**: `{"ts": <unix_ns>, "pane": "%34", "window": "@5", "window_name": "api", "session": "main", "cleared": false}`

### D7a: Binary auto-configures settings.json when hook is missing

**Decision**: When `claude-notify` detects the Stop hook is absent, it writes it into `~/.claude/settings.json` automatically (creating the file if needed), then reports what it did. Malformed JSON is never overwritten — only a parse error is shown.
**Rationale**: The previous decision (read-only check) was reversed: the friction of manual setup outweighs the risk of touching a user file. The binary already knows its own path via `os.Executable()`, so it can write the correct command without any user input. Idempotent — no duplicate entries added.

### D7b: Dashboard is a bubbletea TUI — no fzf dependency

**Decision**: `claude-notify` (default, no verb) renders an interactive list via bubbletea/lipgloss. It reads the notification log, filters to live panes via `tmux list-panes -a`, sorts by `ts` descending, and lets the user select an entry to switch to that window.  
**Rationale**: Since the binary owns the entire hook path and storage, it makes sense for it to own the UI too. A bubbletea TUI eliminates the fzf runtime dependency, gives full control over layout and colors (matching the `#AD8EE6` accent), and keeps the plugin self-contained.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  Shell session (the pane running `claude`)                       │
│                                                                  │
│  claude process ──Stop──► ~/.claude/settings.json hook          │
│                            └─► scripts/notify.sh                │
│                                  │                               │
│                                  │ $TMUX_PANE (inherited env)   │
│                                  ▼                               │
│                         tmux display-message                     │
│                         → window_id (@N)                         │
│                                  │                               │
│                    ┌─────────────┴──────────────┐               │
│                    ▼                             ▼               │
│           tmux set-option                  notify-send           │
│           window-status-style              (desktop toast)       │
│           fg=#AD8EE6,bold                                        │
│                    │                                             │
│                    ▼                                             │
│           tmux set-hook                                          │
│           pane-focus-in[N]                                       │
│           → scripts/clear.sh                                     │
│                    │                                             │
│  ◄─── user focuses the pane ───────────────────────────────────► │
│                    │                                             │
│                    ▼                                             │
│           scripts/clear.sh                                       │
│           tmux set-option -u window-status-style                 │
│           tmux unset-hook pane-focus-in[N]                       │
└─────────────────────────────────────────────────────────────────┘

TPM load time (tmux startup):
  tmux-claude-notify.tmux
  └─► tmux bind-key C-M-p  → jump to last-notified window
```

## Risks / Trade-offs

- **`$TMUX_PANE` not set** → `notify.sh` must guard: if unset, exit 0 silently. Risk of silently doing nothing outside tmux. Mitigation: check `$TMUX` first (confirms tmux context), then `$TMUX_PANE`.
- **Hook index collision** → if many panes notify simultaneously, hook indices could collide. Mitigation: use pane numeric ID (strip `%`) as the hook index — unique per pane.
- **`notify-send` unavailable** → guard with `command -v notify-send || true`. Silent skip on systems without it.
- **Window closed before clear** → `set-option -u` on a dead window will error. Mitigation: wrap in `tmux if-shell` or ignore non-zero exit in clear.sh.
- **Multiple sessions** → `window_id` is session-scoped. If the same window ID exists in two sessions, we target by `session:window`. Use full `-t $TMUX_PANE` target throughout to stay session-scoped.

## Open Questions

- Should `C-M-p` cycle through all waiting windows or jump to the single most-recent one? Start with most-recent (simplest); revisit if multi-window notification is needed.
- Should the plugin store waiting-window state in a tmux global option (e.g. `@claude_notify_last_pane`) so `C-M-p` can find it after a tmux restart? Likely yes — decision deferred to tasks.
