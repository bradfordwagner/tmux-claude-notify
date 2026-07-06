## 1. tmux Helper

- [x] 1.1 Add `IsShpellOpen() bool` to `internal/tmux/tmux.go` — runs `tmux has-session -t _shpell-session` and returns `true` when it exits 0

## 2. Auto-Reset Guard

- [x] 2.1 In `cmd/claude-notify/main.go` `runAutoReset`, add popup check after the sleep: call `tmuxclient.IsShpellOpen()` and return early (skip clear) if it returns `true`

## 3. Spec Sync

- [x] 3.1 Update `openspec/specs/auto-reset-active-pane/spec.md` to add the popup-open precondition to the "Auto-reset subprocess clears notification after delay" requirement (run `/opsx:sync` or manually merge)

## 4. Docs

- [x] 4.1 Update `architecture.md` to note the popup-open guard in the auto-reset flow
