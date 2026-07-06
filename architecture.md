# Architecture

## Component Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│  Shell session (the pane running `claude`)                           │
│                                                                      │
│  claude process ──Stop──► ~/.claude/settings.json hook              │
│                            └─► bin/claude-notify notify             │
│                                         │                            │
│                                         │  $TMUX_PANE (inherited)   │
│                                         ▼                            │
│                                tmux display-message                  │
│                                → window_id (@N), window_name, sess  │
│                                         │                            │
│                              ┌──────────┴──────────┐                │
│                              ▼                      ▼                │
│                   HasUnclearedPane?           (first call only)      │
│                        yes │                       │ no              │
│                            ▼                       ▼                 │
│                   re-apply styles          store.Append              │
│                   (idempotent return)      notifications.jsonl       │
│                                           + SetWindowStyle           │
│                                           + SetPopStyle              │
│                                           + notify-send (optional)   │
└─────────────────────────────────────────────────────────────────────┘

TPM load time (tmux startup):
  tmux-claude-notify.tmux
  ├─ go build -o bin/claude-notify ./cmd/claude-notify  (if stale)
  │   └─ on failure: tmux display-message error, exit 1
  └─ tmux bind-key <@claude-notify-key|C-M-p>
      ├─ if ~/.tmux/plugins/tmux-grimoire/bin/custom_shpell exists:
      │   └─ run-shell "custom_shpell standard claude-notify 'bin/claude-notify'"
      └─ else: popup -E -w 80% -h 80% "bin/claude-notify"

User invokes keybinding (C-M-p by default):
  bin/claude-notify  (no args → dashboard)
  ├─ setup.Check()        → reads ~/.claude/settings.json, auto-configures if missing
  ├─ store.ReadAll()      → parse notifications.jsonl, sort by ts desc
  ├─ watcher.Reconcile()  → scan active transcripts, correct stale JSONL entries
  ├─ tmux list-panes -a   → filter to live panes only
  └─ bubbletea TUI  (fsnotify auto-refresh on JSONL/settings; transcript watcher for live state)
      ├─ transcript watcher: watches ~/.claude/projects/**/*.jsonl via fsnotify
      │   ├─ state change → running/waiting/stale → UpdateStatus or Append
      │   └─ user message  → ClearPane + ClearWindowStyle (user responded)
      ├─ select entry → ClearWindowStyle + ClearPopStyle + store.ClearPane
      │                  + SelectWindow (session-level) + DetachIfShpell
      └─ q/esc → close transcript watcher + DetachIfShpell → tea.Quit
