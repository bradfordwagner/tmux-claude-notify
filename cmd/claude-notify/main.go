package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bradfordwagner/tmux-claude-notify/internal/store"
	tmuxclient "github.com/bradfordwagner/tmux-claude-notify/internal/tmux"
	"github.com/bradfordwagner/tmux-claude-notify/internal/ui"
)

func debugLog(msg string) {
	f, err := os.OpenFile(os.ExpandEnv("$HOME/.local/share/tmux-claude-notify/debug.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("15:04:05.000"), msg)
}

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
	default:
		if err := ui.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func runNotify() error {
	if !tmuxclient.InTmux() {
		debugLog("runNotify: not in tmux, skipping")
		return nil
	}
	paneID := tmuxclient.PaneID()
	if paneID == "" {
		debugLog("runNotify: no TMUX_PANE, skipping")
		return nil
	}

	windowID, err := tmuxclient.WindowID(paneID)
	if err != nil {
		debugLog(fmt.Sprintf("runNotify: WindowID error for pane %s: %v", paneID, err))
		return err
	}

	// If a notification already exists for this pane, just re-apply styles and return.
	// Claude Code fires Stop once per AI response segment, so rapid successive calls
	// (e.g. during /cc) must not clear-and-recreate the entry each time.
	already, _ := store.HasUnclearedPane(paneID)
	debugLog(fmt.Sprintf("runNotify: pane=%s window=%s already=%v", paneID, windowID, already))
	if already {
		_ = tmuxclient.SetWindowStyle(windowID)
		_ = tmuxclient.SetPopStyle(windowID)
		return nil
	}

	windowName, _ := tmuxclient.WindowName(paneID)
	session, _ := tmuxclient.Session(paneID)

	if err := tmuxclient.SetWindowStyle(windowID); err != nil {
		debugLog(fmt.Sprintf("runNotify: SetWindowStyle error: %v", err))
		return err
	}
	_ = tmuxclient.SetPopStyle(windowID)

	if err := tmuxclient.NotifySend(windowName); err != nil {
		_ = err
	}

	debugLog(fmt.Sprintf("runNotify: appending new entry pane=%s window=%s session=%s", paneID, windowID, session))
	return store.Append(store.Record{
		TS:         store.NowNano(),
		Pane:       paneID,
		Window:     windowID,
		WindowName: windowName,
		Session:    session,
		Cleared:    false,
	})
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
