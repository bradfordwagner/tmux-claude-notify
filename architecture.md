# Architecture

## Component Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│  Shell session (the pane running `claude`)                           │
│                                                                      │
│  claude process ──Stop──► ~/.claude/settings.json hook              │
│                            └─► bin/claude-notify notify             │
│                                  │                                   │
│                                  │  $TMUX_PANE (inherited env)      │
│                                  ▼                                   │
│                         tmux display-message                         │
│                         → window_id (@N), window_name, session      │
│                                  │                                   │
│                    ┌─────────────┼──────────────┐                   │
│                    ▼             ▼               ▼                   │
│           tmux set-option   notify-send    store.Append             │
│           window-status-style  (optional)  notifications.jsonl      │
│           fg=#AD8EE6,bold                                            │
│           window-active-style                                        │
│           bg=<@tmux-pop-color>                                       │
│                    │                                                 │
│                    ▼                                                 │
│           tmux set-hook                                              │
│           pane-focus-in[N]                                           │
│           → bin/claude-notify clear --pane %N                       │
│                    │                                                 │
│  ◄─── user focuses the pane ────────────────────────────────────►   │
│                    │                                                 │
│                    ▼                                                 │
│           bin/claude-notify clear --pane %N                         │
│           ├─ tmux set-option -u window-status-style                 │
│           ├─ tmux set-option -u window-active-style                 │
│           ├─ tmux set-hook -u pane-focus-in[N]                      │
│           └─ store.ClearPane (mark cleared in JSONL)                │
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
  ├─ setup.Check()  → reads ~/.claude/settings.json, auto-configures if missing
  ├─ store.ReadAll() → parse notifications.jsonl, sort by ts desc
  ├─ tmux list-panes -a → filter to live panes only
  └─ bubbletea TUI
      ├─ select entry → SelectWindow (session-level) + store.ClearPane
      │   ├─ if more entries remain: stay open
      │   └─ if list now empty: DetachIfShpell → tea.Quit
      └─ q/esc → DetachIfShpell / close popup → tea.Quit
```

## Data Flow: Notification Lifecycle

```
 Stop hook fires
      │
      ▼
 claude-notify notify
      │
      ├──► JSONL record appended   ~/.local/share/tmux-claude-notify/notifications.jsonl
      │    {ts, pane, window, window_name, session, cleared:false}
      │
      ├──► tmux window-status-style set  (tab highlight: #AD8EE6,bold)
      │
      ├──► tmux window-active-style set  (pane pop: bg=@tmux-pop-color)
      │
      ├──► notify-send fired            (desktop toast, if available)
      │
      └──► pane-focus-in hook registered

 User focuses pane OR selects from dashboard (last entry):
      │
      ▼
 claude-notify clear --pane <id>
      │
      ├──► tmux window-status-style unset
      ├──► tmux window-active-style unset
      ├──► pane-focus-in hook deregistered
      └──► JSONL record updated: cleared:true
```

## File Layout

```
tmux-claude-notify.tmux       TPM entry point (bash, thin)
bin/claude-notify              compiled binary (gitignored)
cmd/claude-notify/main.go      binary entry point + subcommand routing
internal/
  store/store.go               JSONL log: Append, ReadAll, ClearPane
  tmux/tmux.go                 tmux command helpers
  setup/setup.go               ~/.claude/settings.json hook check
  ui/model.go                  bubbletea dashboard TUI
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
| Pane pop | persistent `window-active-style bg=@tmux-pop-color` | stays until cleared — no timer, no goroutine; cosmetic only so errors ignored |
| Dashboard keybinding | grimoire shpell if present, `display-popup` fallback | grimoire provides native toggle (C-M-p again to close); popup requires explicit q/esc |
| Dashboard selection | stay open until list empty, then auto-quit | allows handling multiple pending notifications in one session |
