## Context

The dashboard TUI (`internal/ui/model.go`) renders a permanent "✓ hook configured" green badge at the top of every frame once `~/.claude/settings.json` contains the Stop hook. This line is rendered by `renderSetupStatus()` and reflects `m.setupResult.Status`. It is visible on every render even though the fact that the hook is configured is only interesting the moment it is first set up.

Current state: `model` holds `setupResult setup.Result` and `setupMessage string`. `renderSetupStatus()` branches on `StatusConfigured` to always emit the green badge.

## Goals / Non-Goals

**Goals:**
- Remove the permanent "✓ hook configured" line from the steady-state TUI
- Show a transient toast for 10 seconds when the hook is newly wired (auto-configured on startup, or configured after a settings-change event)
- The "⚠ hook not configured" warning remains as a permanent line (still actionable and important)

**Non-Goals:**
- Toast system for any other event type (this is scoped to hook-configured only)
- Persisting the toast across dashboard restarts
- Making the 10s duration user-configurable

## Decisions

### D1: Timer via `charmbracelet/bubbles/timer`

The toast duration is managed by a `bubbles/timer.Model` embedded in the model. The timer runs for 10 seconds; when it times out it sends `timer.TimeoutMsg` which clears the toast. The timer is only started when a toast is set (not always running).

**Alternative considered**: `tea.Tick(10*time.Second, ...)` returning a custom `toastExpiredMsg`. Viable with no new dependency, but `bubbles/timer` provides a proper model with `Init`/`Update`/`View` lifecycle and is the idiomatic component for countdown use cases.

### D2: Toast stored as `string` field on model; timer as `timer.Model`

`m.toast string` holds the message; `m.toastTimer timer.Model` holds the countdown. When `toast == ""` the timer is considered inactive and its `View()` is not rendered. Timer is initialized with `timer.NewWithInterval(10*time.Second, time.Second)` so it ticks per second (could show a countdown if desired, or just be invisible).

### D3: Toast triggers only on new configuration, not on already-configured

`checkAndConfigure()` returns `(setup.Result, string)` — the second return is the action message from `setup.Configure()` (e.g. "hooks added — created ~/.claude/settings.json") and is non-empty only when the hook was actually written. Use that non-empty `setupMessage` as the toast trigger.

On startup: if `setupMessage != ""`, set `m.toast` and call `m.toastTimer.Init()` in `Init()`.  
On `settingsChangedMsg`: same logic — only set toast and restart timer if action message is non-empty.

### D4: `renderSetupStatus` collapses for configured state

When `StatusConfigured`, `renderSetupStatus()` returns `""` (empty string). The caller (`View()`) renders it inline — an empty return means no line is emitted. The "not configured" and "unknown" branches are unchanged.

## Risks / Trade-offs

- [Risk] User misses toast if they open the dashboard after 10s have passed → Mitigation: acceptable — the hook only needs to be configured once; subsequent opens show a clean UI.
- [Risk] `timer.TimeoutMsg` arrives after model is quitting → Mitigation: handler is a no-op if toast is already empty; Bubbletea drops messages after `tea.Quit`.
- [Risk] Adding `charmbracelet/bubbles` as a new dependency → Mitigation: it is the canonical Bubbletea component library maintained by the same team; low churn risk.
