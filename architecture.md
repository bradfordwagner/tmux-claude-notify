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
│                                           + SetPopStyle(paneID)      │
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
  ├─ setup.Check()           → reads ~/.claude/settings.json, auto-configures if missing
  ├─ sessions.Compact(90d)   → remove old unpinned inactive session records from sessions.jsonl
  ├─ store.ReadAll()         → parse notifications.jsonl, sort by ts desc
  ├─ watcher.Reconcile()     → scan active transcripts, correct stale JSONL entries
  │   ├─ sessions.Upsert     → record real project path + pane IDs for every active pane
  │   └─ clearInactivePaneSessions → clear pane fields for sessions whose pane is gone
  ├─ tmux list-panes -a      → build live-pane set
  ├─ sessions.ReadAll()      → merge active + pinned sessions into Notifications view
  └─ bubbletea TUI  (fsnotify auto-refresh; transcript watcher; Tab to switch views)
      ├─ Notifications view  (default): notifications + active sessions + pinned idle sessions
      │   │   columns: STATUS | PIN | P (pop indicator ● when pane-local bg pop active) | WINDOW | PATH | SESSION | AGE
      │   ├─ enter: clear notification + SelectWindow + DetachIfShpell
      │   ├─ p: pin/unpin session via sessions.SetPinned
      │   └─ tab: switch to Sessions view
      ├─ Sessions view: all sessions grouped by project (📌 Pinned group first, then alpha)
      │   ├─ s: cycle within-group sort (age ↔ status)
      │   ├─ f: toggle filter to active panes only (pinned always shown)
      │   ├─ p: pin/unpin session via sessions.SetPinned
      │   ├─ enter/w (active pane): SelectPane + SelectWindow + DetachIfShpell
      │   ├─ enter/w (closed session): sessions.RecoverPath → tmux neww -c <path> -t <outer> -- claude --resume <uuid>
      │   ├─ h/v (closed session): split-window -h/-v -c <path> -t <outer> -- claude --resume <uuid>
      │   ├─ w/h/v (L1 project row): tmux neww/split-window -c <path> -t <outer> -- claude  (fresh session, -n <leaf> for neww)
      │   └─ tab: switch to Notifications view
      ├─ transcript watcher: watches ~/.claude/projects/**/*.jsonl via fsnotify
      │   ├─ state change → running/waiting/stale → UpdateStatus or Append
      │   └─ user message  → ClearPane + ClearWindowStyle (user responded)
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
      │     yes ──► re-apply window-status-style + set-option -p window-style (pane pop)
      │             (idempotent; no new JSONL entry)
      │
      └─── no ──► first notification for this pane
                  │
                  ├──► store.Append               notifications.jsonl
                  │    {ts, pane, window, window_name, session, status:"waiting"}
                  │
                  ├──► tmux window-status-style        (tab: #AD8EE6,bold)
                  ├──► tmux window-status-current-style (active tab: same)
                  ├──► set-option -p window-style bg=<color>  (pane pop bg, pane-scoped)
                  ├──► notify-send                      (desktop toast, optional)
                  │
                  └──► auto-reset subprocess (when delaySecs > 0 OR navDelaySecs > 0)
                       │  reads @claude-notify-active-reset-seconds (default 15; 0=off)
                       │  reads @claude-notify-nav-clear-seconds    (default  2; 0=off)
                       │
                       └─► forkAutoReset → detached subprocess (Setsid, closed stdio)
                                │  args: --pane <id> --delay <activeResetSecs> --nav-delay <navClearSecs>
                                │
                                ├─ IsPaneFocused? (window_active=1 AND pane_active=1)
                                │     yes, delaySecs > 0 → sleep delaySecs → check IsShpellOpen + HasUnclearedPane → runClear
                                │     yes, delaySecs = 0 → exit (already-focused clear disabled)
                                │
                                └─ not focused (different pane or different window)
                                      poll every 2s (up to 4h):
                                      ├─ HasUnclearedPane? no → exit (cleared by other means)
                                      ├─ IsShpellOpen? yes → continue (dashboard handles it)
                                      └─ IsPaneFocused? (window_active=1 AND pane_active=1)
                                            yes, navDelaySecs > 0 → sleep navDelaySecs → runClear
                                            yes, navDelaySecs = 0 → exit (nav-clear disabled)

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
 UI enter handler → runClear(paneID)
      ├──► store.WindowForPane        (window ID from JSONL; works even when pane is gone)
      ├──► store.ClearPane            (marked cleared:true in JSONL)
      ├──► store.UnclearedForWindow   (check remaining notifications for this window)
      │       └─ if none remain ──► tmux window-status-style unset
      │                          ──► tmux window-status-current-style unset
      │                          ──► set-option -p -u window-style  (clears pane pop, pane-scoped)
      │       └─ if siblings remain → window styles preserved
      ├──► UnregisterClearHook
      ├──► SelectPane                 (makes the notified pane active in its window)
      ├──► SelectWindow               (outer session switches to that window)
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
  ├─ listClaudePanes()               → tmux list-panes, filter pane_current_command prefix "claude*"
  ├─ claude.EncodeProjectPath()      → replace "/" and "." with "-" to match ~/.claude/projects/<dir>
  ├─ claude.LatestTranscriptPath()   → most recent .jsonl in project dir, within TranscriptMaxAge
  ├─ tailTranscript()                → read last 20 lines of transcript JSONL
  ├─ deriveStatus()                  → walk events newest-first, return (Status, clear bool)
  ├─ staleDuration()                 → read @claude-notify-stale-minutes option, default 5m
  └─ Reconcile()                     → scan all tracked panes, return current StateChange slice
```

Pane correlation: given a pane's `pane_current_path`, encode it and look up
`~/.claude/projects/<encoded>/` — forward encoding is unambiguous (no decode needed).

## File Layout

```
tmux-claude-notify.tmux       TPM entry point (bash, thin)
bin/claude-notify              compiled binary (gitignored)
cmd/claude-notify/main.go      binary entry point + subcommand routing
internal/
  store/store.go               notifications.jsonl: Append, ReadAll, ClearPane, HasUnclearedPane, UpdateStatus, WindowForPane, UnclearedForWindow
  sessions/sessions.go         sessions.jsonl: SessionRecord, Upsert, ReadAll, SetPinned, Compact, DiscoverAll, RecoverPath
  claude/transcript.go         shared: EncodeProjectPath, LatestTranscriptPath/ID, TranscriptMaxAge (@claude-notify-transcript-age-days, default 14d)
  resurrect/resurrect.go       resurrect.json: Save (snapshot claude panes), Restore (replay --resume into panes)
  tmux/tmux.go                 tmux command helpers (IsPanePopped: show-options -p window-style → pop indicator)
  setup/setup.go               ~/.claude/settings.json hook check + auto-configure
  watcher/watcher.go           transcript file watcher: state derivation, pane correlation, sessions.Upsert on discovery
  ui/model.go                  bubbletea dashboard TUI: Notifications + Sessions views, tab toggle, sort/filter/pin/resume
Taskfile.yml                   build / dev / test / lint / setup tasks
architecture.md                this file (update with every change)
DEVELOPMENT.md                 ordered development items
```

## Data Flow: Session Index

```
 Watcher discovers active claude pane
      │
      ▼
 sessions.Upsert(SessionRecord)      ~/.local/share/tmux-claude-notify/sessions.jsonl
   {session_id, encoded_path, project_path (real), pane_id, window_id,
    window_name, tmux_session, last_activity, status}
      │
      ├─ Pinned flag always preserved (never overwritten by watcher)
      └─ On Reconcile: clearInactivePaneSessions clears pane_id/window fields
         for sessions whose pane is no longer in tmux list-panes

 Dashboard opens → Sessions view activated
      │
      ├─ sessions.ReadAll()      → known sessions from sessions.jsonl (sorted: pinned first, then newest)
      ├─ sessions.DiscoverAll()  → filesystem scan ~/.claude/projects/*/*.jsonl
      │   └─ merge: new UUIDs not yet in sessions.jsonl added to view
      └─ sessions.RecoverPath(encodedPath, stored)
          1. return stored project_path if non-empty (exact, from active-pane discovery)
          2. BFS filesystem walk: split encoded on "-", try combining segments as dir components
          3. fallback: replace leading "-" with "/" and treat all "-" as "/"

 User pins session (p key)
      └─► sessions.SetPinned(sessionID, pinned)  → atomic JSONL rewrite

 User opens/resumes session (w/h/v in Sessions view)
      ├─ L1 project row: OuterSession() → tmux neww/split-window -c <path> -t <outer> -- claude  (fresh)
      └─ L2 closed session: OuterSession() → tmux neww/split-window -c <path> -t <outer> -- claude --resume <session_id>
          └─► DetachIfShpell (close popup)
```

## Data Flow: Resurrect

```
 claude-notify resurrect save
      │
      ├─ tmux list-panes -a → filter pane_current_command starts with "claude*"
      │
      ├─ sessions.ReadAll() → build map[pane_id → SessionRecord]
      │
      └─ for each claude pane:
           ├─ pane_id in sessions map?
           │     YES → session_id + project_path from SessionRecord
           │     NO  → latestTranscriptID(pane_current_path) → scan ~/.claude/projects/<encoded>/
           │
           ├─ no session_id derivable → skip pane silently
           │
           └─ append ResurrectPane{tmux_session, window_index, pane_index,
                                   pane_id, session_id, project_path}
      │
      └─► write ResurrectState (JSON) → ~/.local/share/tmux-claude-notify/resurrect.json
          (atomic temp-file + rename)

 claude-notify resurrect restore
      │
      ├─ Load() resurrect.json → empty/missing → exit 0 (no-op)
      │
      ├─ tmux list-panes -a → build map[(session, window_idx, pane_idx) → live pane]
      │
      └─ for each saved entry:
           ├─ no matching live pane → skip silently
           ├─ live pane_current_command starts with "claude" → skip (idempotent)
           ├─ pane_current_path == project_path?
           │     YES → send "claude --resume <session_id>"
           │     NO  → send "cd <project_path> && claude --resume <session_id>"
           └─► tmux send-keys -t <pane_id> <cmd> Enter
```

## Key Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Hook path | Go binary only, no bash scripts | single owner for all logic and state |
| Storage | JSONL append-only | simple bash-writable, Go-readable, no locking |
| TUI | bubbletea + lipgloss | no fzf runtime dependency, full color control |
| Keybinding | TPM option `@claude-notify-key` | user-overridable, standard TPM convention |
| Hook install | auto-write if missing, read-only check otherwise | friction of manual setup outweighs risk of touching user file |
| Pane pop | `set-option -p window-style bg=<color>` (pane-scoped) | targets the specific notified pane without selecting it; `select-pane -P` moves user cursor focus as a side effect in tmux 3.7b |
| Pop color | `@claude-notify-pop-color` → `@tmux-pop-color` → `#1e1e2e` fallback | visible on black terminal; compatible with Catppuccin Mocha |
| Dashboard keybinding | grimoire shpell if present, `display-popup` fallback | grimoire provides native toggle (C-M-p again to close); popup requires explicit q/esc |
| Dashboard selection | stay open until list empty, then DetachIfShpell | allows handling multiple pending notifications in one session |
| Auto-clear hook | none (removed) | pane-focus-in broken in WSL2; after-select-window fires on any tmux command |
| Active-pane auto-reset | detached subprocess spawned when either delay > 0; `@claude-notify-active-reset-seconds` (default 15; 0=off) | clears pop after grace period when user was already in the pane at notify time |
| Navigate-to-clear | polling subprocess uses `@claude-notify-nav-clear-seconds` (default 2; 0=off) for poll-detected focus | entering a popped pane feels instant (2s); separate from grace period so "already watching" and "navigated to" are tuned independently |
| Pane-level focus detection | `IsPaneFocused` checks `#{window_active}#{pane_active}` (both must be 1) | sibling split panes in the same window share `window_active=1`; without `pane_active`, navigating to a sibling pane would erroneously trigger nav-clear on unrelated notified panes |
| Idempotent notify | `HasUnclearedPane` before Append | Stop fires multiple times per skill invocation; only first call creates JSONL entry |
| Multi-pane window clearing | `UnclearedForWindow` after `ClearPane` gates style teardown | clearing one pane must not remove the window highlight while sibling panes still have uncleared notifications |
| Gone-pane visibility | show all uncleared entries; `(gone)` in PATH when pane missing | live-pane filter silently hides valid notifications (e.g. ~/dotfiles session after pane restart) |
| Hook registered | Stop only | fallback for when dashboard is closed; transcript watcher is primary when open |
| Transcript watcher | embedded in dashboard TUI, not daemon | no IPC complexity, lifecycle tied to TUI, TPM entry point unchanged |
| Pane correlation | forward encode `pane_current_path` → lookup project dir | unambiguous; naive decode fails for paths with hyphens/dots |
| Stale threshold | `@claude-notify-stale-minutes` TPM option (default 5) | user-tunable without recompile |
| Session index | separate `sessions.jsonl` (not added to `notifications.jsonl`) | different lifecycle: sessions are permanent records; notifications are ephemeral; commingling breaks HasUnclearedPane idempotency |
| Path recovery | three-tier (stored → BFS walk → naive decode) | forward encoding is lossy (`/` and `.` → `-`); stored path (from active-pane discovery) is exact; BFS resolves most real-world ambiguity; naive decode is last resort |
| Session discovery | `DiscoverAll()` only on Sessions view activation, not on every state change | filesystem scan of all ~/.claude/projects can be slow with many sessions; known sessions reload from JSONL on every state change (fast) |
| Session view filter | active-pane filter always shows pinned sessions | pinned = "always visible"; hiding pinned behind filter defeats the purpose |
