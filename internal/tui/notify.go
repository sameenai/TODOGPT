package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// notifyFn is the notification backend; overridable in tests.
var notifyFn = platformNotify

// notifiedMsg is the no-op message returned after a notification fires.
type notifiedMsg struct{}

// sendNotificationCmd returns a tea.Cmd that fires a system notification and
// rings the terminal bell without blocking the TUI.
func sendNotificationCmd(title, body string) tea.Cmd {
	return func() tea.Msg {
		notifyFn(title, body)
		fmt.Fprint(os.Stderr, "\a") // terminal bell
		return notifiedMsg{}
	}
}

// platformNotify dispatches to the OS-specific notification mechanism.
// Failures are silently ignored — a missed notification is not fatal.
func platformNotify(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		exec.Command("osascript", "-e", script).Run() // #nosec G204
	case "linux":
		exec.Command("notify-send", "--urgency=normal", title, body).Run() // #nosec G204
	}
}
