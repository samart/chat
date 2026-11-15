//go:build windows

package claude

import (
	"os/exec"
	"syscall"
)

// setProcAttrs hides the console window, matching the SDK's windowsHide: true.
func setProcAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// exitSignal: Windows has no terminating signals.
func exitSignal(ee *exec.ExitError) (string, bool) { return "", false }
