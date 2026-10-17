// Package desktop sends and retracts OS-level desktop notifications.
//
// Every call that can block (notify-send waiting on D-Bus activation,
// powershell.exe cold start) runs in a detached child process with a hard
// timeout, so the Stop hook and the dashboard never wait on it.
package desktop

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"os/exec"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/bradfordwagner/tmux-claude-notify/internal/proc"
	tmuxclient "github.com/bradfordwagner/tmux-claude-notify/internal/tmux"
)

const (
	BackendOff        = "off"
	BackendNotifySend = "notify-send"
	BackendToast      = "toast"

	// timeout bounds each child; notify-send can otherwise hang ~60s+ on
	// D-Bus service activation when no notification daemon can start (WSL2).
	timeout = 10 * time.Second

	// toastAppID posts under Windows PowerShell's registered AppUserModelID,
	// which is present on every Windows install.
	toastAppID = `{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe`
	toastGroup = "claude-notify"
)

// Backend resolves @claude-notify-desktop (auto|toast|notify-send|off).
// auto picks a Windows toast under WSL (powershell.exe on PATH), else
// notify-send when installed, else off.
func Backend() string {
	switch v := tmuxclient.GlobalOption("@claude-notify-desktop"); v {
	case BackendOff, BackendNotifySend, BackendToast:
		return v
	}
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		return BackendToast
	}
	if _, err := exec.LookPath("notify-send"); err == nil {
		return BackendNotifySend
	}
	return BackendOff
}

// NotifyAsync shows a desktop notification for paneID in a detached child.
func NotifyAsync(paneID, windowName string) error {
	b := Backend()
	if b == BackendOff {
		return nil
	}
	return proc.SpawnSelf("desktop", "notify", "--backend", b, "--pane", paneID, "--window", windowName)
}

// ClearAsync retracts paneID's toast in a detached child. Only Windows toasts
// can be retracted; other backends are a no-op.
func ClearAsync(paneID string) error {
	if Backend() != BackendToast {
		return nil
	}
	return proc.SpawnSelf("desktop", "clear", "--backend", BackendToast, "--pane", paneID)
}

// Notify runs synchronously (bounded by timeout). Called from the detached child.
func Notify(backend, paneID, windowName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	switch backend {
	case BackendNotifySend:
		return exec.CommandContext(ctx, "notify-send", "claude is waiting", "Window: "+windowName).Run()
	case BackendToast:
		var body strings.Builder
		_ = xml.EscapeText(&body, []byte("Window: "+windowName))
		toastXML := `<toast><visual><binding template="ToastGeneric"><text>claude is waiting</text><text>` +
			body.String() + `</text></binding></visual></toast>`
		script := toastPrelude + `
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml(` + psQuote(toastXML) + `)
$t = [Windows.UI.Notifications.ToastNotification]::new($xml)
$t.Tag = ` + psQuote(toastTag(paneID)) + `
$t.Group = ` + psQuote(toastGroup) + `
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier(` + psQuote(toastAppID) + `).Show($t)`
		return runPowershell(ctx, script)
	}
	return nil
}

// Clear runs synchronously (bounded by timeout). Called from the detached child.
func Clear(backend, paneID string) error {
	if backend != BackendToast {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	script := toastPrelude + `
[Windows.UI.Notifications.ToastNotificationManager]::History.Remove(` +
		psQuote(toastTag(paneID)) + `, ` + psQuote(toastGroup) + `, ` + psQuote(toastAppID) + `)`
	return runPowershell(ctx, script)
}

const toastPrelude = `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null`

// toastTag identifies a pane's toast so it can be retracted later (tags are
// limited to 64 chars; pane IDs like "%34" fit easily).
func toastTag(paneID string) string {
	return "pane-" + strings.TrimPrefix(paneID, "%")
}

// psQuote returns s as a PowerShell single-quoted literal.
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// runPowershell passes script via -EncodedCommand (base64 UTF-16LE) so window
// names with quotes or shell metacharacters can't break the command line.
func runPowershell(ctx context.Context, script string) error {
	u := utf16.Encode([]rune(script))
	buf := make([]byte, len(u)*2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(buf[i*2:], c)
	}
	enc := base64.StdEncoding.EncodeToString(buf)
	return exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", enc).Run()
}
