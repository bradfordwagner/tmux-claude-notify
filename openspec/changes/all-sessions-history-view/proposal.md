## Why

The dashboard currently only surfaces sessions that have fired the Stop hook — it is blind to sessions that were never in the current tmux environment, and to active sessions with no pending notification. Users want a single place to browse all historical Claude sessions across every project, jump back into a closed one, and keep important sessions pinned regardless of notification state.

## What Changes

- New persistent session index (`sessions.jsonl`) records every Claude session discovered, keyed by UUID, storing the real project path and last-activity timestamp.
- Dashboard gains a second tab ("Sessions") reachable with `Tab`, listing all discovered sessions sorted by last activity by default; sort cycles with `s`.
- Sessions can be pinned (`p`); pinned sessions float to the top of both the Notifications and Sessions views.
- Closed (non-active) sessions can be resumed from the Sessions view: pressing `w` opens a new tmux window in the original project directory running `claude --resume <session-id>`.
- Active (pane open) and idle (pane open, no pending Stop-hook notification) sessions now appear in the Notifications view alongside waiting/running ones.
- The transcript watcher records the real `pane_current_path` into the session index whenever it discovers an active pane, eliminating path ambiguity for those sessions.
- For sessions never seen with an active pane, path recovery is attempted by walking the filesystem to find a directory whose encoding matches the project directory name.

## Capabilities

### New Capabilities

- `session-index`: Persistent sessions.jsonl store and filesystem-scan discovery engine; records session UUID, real project path, pinned flag, and last-activity timestamp across all projects.
- `sessions-view`: New "Sessions" tab in the dashboard TUI toggled with `Tab`; shows all discovered sessions with sort-by-field (`s`), pin toggle (`p`), and open/resume action (`w`/`h`/`v`).
- `session-resume`: Actions to open/resume sessions from the Sessions view: `w` (new window), `h` (split-h), `v` (split-v). All target the outer non-shpell session. L1 opens a fresh claude session; L2 uses `--resume <session-id>`.

### Modified Capabilities

- `transcript-watcher`: Must write real `pane_current_path` to session-index on active pane discovery; must emit active and idle panes to the Notifications view even without a Stop-hook entry.
- `notification-log`: Active and idle sessions (discovered via transcript watcher) now appear as rows in the Notifications view, not only Stop-hook-triggered entries.
- `dashboard-row-layout`: Gains a tab-toggle header (Notifications | Sessions), a Pinned column indicator, and the Sessions view column set (STATUS / PROJECT / PATH / SESSION-ID / AGE).

## Impact

- New file: `internal/sessions/` package (index store + discovery + path recovery)
- Modified: `internal/watcher/watcher.go` — populate session-index, emit active/idle rows
- Modified: `internal/ui/model.go` — tab state, Sessions view rendering, sort/pin/resume keybindings
- Modified: `internal/store/store.go` — surface pinned flag to Notifications view ordering
- New persistent file: `~/.local/share/tmux-claude-notify/sessions.jsonl`
- No changes to Stop hook, TPM entry point, or external hook schema
