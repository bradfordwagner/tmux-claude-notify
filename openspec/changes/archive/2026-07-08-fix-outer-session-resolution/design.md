## Context

```go
// OuterSession returns the name of the first tmux session that is not _shpell-session.
// Used to target split-window commands at the user's real session when the dashboard
// is running inside the grimoire popup.
func OuterSession() string {
	out, err := run("list-sessions", "-F", "#{session_name}")
	if err != nil {
		return ""
	}
	for _, name := range strings.Split(out, "\n") {
		if name != "_shpell-session" && name != "" {
			return name
		}
	}
	return ""
}
```

`doNewSession` (Sessions L1 `w`/`h`/`v`, brand-new claude window) and `doResume` (Sessions L2 `w`/`h`/`v` on a closed session, resumed claude window) both call this to get a `-t <session>` target for `tmux neww`/`split-window`, so the new window lands in the user's real session rather than `_shpell-session`. The function enumerates *all* tmux sessions server-wide and returns the first one that isn't `_shpell-session` — with more than one real session attached (a common setup: `edit`, `fwd`, `k8s`), it returns whichever sorts/lists first, not necessarily the session the user actually launched the dashboard from. Confirmed: `tmux list-sessions` currently returns `edit`, `fwd`, `k8s` in that order; opening the dashboard from `k8s` and pressing `w` sends the new window to `edit`.

This repo already fixed the equivalent problem for `enter`-driven navigation in the archived `fix-cross-session-jump-notification` change: `outerClientName()` reliably identifies the real outer client (the one not attached to `_shpell-session`, resolved via an explicit `-t <TMUX_PANE>` target rather than ambiguous implicit lookup) and `SwitchOuterClientToSessionWindow` switches that specific client. `OuterSession()` predates that fix and was never updated to use the same client-based approach — it still reasons about *sessions* directly instead of *the client* that matters.

## Goals / Non-Goals

**Goals:**
- `OuterSession()` returns the session the actual outer client (the one that isn't attached to `_shpell-session`) is attached to, not an arbitrary first-in-list session
- Reuse the already-fixed `outerClientName()` rather than duplicating client-resolution logic
- Preserve today's `""` (no target / use default) behavior when there is no distinguishable outer client — e.g. the non-grimoire `display-popup` fallback path (`tmux-claude-notify.tmux`'s `popup -E` binding), where there's only one client and no `_shpell-session` nesting at all

**Non-Goals:**
- Handling more than one legitimate "outer" client (e.g. two real terminals both outside `_shpell-session` simultaneously) — `outerClientName()` already just returns the first match in that case; unchanged, same limitation as the prior fix accepted
- Changing `doNewSession`/`doResume`'s call sites or the `-t` argument construction — they already handle `OuterSession() == ""` by omitting `-t`

## Decisions

### Decision: Implement `OuterSession()` in terms of `outerClientName()`

```go
func OuterSession() string {
	client := outerClientName()
	if client == "" {
		return ""
	}
	session, err := run("display-message", "-p", "-t", client, "#{client_session}")
	if err != nil {
		return ""
	}
	return session
}
```

`display-message -p -t <client-name> "#{client_session}"` resolves the format relative to that specific client, giving its actual attached session directly — no enumeration or ordering assumption involved. When `outerClientName()` finds nothing (single-client, non-popup, or non-grimoire-popup contexts), `OuterSession()` returns `""` exactly as it does today, so callers' existing `if outer != ""` guards need no changes.

Alternative considered: keep enumerating `list-sessions` but cross-reference against `list-clients` to find which session the non-`_shpell-session` client is attached to inline — functionally equivalent to reusing `outerClientName()`, but duplicates logic that already exists and is already tested by the prior fix. Rejected for the reuse.

## Risks / Trade-offs

- [Risk] `outerClientName()`'s single caveat (returns the *first* non-current, non-`_shpell-session` client when more than one exists) now also applies to `OuterSession()` → Mitigation: this is an existing, accepted limitation from the prior fix, not a new one; out of scope per Non-Goals.
- [Risk] Behavior differs subtly from today when there is exactly one outer session but the dashboard is opened via the non-grimoire `popup -E` fallback (no `_shpell-session` at all) → Mitigation: in that path `outerClientName()` already returns `""` (no second client exists), so `OuterSession()` returns `""` just as before — verified by design, no regression.
