## Why

`OuterSession()` in `internal/tmux/tmux.go` — used by the dashboard's `w`/`h`/`v` keys in the Sessions view (opening a new claude window at L1, or resuming/splitting a closed session at L2) — picks "the outer session" by returning the *first* session in `tmux list-sessions` output that isn't `_shpell-session`. This has nothing to do with which session the user actually invoked the popup from. Confirmed reproduction: with sessions `edit`, `fwd`, `k8s` all attached (in that `list-sessions` order), opening the dashboard from `k8s` and pressing `w` on a project sends the new window to `edit` instead of `k8s`. This is the same class of bug as the one already fixed for `enter`-driven navigation (`outerClientName`/`SwitchOuterClientToSessionWindow`, see the archived `fix-cross-session-jump-notification` change) — ambiguous/wrong resolution of "the outer session" — but in a different function that change didn't touch.

## What Changes

- Replace `OuterSession()`'s "first non-`_shpell-session` session" heuristic with a lookup based on the actual outer client: reuse `outerClientName()` (already fixed to reliably identify the real outer client via an explicit pane target) and query that specific client's `#{client_session}`
- `doNewSession` and `doResume` in `internal/ui/model.go` keep calling `tmuxclient.OuterSession()` unchanged — only its internal resolution logic changes
- No behavior change when only one outer session exists (today's common case, where the bug is unobservable)

## Capabilities

### New Capabilities
<!-- none: this is a bug fix, not a new capability -->

### Modified Capabilities
- `always-on-dashboard`: the "Sessions L1 — `w`/`h`/`v` open a new claude session" and "Sessions L2 — `w`/`h`/`v` navigate to or resume selected session" requirements are clarified so the "outer session" they target is resolved from the actual outer client rather than an arbitrary session-list ordering, matching multiple simultaneously attached tmux sessions correctly.

## Impact

- `internal/tmux/tmux.go` — `OuterSession()`: resolve via the outer client's actual session instead of `list-sessions` order
- No change to `internal/ui/model.go` call sites, no JSONL schema, config, or keybinding changes
- Same-session and single-outer-session usage is unaffected
