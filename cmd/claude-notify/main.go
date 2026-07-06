package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

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
	case "auto-reset":
		paneID := ""
		delaySecs := 15
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
		}
		if paneID == "" {
			os.Exit(0)
		}
		runAutoReset(paneID, delaySecs)
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
		_ = tmuxclient.SetPopStyle(windowID)
		return nil
	}

	windowName, _ := tmuxclient.WindowName(paneID)
	session, _ := tmuxclient.Session(paneID)

	if err := tmuxclient.SetWindowStyle(windowID); err != nil {
		return err
	}
	_ = tmuxclient.SetPopStyle(windowID)

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

	// If the pane is currently focused and auto-reset is enabled, fork a detached
	// subprocess to clear the notification after the configured delay.
	if delaySecs := tmuxclient.ActiveResetSeconds(); delaySecs > 0 {
		if tmuxclient.IsPaneFocused(paneID) {
			_ = forkAutoReset(paneID, delaySecs)
		}
	}

	return nil
}

func runClear(paneID string) error {
	windowID, err := tmuxclient.WindowIDForPane(paneID)
	if err == nil {
		_ = tmuxclient.ClearWindowStyle(windowID)
		_ = tmuxclient.ClearPopStyle(windowID)
	}
	_ = tmuxclient.UnregisterClearHook(paneID)
	return store.ClearPane(paneID)
}

// runAutoReset sleeps delaySecs then clears the notification if still uncleared.
// Runs in a detached subprocess; errors are silently ignored.
func runAutoReset(paneID string, delaySecs int) {
	time.Sleep(time.Duration(delaySecs) * time.Second)
	uncleared, _ := store.HasUnclearedPane(paneID)
	if uncleared {
		_ = runClear(paneID)
	}
}

// forkAutoReset spawns a detached background process to run auto-reset.
// The parent returns immediately; the child sleeps and clears.
func forkAutoReset(paneID string, delaySecs int) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{exe, "auto-reset", "--pane", paneID, "--delay", strconv.Itoa(delaySecs)}
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
