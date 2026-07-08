package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/bradfordwagner/tmux-claude-notify/internal/resurrect"
	"github.com/bradfordwagner/tmux-claude-notify/internal/store"
	tmuxclient "github.com/bradfordwagner/tmux-claude-notify/internal/tmux"
	"github.com/bradfordwagner/tmux-claude-notify/internal/ui"
)

func main() {
	if len(os.Args) < 2 {
		if err := ui.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "status":
		runStatus()
	case "jump":
		if err := runJump(); err != nil {
			_ = err
		}
	case "notify":
		if err := runNotify(); err != nil {
			// Exit 0 — hook errors must not surface noisily in tmux
			_ = err
		}
	case "clear":
		paneID := ""
		for i, arg := range os.Args[2:] {
			if arg == "--pane" && i+1 < len(os.Args[2:]) {
				paneID = os.Args[i+3]
			}
		}
		if paneID == "" {
			os.Exit(0)
		}
		if err := runClear(paneID); err != nil {
			_ = err
		}
	case "resurrect":
		if len(os.Args) >= 3 {
			switch os.Args[2] {
			case "save":
				_ = resurrect.Save()
			case "restore":
				_ = resurrect.Restore()
			}
		}
	case "auto-reset":
		paneID := ""
		delaySecs := 15
		navDelaySecs := 2
		args := os.Args[2:]
		for i, arg := range args {
			if arg == "--pane" && i+1 < len(args) {
				paneID = args[i+1]
			}
			if arg == "--delay" && i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					delaySecs = n
				}
			}
			if arg == "--nav-delay" && i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					navDelaySecs = n
				}
			}
		}
		if paneID == "" {
			os.Exit(0)
		}
		runAutoReset(paneID, delaySecs, navDelaySecs)
	default:
		if err := ui.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func runNotify() error {
	if !tmuxclient.InTmux() {
		return nil
	}
	paneID := tmuxclient.PaneID()
	if paneID == "" {
		return nil
	}

	windowID, err := tmuxclient.WindowID(paneID)
	if err != nil {
		return err
	}

	// If a notification already exists for this pane, just re-apply styles and return.
	// Stop and PreToolUse both fire per-segment; idempotency prevents duplicate entries.
	already, _ := store.HasUnclearedPane(paneID)
	if already {
		_ = tmuxclient.SetWindowStyle(windowID)
		_ = tmuxclient.SetPopStyle(paneID)
		return nil
	}

	windowName, _ := tmuxclient.WindowName(paneID)
	session, _ := tmuxclient.Session(paneID)

	if err := tmuxclient.SetWindowStyle(windowID); err != nil {
		return err
	}
	_ = tmuxclient.SetPopStyle(paneID)

	if err := tmuxclient.NotifySend(windowName); err != nil {
		_ = err
	}

	if err := store.Append(store.Record{
		TS:         store.NowNano(),
		Pane:       paneID,
		Window:     windowID,
		WindowName: windowName,
		Session:    session,
		Cleared:    false,
	}); err != nil {
		return err
	}

	// Fork a detached subprocess to auto-clear the notification.
	// If the pane is already focused, it clears after delaySecs (active-reset grace period).
	// If not focused, it polls until the user navigates to the pane, then clears after navDelaySecs.
	delaySecs := tmuxclient.ActiveResetSeconds()
	navDelaySecs := tmuxclient.NavClearSeconds()
	if delaySecs > 0 || navDelaySecs > 0 {
		_ = forkAutoReset(paneID, delaySecs, navDelaySecs)
	}

	return nil
}

func runClear(paneID string) error {
	// Resolve window ID from JSONL first (reliable even when pane no longer exists),
	// falling back to a live tmux query.
	windowID, _ := store.WindowForPane(paneID)
	if windowID == "" {
		windowID, _ = tmuxclient.WindowIDForPane(paneID)
	}

	// Pane pop is per-pane — always clear it for this specific pane.
	_ = tmuxclient.ClearPopStyle(paneID)
	_ = tmuxclient.UnregisterClearHook(paneID)
	if err := store.ClearPane(paneID); err != nil {
		return err
	}

	clearWindowIfEmpty(windowID)
	return nil
}

