package main

import (
	"fmt"
	"os"

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
