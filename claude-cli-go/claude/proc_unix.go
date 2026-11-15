//go:build !windows

package claude

import (
	"fmt"
	"os/exec"
	"syscall"
)

// setProcAttrs is the hook for platform process attributes. The SDK spawns the
// CLI as a plain child (no new process group), so signals sent to the host's
// process group still reach it. Keep that behaviour.
func setProcAttrs(cmd *exec.Cmd) {}

// exitSignal reports the signal that terminated the process, if any.
func exitSignal(ee *exec.ExitError) (string, bool) {
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() {
		return "", false
	}
	switch s := ws.Signal(); s {
	case syscall.SIGTERM:
		return "SIGTERM", true
	case syscall.SIGKILL:
		return "SIGKILL", true
	case syscall.SIGINT:
		return "SIGINT", true
	case syscall.SIGHUP:
		return "SIGHUP", true
	case syscall.SIGPIPE:
		return "SIGPIPE", true
	case syscall.SIGSEGV:
		return "SIGSEGV", true
	case syscall.SIGABRT:
		return "SIGABRT", true
	default:
		return fmt.Sprintf("signal %d", int(s)), true
	}
}
