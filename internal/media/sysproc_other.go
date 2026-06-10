//go:build !windows

package media

import "os/exec"

// noWindow is a no-op on non-Windows platforms.
func noWindow(cmd *exec.Cmd) {}
