package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/bradfordwagner/tmux-claude-notify/internal/setup"
	"github.com/bradfordwagner/tmux-claude-notify/internal/store"
	tmuxclient "github.com/bradfordwagner/tmux-claude-notify/internal/tmux"
)

var (
	accent        = lipgloss.Color("#AD8EE6")
	subtle        = lipgloss.Color("#555555")
	okColor       = lipgloss.Color("#50FA7B")
	warnColor     = lipgloss.Color("#FFB86C")
	unknColor     = lipgloss.Color("#FF5555")
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	selectedStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	dimStyle      = lipgloss.NewStyle().Foreground(subtle)
	statusOK      = lipgloss.NewStyle().Foreground(okColor)
	statusWarn    = lipgloss.NewStyle().Foreground(warnColor)
	statusUnknown = lipgloss.NewStyle().Foreground(unknColor)
)

type (
	settingsChangedMsg     struct{}
	notificationsChangedMsg struct{}
)

type entry struct {
	record store.Record
}

type model struct {
	entries      []entry
	cursor       int
	setupResult  setup.Result
	setupMessage string
	watcher      *fsnotify.Watcher
	quitting     bool
}

func newModel() (model, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return model{}, err
	}

	// Watch directory containing settings.json (handles create/write/delete).
	settingsDir := filepath.Dir(setup.SettingsPath())
	_ = watcher.Add(settingsDir)

	// Ensure log directory exists so fsnotify can watch it from first open.
	logDir := filepath.Dir(store.LogPath())
	_ = os.MkdirAll(logDir, 0o755)
	_ = watcher.Add(logDir)

	m := model{watcher: watcher}
	m.setupResult, m.setupMessage = checkAndConfigure()
	m.entries = loadEntries()
	return m, nil
}

func (m model) Init() tea.Cmd {
	return watchCmd(m.watcher)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case settingsChangedMsg:
		m.setupResult, m.setupMessage = checkAndConfigure()
		// Re-arm — wait for the next event.
		return m, watchCmd(m.watcher)

	case notificationsChangedMsg:
		m.entries = loadEntries()
		// Keep cursor in bounds after reload.
		if m.cursor >= len(m.entries) {
			m.cursor = max(0, len(m.entries)-1)
		}
		return m, watchCmd(m.watcher)

	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {
		case "q", "esc", "ctrl+c":
			m.watcher.Close()
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.entries) > 0 {
				selected := m.entries[m.cursor]
				_ = store.ClearPane(selected.record.Pane)
				_ = tmuxclient.SelectWindow(selected.record.Session, selected.record.Window)
				m.entries = loadEntries()
				if m.cursor >= len(m.entries) {
					m.cursor = max(0, len(m.entries)-1)
				}
				if len(m.entries) == 0 {
					m.watcher.Close()
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("claude-notify") + "\n")
	b.WriteString(m.renderSetupStatus() + "\n\n")

	if len(m.entries) == 0 {
		b.WriteString(dimStyle.Render("No pending notifications.") + "\n")
	} else {
		for i, e := range m.entries {
			r := e.record
			line := fmt.Sprintf("%-20s  %-12s  %-10s  %s",
				r.WindowName, r.Session, r.Pane, formatAge(r.TS))
			if i == m.cursor {
				b.WriteString(selectedStyle.Render("> "+line) + "\n")
			} else {
				b.WriteString(normalStyle.Render("  "+line) + "\n")
			}
		}
		b.WriteString("\n" + dimStyle.Render("↑/↓ or j/k to move  •  enter to switch  •  q to quit") + "\n")
	}

	return b.String()
}

func (m model) renderSetupStatus() string {
	switch m.setupResult.Status {
	case setup.StatusConfigured:
		line := statusOK.Render("✓ hook configured")
		if m.setupMessage != "" {
			line += "\n" + dimStyle.Render("  "+m.setupMessage)
		}
		return line
	case setup.StatusNotConfigured:
		return statusWarn.Render("⚠ hook not configured") + "\n" + dimStyle.Render("  "+m.setupResult.Message)
	default:
		return statusUnknown.Render("? " + m.setupResult.Message)
	}
}

// watchCmd blocks on the watcher channel and dispatches the appropriate message
// type based on which file changed.
func watchCmd(watcher *fsnotify.Watcher) tea.Cmd {
	settingsFile := filepath.Base(setup.SettingsPath())
	logFile := filepath.Base(store.LogPath())
	return func() tea.Msg {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				switch filepath.Base(event.Name) {
				case settingsFile:
					return settingsChangedMsg{}
				case logFile:
					return notificationsChangedMsg{}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

func checkAndConfigure() (setup.Result, string) {
	result := setup.Check()
	if result.Status != setup.StatusNotConfigured {
		return result, ""
	}
	msg, err := setup.Configure()
	if err != nil {
		return setup.Result{Status: setup.StatusUnknown, Message: "auto-configure failed: " + err.Error()}, ""
	}
	return setup.Result{Status: setup.StatusConfigured, Message: "Stop hook configured"}, msg
}

func loadEntries() []entry {
	records, _ := store.ReadAll()
	livePanes, _ := tmuxclient.ListLivePanes()
	liveSet := make(map[string]bool, len(livePanes))
	for _, p := range livePanes {
		liveSet[p] = true
	}
	var entries []entry
	for _, r := range records {
		if !r.Cleared && liveSet[r.Pane] {
			entries = append(entries, entry{record: r})
		}
	}
	return entries
}

func formatAge(tsNano int64) string {
	d := time.Since(time.Unix(0, tsNano))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Run() error {
	m, err := newModel()
	if err != nil {
		return err
	}
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
