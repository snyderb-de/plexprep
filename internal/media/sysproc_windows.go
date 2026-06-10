//go:build windows

package media

import (
	"os/exec"
	"syscall"
)

// noWindow stops ffprobe/ffmpeg from flashing a console window when launched
// from a GUI process (the desktop app has no console of its own).
func noWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
