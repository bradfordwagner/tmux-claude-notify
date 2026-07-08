## 1. Store: OldestUncleared helper

- [x] 1.1 Add `OldestUncleared() (*Record, error)` to `internal/store/store.go` — reads all records, returns the uncleared entry with the smallest `TS` value (nil if none)

## 2. status subcommand

- [x] 2.1 Add `status` case to the subcommand router in `cmd/claude-notify/main.go`
- [x] 2.2 Implement `runStatus()`: call `store.ReadAll()`, count uncleared entries, print `⚡ N` if N > 0 else print nothing, exit 0

## 3. jump subcommand

- [x] 3.1 Add `jump` case to the subcommand router in `cmd/claude-notify/main.go`
- [x] 3.2 Implement `runJump()`: call `store.OldestUncleared()`; if nil, exit 0 silently; else call `runClear(r.Pane)` (reuse existing clear helper)

## 4. TPM entry point wiring

- [x] 4.1 In `tmux-claude-notify.tmux`, read `@claude-notify-jump-key` (default `C-M-P`) and bind it to `run-shell '$PLUGIN_DIR/bin/claude-notify jump'`
- [x] 4.2 In `tmux-claude-notify.tmux`, read `@claude-notify-statusline`; if non-empty, append `set-option -ga status-right " #($PLUGIN_DIR/bin/claude-notify status)"`

## 5. Documentation

- [x] 5.1 Add `@claude-notify-statusline` row to the Configuration table in `README.md` (default: unset/disabled; description: when set, appends `⚡ N` count to `status-right`)
- [x] 5.2 Add `@claude-notify-jump-key` row to the Configuration table in `README.md` (default: `C-M-P`; description: keybinding to jump directly to the oldest waiting pane)
- [x] 5.3 Update `architecture.md` to reflect the new `status` and `jump` subcommands and the conditional `status-right` wiring in the TPM entry point section
