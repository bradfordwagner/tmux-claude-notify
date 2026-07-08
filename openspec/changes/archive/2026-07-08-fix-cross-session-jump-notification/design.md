## Context

`internal/tmux/tmux.go` has two functions that resolve "the outer client" (the real terminal client, as opposed to the nested client created inside the grimoire popup) so the dashboard can switch it to a different tmux session when the user jumps to a notification whose pane lives outside the popup's own session:

```go
func outerClientName() string {
	currentClient, err := run("display-message", "-p", "#{client_name}")
	...
	out, err := run("list-clients", "-F", "#{client_name} #{client_session}")
	...
}

func SwitchOuterClientToSessionWindow(session, windowID string) error {
	currentSession, err := run("display-message", "-p", "#{client_session}")
	if err != nil || currentSession != "_shpell-session" {
		return nil
	}
	...
}
```

Both `display-message` calls omit `-t`, so tmux resolves "the current client" implicitly. Every *other* `display-message` call in this file (`WindowID`, `WindowName`, `PanePath`, `Session`, `IsPaneFocused`, etc.) explicitly pins a target (a pane ID). These two are the only exceptions, and they were added together in the commit that introduced cross-session jump support (`6be04ad`).

This project's popup mechanism is `~/.tmux/plugins/tmux-grimoire`'s `custom_shpell` (`scripts/shpell.sh`, installed locally, not part of this repo). Its own source code explicitly documents and works around exactly this ambiguity:

```
# CRITICAL: Do NOT indent the format lines. Leading whitespace becomes part of
# the format and breaks tmux parsing.
target="${TMUX_PANE:-}:"
...
# Use an explicit target (-t "$target") so pane/session formats resolve from
# scripts or hooks.
```

`shpell.sh` never calls `display-message` without `-t`. Our two calls are invoked from a Go binary running as a background process inside a pane of `_shpell-session:cn` (i.e. exactly the "script or hook" context grimoire's own author flags as unreliable without an explicit target). When client resolution is ambiguous — for example when two clients are attached at once (the outer terminal client and the nested popup client created by grimoire's `attach-session` inside the popup) — `display-message -p` without `-t` is not guaranteed to resolve to the client tied to the invoking pane. This explains the reported symptom: cross-session jumps from the dashboard (`enter`/`w` in Notifications or Sessions view) intermittently fail to switch the outer client to the target session, while the popup still closes normally, leaving the user on whatever session/window was already showing. Same-session dashboard navigation and the standalone `claude-notify jump` keybinding are unaffected because they don't call these two functions (`jump` uses `SwitchClientToSessionWindow`, which targets the invoking client implicitly but unambiguously, since `run-shell` keybindings have exactly one relevant client).

## Goals / Non-Goals

**Goals:**
- Make `outerClientName()` and `SwitchOuterClientToSessionWindow()` resolve the invoking pane's client deterministically, using the same explicit-target pattern already used everywhere else in `tmux.go` and already documented as required by the sibling grimoire plugin
- Minimal diff: fix only the ambiguous `display-message` calls
- No change to same-session navigation, the CLI `jump` command, or any user-visible config/keybindings

**Non-Goals:**
- Changing how `outerClientName()` picks *among* multiple non-`_shpell-session` clients if more than one exists (out of scope; unchanged from today)
- Rewriting `custom_shpell`/grimoire (external dependency, not part of this repo)
- Adding tests that spin up a real multi-client tmux server (no existing test seam for `internal/tmux`; these functions shell out directly and are validated manually, consistent with the rest of the file)

## Decisions

### Decision: Pin `-t $TMUX_PANE` on both ambiguous `display-message` calls

`tmuxclient.PaneID()` already reads `$TMUX_PANE` (used elsewhere, e.g. the Stop hook). Passing `-t <paneID>` to `display-message` makes tmux resolve `#{client_name}` / `#{client_session}` relative to the pane the dashboard process is actually running in, removing the ambiguity — this is exactly the fix grimoire's own script applies to itself.

Alternative considered: pass `-t` as `$TMUX_PANE + ":"` (matching `shpell.sh`'s literal `target="${TMUX_PANE:-}:"` form) — the trailing colon is a session-qualifier convention `shpell.sh` uses because it also targets session-scoped formats (`#{session_name}`) in the same batched call. Our calls are pane-scoped only (`#{client_name}`, `#{client_session}` are resolved from the client attached to that pane), so the plain pane ID (e.g. `%34`) is sufficient and matches every other `-t` usage in `tmux.go`.

### Decision: No change to `outerClientName()`'s client-selection loop

The loop that filters `list-clients` output to find "the other" client is unrelated to the ambiguity bug — `list-clients` always lists all server-wide clients regardless of invocation context, so no `-t` is needed there. Only the two single-value `display-message -p` lookups need pinning.

## Risks / Trade-offs

- [Risk] If `$TMUX_PANE` is unset (e.g. binary invoked outside tmux) `-t ""` would make the `display-message` call fail → Mitigation: these functions are only ever called from the dashboard TUI, which cannot run outside tmux (`InTmux()` is already checked before the dashboard starts); failure mode is unchanged from today (`err != nil` paths already exist and return early).
- [Risk] Pinning the target changes resolution for `#{client_session}` specifically when the pane's window has somehow already moved sessions between calls (e.g. a concurrent swap) → Mitigation: this window is already narrow today (implicit resolution has the same race), and pinning to the pane makes the result *more* consistent, not less.
