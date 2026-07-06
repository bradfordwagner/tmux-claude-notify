# Development

Implementation order follows dependency constraints: scaffolding and storage first, then subcommands that use storage, then the TUI that reads it, then the TPM shell wrapper, then docs.

## 1. Repo Scaffolding

- [x] 1.1 Initialize Go module (`go mod init github.com/bradfordwagner/tmux-claude-notify`)
- [x] 1.2 Create `cmd/claude-notify/main.go` entry point with subcommand routing (`notify`, `clear`, default dashboard)
- [x] 1.3 Add Go dependencies: `bubbletea`, `lipgloss` for TUI
- [x] 1.4 Create `Taskfile.yml` with tasks: `build`, `dev`, `test`, `lint`, `setup`
- [ ] 1.5 Create `README.md`

## 2. Notification Log

- [x] 2.1 Define the JSONL record struct: `ts`, `pane`, `window`, `window_name`, `session`, `cleared`
- [x] 2.2 Implement log append — create `~/.local/share/tmux-claude-notify/` if missing, write record
- [x] 2.3 Implement log read — parse JSONL, return records sorted descending by `ts`
- [x] 2.4 Implement clear-by-pane — mark the most recent uncleared record for a pane

## 3. notify subcommand

- [x] 3.1 Guard: exit 0 silently if `$TMUX` or `$TMUX_PANE` unset
- [x] 3.2 Resolve window ID from `$TMUX_PANE`
- [x] 3.3 Set `window-status-style` and `window-status-current-style` `fg=#AD8EE6,bold` on the target window
- [x] 3.4 ~~Register `pane-focus-in` hook~~ — removed; unreliable in WSL2 and after-select-window fires on any tmux command
- [x] 3.5 Call `notify-send` if available; include window name in body
- [x] 3.6 Append record to notification log (idempotent: skip if uncleared entry already exists for pane)
- [x] 3.7 Set `window-active-style bg=<color>` on the target window (pop color; reads `@claude-notify-pop-color` → `@tmux-pop-color` → `#1e1e2e`)

## 4. clear subcommand

- [x] 4.1 Accept `--pane <id>` flag
- [x] 4.2 Unset `window-status-style` on the window; handle missing window gracefully
- [x] 4.3 Unregister `pane-focus-in` hook
- [x] 4.4 Mark record cleared in notification log
- [x] 4.5 Unset `window-active-style` on the window (clears pane pop)

## 5. Hook setup check

- [x] 5.1 Read `~/.claude/settings.json`, locate `hooks.Stop` and `hooks.PreToolUse` arrays
- [x] 5.2 Detect whether both hooks point at `claude-notify notify`; auto-configure if either is missing
- [x] 5.3 Handle missing file and malformed JSON as distinct states
- [x] 5.4 Dashboard status indicator shows `[Stop,PreToolUse] hooks configured`

## 6. Dashboard TUI

- [x] 6.1 Bubbletea model: load log, cross-reference live panes, filter uncleared
- [x] 6.2 Render list with `#AD8EE6` accent; show window name, session, pane, relative age
- [x] 6.3 Empty state message when no pending notifications
- [x] 6.4 On selection: `SelectWindow` (session-level), mark cleared in log; if more entries remain stay open; auto-quit when list empties
- [x] 6.5 Hook setup status indicator at top of dashboard
- [x] 6.6 Wire default invocation (no subcommand) to dashboard

## 7. TPM Entry Point

- [x] 7.1 `tmux-claude-notify.tmux` — detect stale/missing binary and build
- [x] 7.2 Surface build failure via `tmux display-message`
- [x] 7.3 Read `@claude-notify-key` TPM option, default `C-M-p`
- [x] 7.4 If `~/.tmux/plugins/tmux-grimoire/bin/custom_shpell` exists, bind key via grimoire (toggleable shpell); else fall back to `popup -E -w 80% -h 80%`

## 8. Documentation

- [x] 8.1 `architecture.md` — canonical architecture diagram and design decisions
- [x] 8.2 `DEVELOPMENT.md` — this file
- [ ] 8.3 Link `architecture.md` and `DEVELOPMENT.md` from `README.md`
- [ ] 8.4 Update `CLAUDE.md` plugin structure section
