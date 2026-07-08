## 1. tmux helper

- [x] 1.1 Add `SwitchClientToSessionWindow(session, windowID string) error` to `internal/tmux/tmux.go` — runs `switch-client -t <session>:<windowID>`

## 2. Jump fix

- [x] 2.1 In `cmd/claude-notify/main.go` `runJump()`, replace `SwitchToWindow(r.Window)` with `SwitchClientToSessionWindow(r.Session, r.Window)`

## 3. Spec sync

- [x] 3.1 Update `openspec/specs/quickjump/spec.md` with the cross-session scenarios from the delta spec (merge MODIFIED requirements in)
- [x] 3.2 Update `architecture.md` if the jump data-flow diagram references `switch-client` without a session qualifier
