package camera

import (
	"os/exec"
	"syscall"
)

// hideWindow keeps ffmpeg from flashing a console window when a GUI program starts it.
func hideWindow(oCmd *exec.Cmd) {
	oCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
