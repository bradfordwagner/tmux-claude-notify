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

// HighlightColor reads @claude-notify-highlight-color, defaulting to Catppuccin Mocha green.
func HighlightColor() string {
	color, _ := run("show-option", "-gqv", "@claude-notify-highlight-color")
	if color == "" {
		color = "#a6e3a1"
	}
	return color
}

func SetWindowStyle(windowID string) error {
	color := HighlightColor()
	style := "fg=" + color + ",bold"
	if _, err := run("set-option", "-t", windowID, "window-status-style", style); err != nil {
		return err
	}
	_, err := run("set-option", "-t", windowID, "window-status-current-style", style)
	return err
}

func ClearWindowStyle(windowID string) error {
	_, _ = run("set-option", "-u", "-t", windowID, "window-status-style")
	_, err := run("set-option", "-u", "-t", windowID, "window-status-current-style")
	return err
}

// SetPopStyle sets a persistent background highlight on the specific pane via
// pane-local window-style. Uses set-option -p instead of select-pane -P to avoid
// the side effect of selecting the target pane and moving the user's focus.
// Color resolution: @claude-notify-pop-color → @tmux-pop-color → dark purple default.
func SetPopStyle(paneID string) error {
	color, _ := run("show-option", "-gqv", "@claude-notify-pop-color")
	if color == "" {
		color, _ = run("show-option", "-gqv", "@tmux-pop-color")
	}
	if color == "" || color == "black" || color == "colour0" {
		// Catppuccin Mocha base — visible against a pure-black terminal background.
		color = "#1e1e2e"
	}
	_, err := run("set-option", "-t", paneID, "-p", "window-style", "bg="+color)
	return err
}

func ClearPopStyle(paneID string) error {
	_, err := run("set-option", "-t", paneID, "-p", "-u", "window-style")
	return err
}

// IsPanePopped returns true when the pane currently has a pane-local window-style
// set (i.e. the background pop is active).
func IsPanePopped(paneID string) bool {
	out, err := run("show-options", "-t", paneID, "-p", "window-style")
	return err == nil && out != ""
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

// SelectPane makes the given pane the active pane in its window. Use before
// SelectWindow so the user lands on the right pane when the popup closes.
func SelectPane(paneID string) error {
	_, err := run("select-pane", "-t", paneID)
	return err
}

// SelectWindow sets the active window in the outer session. Use before
// DetachIfShpell so the session is on the right window when the popup closes.
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

// SwitchClientToSessionWindow switches the current client to the given session
// and activates the specified window within it. Correct for cross-session jumps;
// switch-client with a bare window ID does not reliably change the active window.
func SwitchClientToSessionWindow(session, windowID string) error {
	_, err := run("switch-client", "-t", session+":"+windowID)
	return err
}

// outerClientName returns the name of the tmux client that is NOT attached to
// _shpell-session. Used when running inside the grimoire popup to target the
// real outer terminal client.
//
// #{client_name}/#{client_session} are resolved with an explicit -t <pane>
// target rather than tmux's implicit current-client lookup, which is
// ambiguous when invoked from a background process (two clients may be
// attached at once: the outer terminal and the nested popup client). This
// mirrors the workaround tmux-grimoire's own shpell.sh applies to itself.
func outerClientName() string {
	paneID := PaneID()
	if paneID == "" {
		return ""
	}
	currentClient, err := run("display-message", "-p", "-t", paneID, "#{client_name}")
	if err != nil {
		return ""
	}
	out, err := run("list-clients", "-F", "#{client_name} #{client_session}")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 && parts[0] != currentClient && parts[1] != "_shpell-session" {
			return parts[0]
		}
	}
	return ""
}

// SwitchOuterClientToSessionWindow switches the outer tmux client (the one not
// in _shpell-session) to the given session and window. No-op when not inside
// _shpell-session or when no outer client is found.
func SwitchOuterClientToSessionWindow(session, windowID string) error {
	paneID := PaneID()
	if paneID == "" {
		return nil
	}
	currentSession, err := run("display-message", "-p", "-t", paneID, "#{client_session}")
	if err != nil || currentSession != "_shpell-session" {
		return nil
	}
	outerClient := outerClientName()
	if outerClient == "" {
		return nil
	}
	_, err = run("switch-client", "-c", outerClient, "-t", session+":"+windowID)
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

// OuterSession returns the session the real outer tmux client (the one not
// attached to _shpell-session) is actually attached to. Used to target
// split-window commands at the user's real session when the dashboard is
// running inside the grimoire popup.
//
// This resolves via outerClientName() rather than picking the first entry
// from `list-sessions`, which picked an arbitrary session whenever more than
// one non-_shpell-session session was attached (e.g. the popup was opened
// from "k8s" but list-sessions returned "edit" first).
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

// IsPaneFocused returns true when the given pane is the user's currently active
// pane (both its window is active and the pane itself is selected). Checking
// both window_active and pane_active ensures split-pane navigation triggers
// auto-reset correctly — a sibling pane in the same window does not count.
func IsPaneFocused(paneID string) bool {
	out, err := run("display-message", "-t", paneID, "-p", "#{window_active}#{pane_active}")
	return err == nil && out == "11"
}

// IsShpellOpen returns true when the grimoire shpell popup session is live,
// meaning the claude-notify dashboard is currently displayed.
func IsShpellOpen() bool {
	_, err := exec.Command("tmux", "has-session", "-t", "_shpell-session").Output()
	return err == nil
}

// ActiveResetSeconds reads @claude-notify-active-reset-seconds from global tmux
// options. Returns 15 if unset, 0 if explicitly "0" (disables auto-reset).
func ActiveResetSeconds() int {
	return readSecondsOption("@claude-notify-active-reset-seconds", 15)
}

// NavClearSeconds reads @claude-notify-nav-clear-seconds from global tmux options.
// Returns 2 if unset, 0 if explicitly "0" (disables navigate-to-clear).
func NavClearSeconds() int {
	return readSecondsOption("@claude-notify-nav-clear-seconds", 2)
}

// readSecondsOption reads a tmux global option that holds an integer number of
// seconds. Returns defaultVal if the option is unset or empty, 0 if set to "0",
// defaultVal if the value is not a valid non-negative integer.
func readSecondsOption(option string, defaultVal int) int {
	val, _ := run("show-option", "-gqv", option)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	if val == "0" {
		return 0
	}
	// strconv.Atoi failures fall back to default
	n := 0
	for _, c := range val {
		if c < '0' || c > '9' {
			return defaultVal
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func RefreshStatusBar() {
	_ = exec.Command("tmux", "refresh-client", "-S").Run()
}

func NotifySend(windowName string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return nil
	}
	return exec.Command("notify-send", "claude is waiting", "Window: "+windowName).Run()
}