```

## Data Flow: Notification Lifecycle

```
 Stop hook fires (fallback — always active)
      │
      ▼
 claude-notify notify
      │
      ├─ HasUnclearedPane(paneID)?
      │     yes ──► re-apply window-status-style + window-active-style
      │             (idempotent; no new JSONL entry)
      │
      └─── no ──► first notification for this pane
                  │
                  ├──► store.Append               notifications.jsonl
                  │    {ts, pane, window, window_name, session, status:"waiting"}
                  │
                  ├──► tmux window-status-style        (tab: #AD8EE6,bold)
                  ├──► tmux window-status-current-style (active tab: same)
                  ├──► tmux window-active-style         (pane pop bg)
                  └──► notify-send                      (desktop toast, optional)

 Transcript watcher fires (while dashboard is open)
      │
      ├─ transcript event: assistant + tool_use ──► status:"running"
      │   └─► store.UpdateStatus only (no window highlight)
      │
      ├─ transcript event: assistant + end_turn ──► status:"waiting"
      │   ├─► store.Append (if no entry) or store.UpdateStatus
      │   └─► tmuxclient.SetWindowStyle
      │
      ├─ transcript event: user + tool_result ──► status:"running"
      │   └─► store.UpdateStatus only
      │
      ├─ transcript event: user + text ──► user responded → clear
      │   ├─► store.ClearPane
      │   └─► tmuxclient.ClearWindowStyle + ClearPopStyle
      │
      └─ no activity for @claude-notify-stale-minutes (default 5) ──► status:"stale"
          └─► store.UpdateStatus only

 Dashboard open — reconciliation
      │
      ├─ watcher.Reconcile() scans all active transcripts
      ├─ JSONL entries whose pane already responded → store.ClearPane
      └─ JSONL entries still waiting → store.UpdateStatus confirmed

 User selects entry from dashboard:
      │
      ▼
 UI enter handler
      ├──► store.ClearPane            (marked cleared:true in JSONL)
      ├──► tmux window-status-style unset
      ├──► tmux window-status-current-style unset
      ├──► tmux window-active-style unset   (clears pane pop)
      ├──► UnregisterClearHook
      ├──► SelectWindow               (outer session jumps to that window)
      └──► DetachIfShpell             (closes grimoire popup)
```

## Hook Configuration

One hook in `~/.claude/settings.json` calls `claude-notify notify`:

| Hook | When it fires |
|---|---|
| `Stop` | Claude finishes its response turn and returns to the user prompt |

The Stop hook is a fallback: it writes to JSONL when the dashboard is closed. While
the dashboard is open, the transcript watcher provides richer real-time state.

The `hookEvents` slice in `setup.go` is the single source of truth for which
events are registered — adding a name there includes it in both check and
configure paths.

## Transcript Watcher

The watcher runs embedded in the dashboard TUI process (not a daemon):

```
internal/watcher/watcher.go
  ├─ listClaudePanes()     → tmux list-panes, filter pane_current_command prefix "claude*"
  ├─ encodeProjectPath()   → replace "/" and "." with "-" to match ~/.claude/projects/<dir>
  ├─ latestTranscript()    → most recent .jsonl in project dir, modified < 24h ago
  ├─ tailTranscript()      → read last 20 lines of transcript JSONL
  ├─ deriveStatus()        → walk events newest-first, return (Status, clear bool)
  ├─ staleDuration()       → read @claude-notify-stale-minutes option, default 5m
  └─ Reconcile()           → scan all tracked panes, return current StateChange slice
```

Pane correlation: given a pane's `pane_current_path`, encode it and look up
`~/.claude/projects/<encoded>/` — forward encoding is unambiguous (no decode needed).

## File Layout

```
tmux-claude-notify.tmux       TPM entry point (bash, thin)
bin/claude-notify              compiled binary (gitignored)
cmd/claude-notify/main.go      binary entry point + subcommand routing
internal/
  store/store.go               JSONL log: Append, ReadAll, ClearPane, HasUnclearedPane, UpdateStatus
  tmux/tmux.go                 tmux command helpers
  setup/setup.go               ~/.claude/settings.json hook check + auto-configure
  watcher/watcher.go           transcript file watcher: state derivation, pane correlation
  ui/model.go                  bubbletea dashboard TUI (embeds transcript watcher)
Taskfile.yml                   build / dev / test / lint / setup tasks
architecture.md                this file (update with every change)
DEVELOPMENT.md                 ordered development items
```

## Key Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Hook path | Go binary only, no bash scripts | single owner for all logic and state |
| Storage | JSONL append-only | simple bash-writable, Go-readable, no locking |
| TUI | bubbletea + lipgloss | no fzf runtime dependency, full color control |
| Keybinding | TPM option `@claude-notify-key` | user-overridable, standard TPM convention |
| Hook install | auto-write if missing, read-only check otherwise | friction of manual setup outweighs risk of touching user file |
| Pane pop | persistent `window-active-style bg=<color>` | stays until cleared — no timer, no goroutine; cosmetic only so errors ignored |
| Pop color | `@claude-notify-pop-color` → `@tmux-pop-color` → `#1e1e2e` fallback | visible on black terminal; compatible with Catppuccin Mocha |
| Dashboard keybinding | grimoire shpell if present, `display-popup` fallback | grimoire provides native toggle (C-M-p again to close); popup requires explicit q/esc |
| Dashboard selection | stay open until list empty, then DetachIfShpell | allows handling multiple pending notifications in one session |
| Auto-clear hook | none (removed) | pane-focus-in broken in WSL2; after-select-window fires on any tmux command |
| Idempotent notify | `HasUnclearedPane` before Append | Stop fires multiple times per skill invocation; only first call creates JSONL entry |
| Hook registered | Stop only | fallback for when dashboard is closed; transcript watcher is primary when open |
| Transcript watcher | embedded in dashboard TUI, not daemon | no IPC complexity, lifecycle tied to TUI, TPM entry point unchanged |
| Pane correlation | forward encode `pane_current_path` → lookup project dir | unambiguous; naive decode fails for paths with hyphens/dots |
| Stale threshold | `@claude-notify-stale-minutes` TPM option (default 5) | user-tunable without recompile |
