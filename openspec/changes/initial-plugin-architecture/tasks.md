## 1. Repo Scaffolding

- [x] 1.1 Initialize Go module (`go mod init github.com/bradfordwagner/tmux-claude-notify`)
- [x] 1.2 Create `cmd/claude-notify/main.go` entry point with subcommand routing (`notify`, `clear`, default dashboard)
- [x] 1.3 Add Go dependencies: `bubbletea`, `lipgloss` for TUI
- [x] 1.4 Create `Taskfile.yml` with tasks: `build`, `dev` (build + install to bin/), `test`, `lint`, `setup` (checks Go/tmux on PATH)
- [x] 1.5 Create `README.md` with plugin overview, installation steps, TPM snippet, and links to `architecture.md` and `DEVELOPMENT.md`

## 2. Notification Log

- [x] 2.1 Define the JSONL record struct: `ts`, `pane`, `window`, `window_name`, `session`, `cleared`
- [x] 2.2 Implement log append — create `~/.local/share/tmux-claude-notify/` if missing, write record atomically
- [x] 2.3 Implement log read — parse JSONL, return records sorted descending by `ts`
- [x] 2.4 Implement clear-by-pane — mark the most recent uncleared record for a pane as `cleared: true`

## 3. notify subcommand

- [x] 3.1 Guard: exit 0 silently if `$TMUX` or `$TMUX_PANE` unset
- [x] 3.2 Resolve window ID from `$TMUX_PANE` via `tmux display-message -t $TMUX_PANE -p '#{window_id}'`
- [x] 3.3 Set `window-status-style fg=#AD8EE6,bold` on the target window
- [x] 3.4 Register `pane-focus-in[<pane-numeric-id>]` hook calling `claude-notify clear --pane <id>`
- [x] 3.5 Call `notify-send` if available (`command -v` check); include window name in body
- [x] 3.6 Append record to notification log
- [x] 3.7 Set `window-active-style bg=<@tmux-pop-color>` on the target window; default to `black` if option unset; errors non-fatal

## 4. clear subcommand

- [x] 4.1 Accept `--pane <id>` flag
- [x] 4.2 Unset `window-status-style` on the window containing the pane (`set-option -u`); handle missing window gracefully
- [x] 4.3 Unregister `pane-focus-in[<pane-numeric-id>]` hook
- [x] 4.4 Mark record cleared in notification log
- [x] 4.5 Unset `window-active-style` on the window (`set-option -u`); same graceful error handling as 4.2

## 5. hook-setup check

- [x] 5.1 Implement `~/.claude/settings.json` reader — parse JSON, locate `hooks.Stop` array
- [x] 5.2 Check whether any Stop hook command contains `claude-notify notify`; return configured/not-configured/unknown status
- [x] 5.3 Handle missing file and malformed JSON as distinct states with appropriate messages

## 6. Dashboard TUI

- [x] 6.1 Implement bubbletea model: load log, cross-reference `tmux list-panes -a`, filter to uncleared live-pane entries
- [x] 6.2 Render list with `#AD8EE6` accent for selected item; show pane, window name, session, and relative timestamp
- [x] 6.3 Implement empty state message when no pending notifications
- [x] 6.4 On selection: `SelectWindow` (session-level), mark cleared in log; if entries remain stay open; when list empties call `DetachIfShpell` then `tea.Quit`
- [x] 6.5 Show hook setup status indicator at top of dashboard (configured / not configured / unknown)
- [x] 6.6 Wire default invocation (no subcommand) to open dashboard

## 7. TPM Entry Point

- [x] 7.1 Create `tmux-claude-notify.tmux` — detect stale/missing binary and run `go build -o bin/claude-notify ./cmd/claude-notify`
- [x] 7.2 On build failure: `tmux display-message -d 0 "tmux-claude-notify: build failed — check Go install"` and exit non-zero
- [x] 7.3 Read `@claude-notify-key` TPM option, default to `C-M-p`
- [x] 7.4 If `~/.tmux/plugins/tmux-grimoire/bin/custom_shpell` is executable, bind key via `custom_shpell standard claude-notify '<binary>'`; else bind to `popup -E -w 80% -h 80% '<binary>'`

## 8. Documentation Artifacts

- [x] 8.1 Create `architecture.md` at repo root with the full architecture diagram from `design.md` (data flow, component boundaries, hook wiring)
- [x] 8.2 Create `DEVELOPMENT.md` at repo root listing all development items in implementation order, referencing this task list
- [x] 8.3 Link `architecture.md` and `DEVELOPMENT.md` from `README.md`
- [x] 8.4 Update `CLAUDE.md` `## Plugin structure` section to reflect single-binary layout (no `scripts/` dir)
