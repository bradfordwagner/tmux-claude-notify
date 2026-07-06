package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func run(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).Output()
	return strings.TrimSpace(string(out)), err
}

func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

func PaneID() string {
	return os.Getenv("TMUX_PANE")
}

func WindowID(paneID string) (string, error) {
	return run("display-message", "-t", paneID, "-p", "#{window_id}")
}

// WindowIDForPane resolves window ID when called outside the original pane env
// (e.g. from the pane-focus-in hook where TMUX_PANE may differ).
func WindowIDForPane(paneID string) (string, error) {
	return run("display-message", "-t", paneID, "-p", "#{window_id}")
}

func WindowName(paneID string) (string, error) {
	return run("display-message", "-t", paneID, "-p", "#{window_name}")
}

func PanePath(paneID string) (string, error) {
	return run("display-message", "-t", paneID, "-p", "#{pane_current_path}")
}

func Session(paneID string) (string, error) {
	return run("display-message", "-t", paneID, "-p", "#{session_name}")
}

func SetWindowStyle(windowID string) error {
	if _, err := run("set-option", "-t", windowID, "window-status-style", "fg=#AD8EE6,bold"); err != nil {
		return err
	}
	_, err := run("set-option", "-t", windowID, "window-status-current-style", "fg=#AD8EE6,bold")
	return err
}

func ClearWindowStyle(windowID string) error {
	_, _ = run("set-option", "-u", "-t", windowID, "window-status-style")
	_, err := run("set-option", "-u", "-t", windowID, "window-status-current-style")
	return err
}

// SetPopStyle sets a persistent pane background highlight. Reads @claude-notify-pop-color
// first, then falls back to @tmux-pop-color, then a dark purple default. Stays until ClearPopStyle.
func SetPopStyle(windowID string) error {
	color, _ := run("show-option", "-gqv", "@claude-notify-pop-color")
	if color == "" {
		color, _ = run("show-option", "-gqv", "@tmux-pop-color")
	}
	if color == "" || color == "black" || color == "colour0" {
		// Catppuccin Mocha base — visible against a pure-black terminal background.
		color = "#1e1e2e"
	}
	_, err := run("set-option", "-t", windowID, "window-active-style", "bg="+color)
	return err
}

func ClearPopStyle(windowID string) error {
	_, err := run("set-option", "-u", "-t", windowID, "window-active-style")
	return err
}

// RegisterClearHook registers a global after-select-window hook that clears the
// notification when the user switches to the notified window. pane-focus-in is
// not used because it requires terminal focus event reporting (broken in WSL2).
func RegisterClearHook(paneID, windowID, binaryPath string) error {
	idx := strings.TrimPrefix(paneID, "%")
	hookCmd := fmt.Sprintf("if-shell -F '#{==:#{window_id},%s}' 'run-shell \"%s clear --pane %s\"'",
		windowID, binaryPath, paneID)
	_, err := run("set-hook", "-g", "-a", fmt.Sprintf("after-select-window[%s]", idx), hookCmd)
	return err
}

func UnregisterClearHook(paneID string) error {
	idx := strings.TrimPrefix(paneID, "%")
	_, err := run("set-hook", "-g", "-u", fmt.Sprintf("after-select-window[%s]", idx))
	return err
}

// SelectWindow sets the active window in a session without switching any client.
// Use this before DetachIfShpell so the outer session is on the right window when
// the popup closes.
func SelectWindow(session, windowID string) error {
	_, err := run("select-window", "-t", session+":"+windowID)
	return err
}

// SwitchToWindow switches the current client to a window. Use only when not inside
// a grimoire shpell (popup fallback path).
func SwitchToWindow(windowID string) error {
	_, err := run("switch-client", "-t", windowID)
	return err
}

func ListLivePanes() ([]string, error) {
	out, err := run("list-panes", "-a", "-F", "#{pane_id}")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// DetachIfShpell calls detach-client when running inside grimoire's _shpell-session,
// which triggers grimoire's cleanup hook and closes the popup.
func DetachIfShpell() error {
	session, err := run("display-message", "-p", "#{client_session}")
	if err != nil {
		return err
	}
	if session == "_shpell-session" {
		_, err = run("detach-client")
		return err
	}
	return nil
}

// IsPaneFocused returns true only when the given pane is the active pane in the
// currently active window — i.e. the user is looking at it right now.
func IsPaneFocused(paneID string) bool {
	out, err := run("display-message", "-t", paneID, "-p", "#{pane_active}#{window_active}")
	return err == nil && out == "11"
}

// ActiveResetSeconds reads @claude-notify-active-reset-seconds from global tmux
// options. Returns 15 if unset, 0 if explicitly "0" (disables auto-reset).
func ActiveResetSeconds() int {
	val, _ := run("show-option", "-gqv", "@claude-notify-active-reset-seconds")
	val = strings.TrimSpace(val)
	if val == "" {
		return 15
	}
	if val == "0" {
		return 0
	}
	// strconv.Atoi failures fall back to default
	n := 0
	for _, c := range val {
		if c < '0' || c > '9' {
			return 15
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func NotifySend(windowName string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return nil
	}
	return exec.Command("notify-send", "claude is waiting", "Window: "+windowName).Run()
}
