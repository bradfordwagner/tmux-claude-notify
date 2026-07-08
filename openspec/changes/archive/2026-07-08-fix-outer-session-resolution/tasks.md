## 1. Fix outer session resolution

- [x] 1.1 In `internal/tmux/tmux.go`, rewrite `OuterSession()` to call `outerClientName()` and then resolve that specific client's `#{client_session}` via `display-message -p -t <client> "#{client_session}"`, instead of enumerating `list-sessions` and returning the first non-`_shpell-session` entry
- [x] 1.2 Preserve the `""` return value (no target / caller omits `-t`) when `outerClientName()` finds no distinguishable outer client, so `doNewSession`/`doResume`'s existing `if outer != ""` guards keep working unchanged
- [x] 1.3 Remove the now-unused `list-sessions`-based enumeration from `OuterSession()` (do not leave dead code)

## 2. Verify

- [x] 2.1 `go build ./...` and `go vet ./...` pass
- [x] 2.2 Manually reproduce: with 2+ non-shpell tmux sessions attached (e.g. `edit`, `fwd`, `k8s`), open the dashboard from a session that does NOT sort first in `tmux list-sessions`, press `w` on a project in Sessions L1, confirm the new claude window opens in the session the dashboard was actually opened from — confirmed by user
- [x] 2.3 Manually verify `w` on a closed session at Sessions L2 (resume) also targets the correct outer session under the same multi-session setup — confirmed by user
- [x] 2.4 Manually verify single-outer-session usage (today's common case) and the non-grimoire `popup -E` fallback path are unaffected — confirmed by user

## 3. Close out

- [x] 3.1 Confirm no architecture.md diagram update is needed (internal client/session-resolution fix only, reuses existing `outerClientName()`, no data-flow/component-boundary/hook-wiring change)
- [ ] 3.2 Run `/opsx:archive` once merged