// clearWindowIfEmpty tears down window-level tab highlight once no sibling
// panes in that window still have uncleared notifications.
func clearWindowIfEmpty(windowID string) {
	if windowID == "" {
		return
	}
	remaining, _ := store.UnclearedForWindow(windowID)
	if len(remaining) == 0 {
		_ = tmuxclient.ClearWindowStyle(windowID)
	}
}

// runAutoReset clears the notification once the user's window becomes focused.
// delaySecs governs the already-focused path; navDelaySecs governs the poll-detected path.
// Both paths call clearAfterGracePeriod so the grace-period invariant is enforced
// in one place — adding a new path must call it too.
func runAutoReset(paneID string, delaySecs, navDelaySecs int) {
	if tmuxclient.IsPaneFocused(paneID) {
		if delaySecs > 0 {
			clearAfterGracePeriod(paneID, delaySecs)
		}
		return
	}
	const maxPoll = 4 * time.Hour
	const pollInterval = 2 * time.Second
	start := time.Now()
	for time.Since(start) < maxPoll {
		time.Sleep(pollInterval)
		if uncleared, _ := store.HasUnclearedPane(paneID); !uncleared {
			return
		}
		if tmuxclient.IsShpellOpen() {
			continue
		}
		if tmuxclient.IsPaneFocused(paneID) {
			if navDelaySecs > 0 {
				clearAfterGracePeriod(paneID, navDelaySecs)
			}
			return
		}
	}
}

// clearAfterGracePeriod sleeps delaySecs then clears the notification if it is
// still uncleared and the dashboard is not open. All auto-reset paths must call
// this — never inline the sleep+clear logic in a new branch.
func clearAfterGracePeriod(paneID string, delaySecs int) {
	time.Sleep(time.Duration(delaySecs) * time.Second)
	if !tmuxclient.IsShpellOpen() {
		if uncleared, _ := store.HasUnclearedPane(paneID); uncleared {
			_ = runClear(paneID)
		}
	}
}

func runStatus() {
	records, err := store.ReadAll()
	if err != nil {
		return
	}
	count := 0
	for _, r := range records {
		if !r.Cleared {
			count++
		}
	}
	if count > 0 {
		bg := tmuxclient.HighlightColor()
		fmt.Printf("#[fg=#000000,bg=%s,bold] ⚡ %d #[default]", bg, count)
	}
}

// runJump finds and clears the oldest uncleared notification as one atomic
// store operation (store.ClearOldestUncleared), so a concurrent writer (the
// Stop hook, an auto-reset subprocess, or another jump invocation) can never
// observe or revert the clear between "find oldest" and "mark cleared" — the
// gap that let jump repeatedly re-select the same pane.
func runJump() error {
	r, err := store.ClearOldestUncleared()
	if r == nil {
		return err
	}
	_ = tmuxclient.SelectPane(r.Pane)
	_ = tmuxclient.DetachIfShpell()
	_ = tmuxclient.SwitchClientToSessionWindow(r.Session, r.Window)
	_ = tmuxclient.ClearPopStyle(r.Pane)
	_ = tmuxclient.UnregisterClearHook(r.Pane)
	clearWindowIfEmpty(r.Window)
	// Refresh even if the store write above failed — the badge should reflect
	// whatever the store actually contains, not stale pre-jump state.
	tmuxclient.RefreshStatusBar()
	return err
}

// forkAutoReset spawns a detached background process to run auto-reset.
// The parent returns immediately; the child sleeps and clears.
func forkAutoReset(paneID string, delaySecs, navDelaySecs int) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{
		exe, "auto-reset",
		"--pane", paneID,
		"--delay", strconv.Itoa(delaySecs),
		"--nav-delay", strconv.Itoa(navDelaySecs),
	}
	proc, err := os.StartProcess(exe, args, &os.ProcAttr{
		Files: []*os.File{nil, nil, nil},
		Sys:   &syscall.SysProcAttr{Setsid: true},
	})
	if err != nil {
		return err
	}
	// Release so the parent doesn't accumulate a zombie entry.
	return proc.Release()
}
