## 1. Dependency

- [ ] 1.1 Add `github.com/charmbracelet/bubbles` to `go.mod` via `go get github.com/charmbracelet/bubbles/timer`

## 2. Model Changes

- [ ] 2.1 Add `toast string` and `toastTimer timer.Model` fields to `model` struct in `internal/ui/model.go`
- [ ] 2.2 Import `github.com/charmbracelet/bubbles/timer`

## 3. Toast Trigger

- [ ] 3.1 In `newModel()`, after calling `checkAndConfigure()`, if `setupMessage != ""` set `m.toast = setupMessage` and `m.toastTimer = timer.NewWithInterval(10*time.Second, time.Second)`
- [ ] 3.2 In `Update()` `settingsChangedMsg` branch, after calling `checkAndConfigure()`, if `setupMessage != ""` set `m.toast = setupMessage`, reset `m.toastTimer`, and include `m.toastTimer.Init()` in the returned batch

## 4. Timer Lifecycle

- [ ] 4.1 In `Init()`, if `m.toast != ""` include `m.toastTimer.Init()` in the initial `tea.Batch`
- [ ] 4.2 In `Update()`, forward all `timer.TickMsg` and `timer.TimeoutMsg` to `m.toastTimer.Update(msg)` and collect its cmd
- [ ] 4.3 On `timer.TimeoutMsg`: set `m.toast = ""`

## 5. View Changes

- [ ] 5.1 In `renderSetupStatus()`, return `""` for the `StatusConfigured` case (remove permanent badge)
- [ ] 5.2 In `View()`, after `renderSetupStatus()`, if `m.toast != ""` render one line using `statusOK` style with "✓ " prefix and the toast text

## 6. Verify

- [ ] 6.1 Build the binary (`task build`) and confirm no compile errors
- [ ] 6.2 Run the dashboard when hook is already configured — confirm no permanent "✓ hook configured" line appears
- [ ] 6.3 Simulate first-run by removing the Stop hook from settings.json, run dashboard, confirm toast appears for ~10s then disappears
